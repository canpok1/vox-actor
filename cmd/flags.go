package cmd

import (
	"os"
	"strconv"

	"github.com/spf13/cobra"
)

// registerCommonFlags はact/sayコマンド共通のフラグを登録する。
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
