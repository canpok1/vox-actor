package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

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
	DirWatcherFactory func(logger *slog.Logger) app.DirWatcher
	// QueuePathResolver は --queue 指定時に監視対象パスを解決する関数。
	// nil の場合は infra.ResolveQueuePath がデフォルトで使われる。
	QueuePathResolver func() (string, error)
	// LockPathResolver は viewer ロックファイルのパスを解決する関数。
	// nil の場合は infra.ResolveHomeViewerLockPath がデフォルトで使われる。
	LockPathResolver func() (string, error)
	// AudioProbe は起動時に音声デバイス可用性を検査する関数。
	// nil の場合は検査をスキップする。--dry-run 時は呼ばれない。
	AudioProbe func() error
}

func makeWatchCmd(deps *WatchDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "watch <dir1> [<dir2> ...]",
		Short: "複数ディレクトリを監視してテキストファイルを読み上げる",
		Long:  "1つ以上のディレクトリを並列監視し、検知したファイルを順次読み上げる。",
		// 位置引数と --queue の排他・片方必須は RunE 側で検証する。
		Args: cobra.ArbitraryArgs,
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			if cmd.Flags().Changed("stream") {
				return fmt.Errorf("%w: --stream is removed; use 'vox-actor viewer' instead", ErrUsage)
			}
			if cmd.Flags().Changed("stream-addr") {
				return fmt.Errorf("%w: --stream-addr is removed; use 'vox-actor viewer --host <host> --port <port>' instead", ErrUsage)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWatch(cmd, args, deps)
		},
	}

	registerCommonFlags(cmd)
	registerSaveWavDirFlag(cmd)
	cmd.Flags().Bool("delete", false, "処理済みファイルを削除する（未指定時は各ディレクトリの done/ に移動）")
	cmd.Flags().Bool("queue", false, "vox-actor config path.queue で解決される queue ディレクトリを監視対象に自動選択する（位置引数と併用不可、起動時に自動作成）")

	// --stream / --stream-addr は廃止済み。廃止フラグが渡された場合は PreRunE でエラーを返す。
	cmd.Flags().Bool("stream", false, "")
	cmd.Flags().String("stream-addr", "", "")
	_ = cmd.Flags().MarkHidden("stream")
	_ = cmd.Flags().MarkHidden("stream-addr")

	return cmd
}

func resolveWatchLockPath(deps *WatchDeps) (string, error) {
	if deps != nil {
		return resolveViewerLockPathWith(deps.LockPathResolver)
	}
	return resolveViewerLockPathWith(nil)
}

// viewerWatchPlayer は watch 時に viewer へ音声再生リクエストを転送するプレイヤー。
// app.AudioPlayer と textPlayer の両インターフェースを実装する。
type viewerWatchPlayer struct {
	vc         *infra.ViewerAPIClient
	speed      *float64
	pitch      *float64
	intonation *float64
}

func (p *viewerWatchPlayer) Play(ctx context.Context, _ []byte, meta app.PlayMeta) error {
	return p.playViaViewer(ctx, meta)
}

func (p *viewerWatchPlayer) PlayText(ctx context.Context, meta app.PlayMeta) error {
	return p.playViaViewer(ctx, meta)
}

func (p *viewerWatchPlayer) playViaViewer(ctx context.Context, meta app.PlayMeta) error {
	_, err := p.vc.Play(ctx, infra.ViewerPlayRequest{
		Clips: []infra.ViewerClip{{
			Text:       meta.Text,
			SpeakerID:  meta.SpeakerID.Value(),
			Speed:      p.speed,
			Pitch:      p.pitch,
			Intonation: p.intonation,
		}},
	})
	return err
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

	var resolver func() (string, error)
	if deps != nil {
		resolver = deps.QueuePathResolver
	}
	queuePath, err := resolveQueueDir(resolver)
	if err != nil {
		return nil, err
	}
	return []string{queuePath}, nil
}

// resolveQueueDir はキュー解決関数を使ってキューディレクトリパスを解決・自動作成する。
// resolver が nil の場合は infra.ResolveQueuePath を使う。
func resolveQueueDir(resolver func() (string, error)) (string, error) {
	if resolver == nil {
		resolver = infra.ResolveQueuePath
	}
	queuePath, err := resolver()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(queuePath, 0o755); err != nil {
		return "", fmt.Errorf("failed to create queue directory %s: %w", queuePath, err)
	}
	return queuePath, nil
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
	speakerIDInt, _ := cmd.Flags().GetInt("speaker")
	speed, _ := cmd.Flags().GetFloat64("speed")
	pitch, _ := cmd.Flags().GetFloat64("pitch")
	intonation, _ := cmd.Flags().GetFloat64("intonation")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	saveWavDir, _ := cmd.Flags().GetString("save-wav-dir")

	speakerID, err := entity.NewSpeakerID(speakerIDInt)
	if err != nil {
		return fmt.Errorf("%w: --speaker: %v", ErrUsage, err)
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

	watcher := deps.DirWatcherFactory(logger)
	opts := []app.WatchOption{app.WithWatchLogger(logger)}
	if deleteMode {
		opts = append(opts, app.WithDeleteMode())
	}

	// viewer 優先順位: 明示 URL > lockfile auto-detect > ローカル再生
	var vwPlayer *viewerWatchPlayer
	if !dryRun {
		viewerURL, _ := cmd.Flags().GetString("viewer-url")
		if viewerURL != "" {
			// 明示 URL: lockfile スキップ、AudioProbe スキップ
			logger.Info("using explicit viewer URL for watch", "url", viewerURL)
			vwPlayer = &viewerWatchPlayer{
				vc:         infra.NewViewerAPIClientFromURL(viewerURL),
				speed:      &speed,
				pitch:      &pitch,
				intonation: &intonation,
			}
		} else {
			// lockfile auto-detect
			lockPath, lockErr := resolveWatchLockPath(deps)
			if lockErr != nil {
				logger.Debug("could not resolve viewer lock path", "error", lockErr)
			} else if addr, ok := infra.DetectViewer(lockPath); ok {
				logger.Info("viewer detected, will use viewer for watch playback", "addr", addr)
				vwPlayer = &viewerWatchPlayer{
					vc:         infra.NewViewerAPIClient(addr),
					speed:      &speed,
					pitch:      &pitch,
					intonation: &intonation,
				}
			}
		}
	}

	if vwPlayer != nil {
		// viewer 使用時: AudioProbe スキップ、WithSilent で VOICEVOX 合成もスキップ
		opts = append(opts, app.WithSilent())
		uc := app.NewWatchUsecase(deps.Reader, client, vwPlayer, deps.Mover, watcher, opts...)
		return uc.Run(ctx, app.WatchParams{
			Paths:      args,
			SpeakerID:  speakerID,
			Speed:      &speed,
			Pitch:      &pitch,
			Intonation: &intonation,
			DryRun:     dryRun,
		})
	}

	if err := runAudioProbe(deps.AudioProbe, dryRun); err != nil {
		return err
	}

	if saveWavDir != "" {
		opts = append(opts, app.WithWatchWavSaver(infra.NewWavSaver()))
	}

	uc := app.NewWatchUsecase(deps.Reader, client, deps.Player, deps.Mover, watcher, opts...)
	return uc.Run(ctx, app.WatchParams{
		Paths:      args,
		SpeakerID:  speakerID,
		Speed:      &speed,
		Pitch:      &pitch,
		Intonation: &intonation,
		DryRun:     dryRun,
		SaveWavDir: saveWavDir,
	})
}
