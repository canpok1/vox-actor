package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/canpok1/vox-actor/internal/app"
	"github.com/canpok1/vox-actor/internal/domain/entity"
	"github.com/spf13/cobra"
)

// SayDeps はsayコマンドの依存を保持する。
type SayDeps struct {
	ClientFactory func(engineURL string) app.VoicevoxClient
	Player        app.AudioPlayer
	// WriterFactory は -o / --output 指定時のファイル書き出し用 ScriptWriter を生成する。
	// runSay で構築したロガーを Writer に注入できるようファクトリ形式とする。
	// nil の場合は -o オプション利用時にエラーを返す。
	WriterFactory func(logger *slog.Logger) app.ScriptWriter
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

	registerCommonFlags(cmd)
	cmd.Flags().StringP("output", "o", "", "セリフをファイル出力する（既存ファイルには追記）（指定時はVOICEVOX接続・音声再生を行わない）")

	return cmd
}

func runSay(cmd *cobra.Command, args []string, deps *SayDeps) error {
	if deps == nil {
		return fmt.Errorf("say command dependencies are not initialized")
	}

	ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	output, _ := cmd.Flags().GetString("output")

	if output != "" && dryRun {
		return fmt.Errorf("%w: --output and --dry-run cannot be specified together", ErrUsage)
	}

	logger := buildLoggerFromFlags(cmd)

	if output != "" {
		return runSayOutput(cmd, args[0], output, deps, logger)
	}

	if deps.ClientFactory == nil || deps.Player == nil {
		return fmt.Errorf("say command dependencies are not initialized")
	}

	engineURL, _ := cmd.Flags().GetString("engine-url")
	speakerID, _ := cmd.Flags().GetInt("speaker")
	speed, _ := cmd.Flags().GetFloat64("speed")
	pitch, _ := cmd.Flags().GetFloat64("pitch")
	intonation, _ := cmd.Flags().GetFloat64("intonation")

	params := app.SayParams{
		Text:       args[0],
		SpeakerID:  speakerID,
		Speed:      &speed,
		Pitch:      &pitch,
		Intonation: &intonation,
		DryRun:     dryRun,
	}

	client := deps.ClientFactory(engineURL)
	if client == nil {
		return fmt.Errorf("failed to create VoicevoxClient for %s", engineURL)
	}

	uc := app.NewSayUsecase(client, deps.Player, app.WithSayLogger(logger))
	return uc.Run(ctx, params)
}

// runSayOutput は --output モードを実行する。VOICEVOX/再生は呼ばず ScriptWriter で書き出す。
func runSayOutput(cmd *cobra.Command, text, output string, deps *SayDeps, logger *slog.Logger) error {
	if deps.WriterFactory == nil {
		return fmt.Errorf("ScriptWriter factory is not configured for --output")
	}

	script := entity.Script{Text: text, IsEmpty: text == ""}
	if cmd.Flags().Changed("speaker") {
		v, _ := cmd.Flags().GetInt("speaker")
		script.SpeakerID = &v
	}
	if cmd.Flags().Changed("speed") {
		v, _ := cmd.Flags().GetFloat64("speed")
		script.SpeedScale = &v
	}
	if cmd.Flags().Changed("pitch") {
		v, _ := cmd.Flags().GetFloat64("pitch")
		script.PitchScale = &v
	}
	if cmd.Flags().Changed("intonation") {
		v, _ := cmd.Flags().GetFloat64("intonation")
		script.IntonationScale = &v
	}

	writer := deps.WriterFactory(logger)
	written, err := writer.Write(output, script)
	if err != nil {
		return err
	}
	logger.Info("output completed", "path", written)
	return nil
}
