package cmd

import "github.com/spf13/cobra"

func makeScriptCmd(appendDeps *ScriptAppendDeps, writeDeps *ScriptWriteDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "script",
		Short: "セリフファイルを操作する",
		Long:  "セリフファイルに対する操作を行うサブコマンド群。",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(makeScriptAppendCmd(appendDeps))
	cmd.AddCommand(makeScriptWriteCmd(writeDeps))

	return cmd
}
