package cmd

import (
	"errors"
	"fmt"

	"github.com/canpok1/vox-actor/internal/infra"
	"github.com/spf13/cobra"
)

// ErrViewerCheckFailed はviewer-checkがviewer未起動を検出した場合のセンチネルエラー。
// main側で終了コード1にマップされる（usageエラーではないため2にはならない）。
var ErrViewerCheckFailed = errors.New("viewer not running")

// ViewerCheckDeps はviewer-checkコマンドの依存を保持する。
type ViewerCheckDeps struct {
	// LockPathResolver はviewerロックファイルのパスを返す関数。
	LockPathResolver func() (string, error)
	// DetectFunc はviewer起動有無を判定する関数。
	// 起動中の場合はアドレスとtrueを返し、未起動の場合は空文字とfalseを返す。
	DetectFunc func(lockPath string) (addr string, ok bool)
}

func makeViewerCheckCmd(deps *ViewerCheckDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "viewer-check",
		Short: "viewer の起動有無を確認する",
		Long: "viewer のロックファイルと /api/status への疎通確認を行い、起動有無を終了コードで返す診断サブコマンド。\n" +
			"通常実行時は標準出力・標準エラー出力ともに空で、--verbose 指定時のみ stdout に診断メッセージを出す。",
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.NoArgs(cmd, args); err != nil {
				return fmt.Errorf("%w: %s", ErrUsage, err)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runViewerCheck(cmd, deps)
		},
	}
	cmd.Flags().BoolP("verbose", "v", false, "診断メッセージを標準出力に出力する")
	// 失敗時も stderr を汚さないため、cobraの自動エラー表示を抑制する。
	cmd.SilenceErrors = true
	return cmd
}

func runViewerCheck(cmd *cobra.Command, deps *ViewerCheckDeps) error {
	if deps == nil || deps.LockPathResolver == nil || deps.DetectFunc == nil {
		return fmt.Errorf("viewer-check command dependencies are not initialized")
	}

	verbose, _ := cmd.Flags().GetBool("verbose")

	lockPath, err := deps.LockPathResolver()
	if err != nil {
		return fmt.Errorf("failed to resolve viewer lock path: %w", err)
	}

	addr, ok := deps.DetectFunc(lockPath)
	if !ok {
		return ErrViewerCheckFailed
	}

	if verbose {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "viewer addr: %s\n", addr)
	}
	return nil
}

// NewViewerCheckDeps は本番用の ViewerCheckDeps を返す。
func NewViewerCheckDeps() *ViewerCheckDeps {
	return &ViewerCheckDeps{
		LockPathResolver: infra.ResolveHomeViewerLockPath,
		DetectFunc:       infra.DetectViewer,
	}
}
