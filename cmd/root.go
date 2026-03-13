package cmd

import (
	"errors"

	"github.com/spf13/cobra"
)

var version = "dev"

// ErrUsage はCLIの引数エラーを示すエラー。
// 呼び出し元はこのエラーを検出して終了コード2で終了すべき。
var ErrUsage = errors.New("usage error")

func makeRootCmd(actDeps ...*ActDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "vox-actor",
		Short:   "AIエージェント構築フレームワークのCLIツール",
		Version: version,
	}
	cmd.SilenceUsage = true
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	}

	var deps *ActDeps
	if len(actDeps) > 0 {
		deps = actDeps[0]
	}
	cmd.AddCommand(makeActCmd(deps))

	return cmd
}

// Execute はルートコマンドを実行する。
func Execute(actDeps *ActDeps) error {
	return makeRootCmd(actDeps).Execute()
}
