package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/canpok1/vox-actor/internal/app"
	"github.com/canpok1/vox-actor/internal/domain/entity"
	"github.com/canpok1/vox-actor/internal/infra"
	"github.com/spf13/cobra"
)

// WatchDeps はwatchコマンドの依存を保持する。
type WatchDeps struct {
	Reader            app.ScriptReader
	ClientFactory     func(engineURL string) app.VoicevoxClient
	Player            app.AudioPlayer
	Mover             app.FileMover
	DirWatcherFactory func() app.DirWatcher
	// StreamPlayerFactory は --stream 指定時にストリーム配信用の AudioPlayer を生成する。
	// speakerLookup は /speakers 取得結果から構築されたマップで、配信時に話者名/スタイル名を解決する。
	// マップが空でも factory はエラーにせず player を返してよい（フォールバック表示が使われる）。
	// client はブラウザ側の /test-clip エンドポイントから音声合成を呼ぶために渡す。
	StreamPlayerFactory func(addr string, logger *slog.Logger, speakerLookup map[int]entity.SpeakerStyleInfo, client app.VoicevoxClient) (app.StreamPlayer, error)
	// QueuePathResolver は --queue 指定時に監視対象パスを解決する関数。
	// nil の場合は infra.ResolveQueuePath がデフォルトで使われる。
	QueuePathResolver func() (string, error)
}

func makeWatchCmd(deps *WatchDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "watch <dir1> [<dir2> ...]",
		Short: "複数ディレクトリを監視してテキストファイルを読み上げる",
		Long:  "1つ以上のディレクトリを並列監視し、検知したファイルを順次読み上げる。",
		// 位置引数と --queue の排他・片方必須は RunE 側で検証する。
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWatch(cmd, args, deps)
		},
	}

	registerCommonFlags(cmd)
	cmd.Flags().Bool("delete", false, "処理済みファイルを削除する（未指定時は各ディレクトリの done/ に移動）")
	cmd.Flags().Bool("stream", false, "HTTPサーバーを起動し、SSE経由でブラウザに音声配信する")
	cmd.Flags().String("stream-addr", "127.0.0.1:8080", "ストリーム配信用のバインドアドレス")
	cmd.Flags().Bool("queue", false, "vox-actor config path.queue で解決される queue ディレクトリを監視対象に自動選択する（位置引数と併用不可、起動時に自動作成）")

	return cmd
}

// resolveWatchPaths は watch の監視対象パスを確定する。
// --queue が指定されていればキュー解決・自動作成を行い、
// それ以外は与えられた位置引数をそのまま返す。
func resolveWatchPaths(args []string, useQueue bool, deps *WatchDeps) ([]string, error) {
	switch {
	case useQueue && len(args) > 0:
		return nil, fmt.Errorf("%w: --queue と位置引数は併用できません", ErrUsage)
	case !useQueue && len(args) == 0:
		return nil, fmt.Errorf("%w: requires at least 1 arg(s), only received 0", ErrUsage)
	}

	if !useQueue {
		return args, nil
	}

	resolver := infra.ResolveQueuePath
	if deps != nil && deps.QueuePathResolver != nil {
		resolver = deps.QueuePathResolver
	}
	queuePath, err := resolver()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(queuePath, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create queue directory %s: %w", queuePath, err)
	}
	return []string{queuePath}, nil
}

func runWatch(cmd *cobra.Command, args []string, deps *WatchDeps) error {
	useQueue, _ := cmd.Flags().GetBool("queue")

	args, err := resolveWatchPaths(args, useQueue, deps)
	if err != nil {
		return err
	}

	for _, path := range args {
		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrUsage, err)
		}
		if !info.IsDir() {
			return fmt.Errorf("%w: %s is not a directory", ErrUsage, path)
		}
	}

	deleteMode, _ := cmd.Flags().GetBool("delete")
	engineURL, _ := cmd.Flags().GetString("engine-url")
	speakerID, _ := cmd.Flags().GetInt("speaker")
	speed, _ := cmd.Flags().GetFloat64("speed")
	pitch, _ := cmd.Flags().GetFloat64("pitch")
	intonation, _ := cmd.Flags().GetFloat64("intonation")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	stream, _ := cmd.Flags().GetBool("stream")
	streamAddr, _ := cmd.Flags().GetString("stream-addr")

	if stream && dryRun {
		return fmt.Errorf("%w: --stream and --dry-run cannot be used together", ErrUsage)
	}

	if deps == nil || deps.ClientFactory == nil || deps.Reader == nil || deps.Player == nil || deps.Mover == nil || deps.DirWatcherFactory == nil {
		return fmt.Errorf("watch command dependencies are not initialized")
	}

	ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger := buildLoggerFromFlags(cmd)

	client := deps.ClientFactory(engineURL)
	if client == nil {
		return fmt.Errorf("failed to create VoicevoxClient for %s", engineURL)
	}

	player := deps.Player
	if stream {
		if deps.StreamPlayerFactory == nil {
			return fmt.Errorf("stream player factory is not initialized")
		}

		// HealthCheck → /speakers 取得 → lookup を組み立ててから stream player を起動する。
		// /speakers の失敗は watch 起動エラーとして即終了する。
		if err := client.HealthCheck(ctx); err != nil {
			return fmt.Errorf("engine health check failed: %w", err)
		}
		speakers, err := client.GetSpeakers(ctx)
		if err != nil {
			return fmt.Errorf("failed to get speakers: %w", err)
		}
		lookup := entity.BuildSpeakerStyleLookup(speakers)
		logger.Info("speakers loaded", "speakerCount", len(speakers), "styleCount", len(lookup))

		sp, err := deps.StreamPlayerFactory(streamAddr, logger, lookup, client)
		if err != nil {
			return fmt.Errorf("failed to create stream player: %w", err)
		}
		if err := sp.Start(ctx); err != nil {
			return fmt.Errorf("failed to start stream server: %w", err)
		}
		defer func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := sp.Shutdown(shutdownCtx); err != nil {
				logger.Error("stream server shutdown error", "error", err)
			}
		}()
		logger.Info("stream server listening", "addr", sp.Addr())
		player = sp
	}

	watcher := deps.DirWatcherFactory()
	opts := []app.WatchOption{app.WithWatchLogger(logger)}
	if deleteMode {
		opts = append(opts, app.WithDeleteMode())
	}

	uc := app.NewWatchUsecase(deps.Reader, client, player, deps.Mover, watcher, opts...)
	return uc.Run(ctx, app.WatchParams{
		Paths:      args,
		SpeakerID:  speakerID,
		Speed:      &speed,
		Pitch:      &pitch,
		Intonation: &intonation,
		DryRun:     dryRun,
	})
}
