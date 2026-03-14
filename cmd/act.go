package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/canpok1/vox-actor/internal/app"
	"github.com/spf13/cobra"
)

// ActDeps はactコマンドの依存を保持する。
type ActDeps struct {
	Reader            app.ScriptReader
	ClientFactory     func(engineURL string) app.VoicevoxClient
	Player            app.AudioPlayer
	Mover             app.FileMover
	DirWatcherFactory func() app.DirWatcher
}

func makeActCmd(deps *ActDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "act <path>",
		Short: "テキストファイルを読み上げる",
		Long:  "テキストファイルを読み込み、VOICEVOXエンジンで音声合成して再生する。",
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.ExactArgs(1)(cmd, args); err != nil {
				return fmt.Errorf("%w: %s", ErrUsage, err)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAct(cmd, args, deps)
		},
	}

	cmd.Flags().String("engine-url", "http://localhost:50021", "VOICEVOXエンジンのURL")
	cmd.Flags().Int("speaker", 3, "キャラクターID")
	cmd.Flags().Float64("speed", 1.0, "話速")
	cmd.Flags().Float64("pitch", 0.0, "音高")
	cmd.Flags().Float64("intonation", 1.0, "抑揚")
	cmd.Flags().Bool("watch", false, "ディレクトリ監視モードを有効化")
	cmd.Flags().Bool("verbose", false, "詳細ログを出力")

	return cmd
}

func runAct(cmd *cobra.Command, args []string, deps *ActDeps) error {
	watch, _ := cmd.Flags().GetBool("watch")
	if watch {
		path := args[0]
		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrUsage, err)
		}
		if !info.IsDir() {
			return fmt.Errorf("%w: --watch requires a directory path, not a file", ErrUsage)
		}
	}

	if deps == nil || deps.ClientFactory == nil || deps.Reader == nil || deps.Player == nil {
		return fmt.Errorf("act command dependencies are not initialized")
	}

	// シグナルハンドリング: SIGINT/SIGTERM受信時にcontextをキャンセルする
	ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	verbose, _ := cmd.Flags().GetBool("verbose")
	engineURL, _ := cmd.Flags().GetString("engine-url")
	speakerID, _ := cmd.Flags().GetInt("speaker")
	speed, _ := cmd.Flags().GetFloat64("speed")
	pitch, _ := cmd.Flags().GetFloat64("pitch")
	intonation, _ := cmd.Flags().GetFloat64("intonation")

	// ログレベルの設定: --verbose 指定時は Debug、通常時は Info
	logLevel := slog.LevelInfo
	if verbose {
		logLevel = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel}))

	client := deps.ClientFactory(engineURL)
	if client == nil {
		return fmt.Errorf("failed to create VoicevoxClient for %s", engineURL)
	}

	params := app.ActParams{
		Path:       args[0],
		SpeakerID:  speakerID,
		Speed:      &speed,
		Pitch:      &pitch,
		Intonation: &intonation,
	}

	if watch {
		if deps.Mover == nil || deps.DirWatcherFactory == nil {
			return fmt.Errorf("watch mode dependencies are not initialized")
		}
		watcher := deps.DirWatcherFactory()
		uc := app.NewWatchUsecase(deps.Reader, client, deps.Player, deps.Mover, watcher, app.WithWatchLogger(logger))
		return uc.Run(ctx, params)
	}

	uc := app.NewActUsecase(deps.Reader, client, deps.Player, app.WithLogger(logger))
	return uc.Run(ctx, params)
}
