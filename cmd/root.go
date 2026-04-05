package cmd

import (
	"errors"

	"github.com/spf13/cobra"
)

var version = "dev"

// ErrUsage はCLIの引数エラーを示すエラー。
// 呼び出し元はこのエラーを検出して終了コード2で終了すべき。
var ErrUsage = errors.New("usage error")

// Deps はルートコマンドの依存を保持する。
type Deps struct {
	Act *ActDeps
	Say *SayDeps
}

func makeRootCmd(deps ...*Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "vox-actor",
		Short:   "テキストをVOICEVOXエンジンで音声合成し読み上げるCLIツール",
		Version: version,
	}
	cmd.SilenceUsage = true
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	}

	var d *Deps
	if len(deps) > 0 {
		d = deps[0]
	}

	var actDeps *ActDeps
	var sayDeps *SayDeps
	if d != nil {
		actDeps = d.Act
		sayDeps = d.Say
	}
	cmd.AddCommand(makeActCmd(actDeps))
	cmd.AddCommand(makeSayCmd(sayDeps))

	return cmd
}

// Execute はルートコマンドを実行する。
func Execute(deps *Deps) error {
	return makeRootCmd(deps).Execute()
}
