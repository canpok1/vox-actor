package cmd

import (
	"fmt"
	"log/slog"

	"github.com/canpok1/vox-actor/internal/app"
	"github.com/canpok1/vox-actor/internal/domain/entity"
	"github.com/spf13/cobra"
)

// ScriptAppendDeps は script append コマンドの依存を保持する。
type ScriptAppendDeps struct {
	WriterFactory func(logger *slog.Logger) app.ScriptWriter
}

func makeScriptAppendCmd(deps *ScriptAppendDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "append <file> <text>",
		Short: "セリフをファイルへ追記する",
		Long:  "指定したセリフファイルにテキストを追記する。既存ファイルには追記、存在しない場合は新規作成する。VOICEVOX接続・音声再生は行わない。",
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.ExactArgs(2)(cmd, args); err != nil {
				return fmt.Errorf("%w: %s", ErrUsage, err)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runScriptAppend(cmd, args, deps)
		},
	}

	registerVoiceParamFlags(cmd)

	return cmd
}

func runScriptAppend(cmd *cobra.Command, args []string, deps *ScriptAppendDeps) error {
	if deps == nil || deps.WriterFactory == nil {
		return fmt.Errorf("ScriptWriter factory is not configured for script append")
	}

	file := args[0]
	text := args[1]

	script := entity.Script{Text: text, IsEmpty: text == ""}
	if cmd.Flags().Changed("speaker") {
		v, _ := cmd.Flags().GetInt("speaker")
		script.SpeakerID = &v
	}
	if cmd.Flags().Changed("speed") {
		v, _ := cmd.Flags().GetFloat64("speed")
		script.Overrides.Speed = &v
	}
	if cmd.Flags().Changed("pitch") {
		v, _ := cmd.Flags().GetFloat64("pitch")
		script.Overrides.Pitch = &v
	}
	if cmd.Flags().Changed("intonation") {
		v, _ := cmd.Flags().GetFloat64("intonation")
		script.Overrides.Intonation = &v
	}

	logger := buildLoggerFromFlags(cmd)
	writer := deps.WriterFactory(logger)
	written, err := writer.Write(file, script)
	if err != nil {
		return err
	}
	logger.Info("output completed", "path", written)
	return nil
}
