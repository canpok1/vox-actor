package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/canpok1/vox-actor/internal/app"
	"github.com/spf13/cobra"
)

// SayDeps はsayコマンドの依存を保持する。
type SayDeps struct {
	ClientFactory func(engineURL string) app.VoicevoxClient
	Player        app.AudioPlayer
}

func makeSayCmd(deps *SayDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "say <text>",
		Short: "テキストを直接指定して読み上げる",
		Long:  "テキストを引数で指定し、VOICEVOXエンジンで音声合成して再生する。",
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.ExactArgs(1)(cmd, args); err != nil {
				return fmt.Errorf("%w: %s", ErrUsage, err)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSay(cmd, args, deps)
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
	cmd.Flags().Bool("verbose", false, "詳細ログを出力")

	return cmd
}

func runSay(cmd *cobra.Command, args []string, deps *SayDeps) error {
	if deps == nil || deps.ClientFactory == nil || deps.Player == nil {
		return fmt.Errorf("say command dependencies are not initialized")
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

	logLevel := slog.LevelInfo
	if verbose {
		logLevel = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel}))

	client := deps.ClientFactory(engineURL)
	if client == nil {
		return fmt.Errorf("failed to create VoicevoxClient for %s", engineURL)
	}

	params := app.SayParams{
		Text:       args[0],
		SpeakerID:  speakerID,
		Speed:      &speed,
		Pitch:      &pitch,
		Intonation: &intonation,
	}

	uc := app.NewSayUsecase(client, deps.Player, app.WithSayLogger(logger))
	return uc.Run(ctx, params)
}
