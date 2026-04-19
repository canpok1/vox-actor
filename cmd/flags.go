package cmd

import (
	"log/slog"
	"os"
	"strconv"

	"github.com/canpok1/vox-actor/internal/infra/logging"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// buildLoggerFromFlags は --verbose フラグと端末/環境変数の状態からロガーを構築する。
// 標準エラー出力が端末でない場合、または NO_COLOR 環境変数が設定されている場合は色を無効化する。
func buildLoggerFromFlags(cmd *cobra.Command) *slog.Logger {
	verbose, _ := cmd.Flags().GetBool("verbose")
	logLevel := slog.LevelInfo
	if verbose {
		logLevel = slog.LevelDebug
	}
	noColor := !term.IsTerminal(int(os.Stderr.Fd())) || os.Getenv("NO_COLOR") != ""
	return slog.New(logging.NewHumanHandler(os.Stderr, &logging.HumanHandlerOptions{
		Level:   logLevel,
		NoColor: noColor,
	}))
}

// registerCommonFlags は各サブコマンド共通のフラグを登録する。
func registerCommonFlags(cmd *cobra.Command) {
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
}
