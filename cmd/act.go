package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"
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

	engineURLDefault := "http://localhost:50021"
	if v := os.Getenv("VOX_ENGINE_URL"); v != "" {
		engineURLDefault = v
	}
	cmd.Flags().String("engine-url", engineURLDefault, "VOICEVOXエンジンのURL")
	speakerDefault := 3
	if v := os.Getenv("VOX_SPEAKER"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			speakerDefault = n
		}
	}
	cmd.Flags().Int("speaker", speakerDefault, "キャラクターID")
	cmd.Flags().Float64("speed", 1.0, "話速")
	cmd.Flags().Float64("pitch", 0.0, "音高")
	cmd.Flags().Float64("intonation", 1.0, "抑揚")
	cmd.Flags().Bool("watch", false, "ディレクトリ監視モードを有効化")

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

	engineURL, _ := cmd.Flags().GetString("engine-url")
	speakerID, _ := cmd.Flags().GetInt("speaker")
	speed, _ := cmd.Flags().GetFloat64("speed")
	pitch, _ := cmd.Flags().GetFloat64("pitch")
	intonation, _ := cmd.Flags().GetFloat64("intonation")

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
		uc := app.NewWatchUsecase(deps.Reader, client, deps.Player, deps.Mover, watcher)
		return uc.Run(ctx, params)
	}

	uc := app.NewActUsecase(deps.Reader, client, deps.Player)
	return uc.Run(ctx, params)
}
