package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

// ErrAudioCheckFailed はaudio-checkがデバイス初期化に失敗した場合のセンチネルエラー。
// main側で終了コード1にマップされる（usageエラーではないため2にはならない）。
var ErrAudioCheckFailed = errors.New("audio device check failed")

// AudioCheckDeps はaudio-checkコマンドの依存を保持する。
type AudioCheckDeps struct {
	// CheckFunc はデバイス可用性を判定する関数。
	// 成功時はバックエンド名を返し、失敗時はエラーを返す。
	CheckFunc func() (string, error)
}

func makeAudioCheckCmd(deps *AudioCheckDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "audio-check",
		Short: "音声出力デバイスの可用性を確認する",
		Long: "音声出力デバイスを open → 即 close のみ行い、可用性を終了コードで返す診断サブコマンド。\n" +
			"通常実行時は標準出力・標準エラー出力ともに空で、--verbose 指定時のみ stderr に診断メッセージを出す。",
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.NoArgs(cmd, args); err != nil {
				return fmt.Errorf("%w: %s", ErrUsage, err)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAudioCheck(cmd, deps)
		},
	}
	cmd.Flags().BoolP("verbose", "v", false, "診断メッセージを標準エラー出力に出力する")
	// 失敗時も stderr を汚さないため、cobraの自動エラー表示を抑制する。
	cmd.SilenceErrors = true
	return cmd
}

func runAudioCheck(cmd *cobra.Command, deps *AudioCheckDeps) error {
	if deps == nil || deps.CheckFunc == nil {
		return fmt.Errorf("audio-check command dependencies are not initialized")
	}

	verbose, _ := cmd.Flags().GetBool("verbose")

	backend, err := deps.CheckFunc()
	if err != nil {
		if verbose {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Error: %v\n", err)
		}
		return ErrAudioCheckFailed
	}

	if verbose {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "audio backend: %s\n", backend)
	}
	return nil
}
