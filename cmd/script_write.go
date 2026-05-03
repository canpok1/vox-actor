package cmd

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/canpok1/vox-actor/internal/app"
	"github.com/canpok1/vox-actor/internal/domain/entity"
	"github.com/spf13/cobra"
)

// ScriptWriteDeps は script write コマンドの依存を保持する。
type ScriptWriteDeps struct {
	WriterFactory func(logger *slog.Logger) app.ScriptBulkWriter
}

// scriptWriteInput は --json フラグで受け取る JSON 配列の各要素のスキーマ。
type scriptWriteInput struct {
	Text       string   `json:"text"`
	Speaker    *int     `json:"speaker"`
	Speed      *float64 `json:"speed"`
	Pitch      *float64 `json:"pitch"`
	Intonation *float64 `json:"intonation"`
}

func makeScriptWriteCmd(deps *ScriptWriteDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "write <file>",
		Short: "セリフを JSON 配列でファイルへ一括書き込みする",
		Long:  "JSON 配列で渡した複数のセリフを指定ファイルへ一括書き込みする（上書き）。VOICEVOX接続・音声再生は行わない。",
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.ExactArgs(1)(cmd, args); err != nil {
				return fmt.Errorf("%w: %s", ErrUsage, err)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runScriptWrite(cmd, args, deps)
		},
	}

	cmd.Flags().String("json", "", "セリフの JSON 配列（必須）")
	cmd.Flags().Bool("verbose", false, "詳細ログを出力")

	return cmd
}

func runScriptWrite(cmd *cobra.Command, args []string, deps *ScriptWriteDeps) error {
	if deps == nil || deps.WriterFactory == nil {
		return fmt.Errorf("ScriptBulkWriter factory is not configured for script write")
	}

	file := args[0]
	if !cmd.Flags().Changed("json") {
		return fmt.Errorf("%w: required flag --json not set", ErrUsage)
	}
	jsonStr, _ := cmd.Flags().GetString("json")

	var inputs []scriptWriteInput
	if err := json.Unmarshal([]byte(jsonStr), &inputs); err != nil {
		return fmt.Errorf("%w: invalid JSON: %s", ErrUsage, err)
	}

	scripts := make([]entity.Script, len(inputs))
	for i, in := range inputs {
		scripts[i] = entity.Script{
			Text:            in.Text,
			IsEmpty:         in.Text == "",
			SpeakerID:       in.Speaker,
			SpeedScale:      in.Speed,
			PitchScale:      in.Pitch,
			IntonationScale: in.Intonation,
		}
	}

	logger := buildLoggerFromFlags(cmd)
	writer := deps.WriterFactory(logger)
	written, err := writer.WriteAll(file, scripts)
	if err != nil {
		return err
	}
	logger.Info("output completed", "path", written)
	return nil
}
