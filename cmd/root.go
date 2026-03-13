package cmd

import (
	"github.com/spf13/cobra"
)

var version = "dev"

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
