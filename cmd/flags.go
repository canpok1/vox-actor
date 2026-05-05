package cmd

import (
	"log/slog"
	"os"
	"strconv"

	"github.com/canpok1/vox-actor/internal/infra/logging"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// buildLoggerFromFlags は --verbose / --dry-run フラグと端末/環境変数の状態からロガーを構築する。
// 標準エラー出力が端末でない場合、または NO_COLOR 環境変数が設定されている場合は色を無効化する。
// --dry-run が指定されている場合はログメッセージに [dry run] プレフィックスを挿入する。
func buildLoggerFromFlags(cmd *cobra.Command) *slog.Logger {
	verbose, _ := cmd.Flags().GetBool("verbose")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	logLevel := slog.LevelInfo
	if verbose {
		logLevel = slog.LevelDebug
	}
	noColor := !term.IsTerminal(int(os.Stderr.Fd())) || os.Getenv("NO_COLOR") != ""
	return slog.New(logging.NewHumanHandler(os.Stderr, &logging.HumanHandlerOptions{
		Level:   logLevel,
		NoColor: noColor,
		DryRun:  dryRun,
	}))
}

// speakerDefaultFromEnv は VOX_SPEAKER 環境変数からデフォルトのスピーカーIDを解決する。
func speakerDefaultFromEnv() int {
	if v := os.Getenv("VOX_SPEAKER"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return 3
}

// registerVoiceParamFlags はキャラクター音声パラメータ（speaker/speed/pitch/intonation/verbose）を登録する。
// VOICEVOX エンジン接続が不要なコマンドでも使える共通フラグ群。
func registerVoiceParamFlags(cmd *cobra.Command) {
	cmd.Flags().Int("speaker", speakerDefaultFromEnv(), "キャラクターID")
	cmd.Flags().Float64("speed", 1.0, "話速")
	cmd.Flags().Float64("pitch", 0.0, "音高")
	cmd.Flags().Float64("intonation", 1.0, "抑揚")
	cmd.Flags().Bool("verbose", false, "詳細ログを出力")
}

// viewerURLDefaultFromEnv は VOX_VIEWER_URL 環境変数からデフォルトの viewer URL を解決する。
func viewerURLDefaultFromEnv() string {
	return os.Getenv("VOX_VIEWER_URL")
}

// registerBaseFlags は dry-run を除く共通フラグを登録する。viewer など dry-run を持たないコマンドで使う。
func registerBaseFlags(cmd *cobra.Command) {
	engineURLDefault := "http://localhost:50021"
	if v := os.Getenv("VOX_ENGINE_URL"); v != "" {
		engineURLDefault = v
	}
	cmd.Flags().String("engine-url", engineURLDefault, "VOICEVOXエンジンのURL")

	registerVoiceParamFlags(cmd)
}

// registerViewerURLFlag は --viewer-url フラグを登録する。
func registerViewerURLFlag(cmd *cobra.Command) {
	cmd.Flags().String("viewer-url", viewerURLDefaultFromEnv(), "viewer の HTTP エンドポイント URL (例: http://192.168.1.10:8080)。指定時は lockfile auto-detect をスキップして明示 URL の viewer に POST する。")
}

// saveWavDefaultFromEnv は VOX_SAVE_WAV 環境変数からデフォルトの保存パスを解決する。
func saveWavDefaultFromEnv() string {
	return os.Getenv("VOX_SAVE_WAV")
}

// registerCommonFlags は各サブコマンド共通のフラグを登録する。
func registerCommonFlags(cmd *cobra.Command) {
	registerBaseFlags(cmd)
	cmd.Flags().Bool("dry-run", false, "VOICEVOX・音声再生を行わず、読み上げ対象をログ出力")
	registerViewerURLFlag(cmd)
}

// registerSaveWavFlag は --save-wav フラグを登録する。say コマンドなど wav 保存が必要なコマンドで呼ぶ。
func registerSaveWavFlag(cmd *cobra.Command) {
	cmd.Flags().String("save-wav", saveWavDefaultFromEnv(), "合成した WAV を保存するパス（ローカル合成時のみ。親ディレクトリは自動作成）")
}
