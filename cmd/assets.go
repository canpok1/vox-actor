package cmd

import "github.com/spf13/cobra"

func makeAssetsCmd(deps *AssetsDownloadDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "assets",
		Short: "アセット管理コマンド",
		Long:  "キャラクター画像などのアセットを管理するサブコマンド群。",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(makeAssetsDownloadCmd(deps))
	return cmd
}
