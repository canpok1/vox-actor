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
}

func makeWatchCmd(deps *WatchDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "watch <dir1> [<dir2> ...]",
		Short: "複数ディレクトリを監視してテキストファイルを読み上げる",
		Long:  "1つ以上のディレクトリを並列監視し、検知したファイルを順次読み上げる。",
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.MinimumNArgs(1)(cmd, args); err != nil {
				return fmt.Errorf("%w: %s", ErrUsage, err)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWatch(cmd, args, deps)
		},
	}

	registerCommonFlags(cmd)
	cmd.Flags().Bool("delete", false, "処理済みファイルを削除する（未指定時は各ディレクトリの done/ に移動）")
	cmd.Flags().Bool("stream", false, "HTTPサーバーを起動し、SSE経由でブラウザに音声配信する")
	cmd.Flags().String("stream-addr", "127.0.0.1:8080", "ストリーム配信用のバインドアドレス")

	return cmd
}

func runWatch(cmd *cobra.Command, args []string, deps *WatchDeps) error {
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
