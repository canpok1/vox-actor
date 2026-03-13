package cmd

import (
	"github.com/spf13/cobra"
)

var version = "dev"

func makeRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "vox-actor",
		Short:   "AIエージェント構築フレームワークのCLIツール",
		Version: version,
	}
	cmd.SilenceUsage = true
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	}
	return cmd
}

// Execute はルートコマンドを実行する。
func Execute() error {
	return makeRootCmd().Execute()
}
