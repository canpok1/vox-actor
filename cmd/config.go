package cmd

import (
	"fmt"
	"strings"

	"github.com/canpok1/vox-actor/internal/infra"
	"github.com/spf13/cobra"
)

// supportedConfigKeys は `vox-actor config <key>` で解決可能なキーをアルファベット順で保持する。
var supportedConfigKeys = []string{
	"path.queue",
	"path.tmp",
	"path.workspace",
}

func makeConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config <key>",
		Short: "設定値（パス等）を取得する",
		Long:  "指定したキーの設定値を取得して標準出力に返す。現時点ではパス解決のみ対応する読み取り専用サブコマンド。",
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.ExactArgs(1)(cmd, args); err != nil {
				return fmt.Errorf("%w: %s", ErrUsage, err)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfig(cmd, args[0])
		},
	}
	return cmd
}

func runConfig(cmd *cobra.Command, key string) error {
	var (
		path string
		err  error
	)
	switch key {
	case "path.queue":
		path, err = infra.ResolveQueuePath()
	case "path.tmp":
		path, err = infra.ResolveTmpPath()
	case "path.workspace":
		path, err = infra.ResolveWorkspacePath()
	default:
		return fmt.Errorf("%w: unknown key: %s\nsupported keys:\n%s", ErrUsage, key, formatSupportedKeys())
	}
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintln(cmd.OutOrStdout(), path); err != nil {
		return err
	}
	return nil
}

func formatSupportedKeys() string {
	var b strings.Builder
	for _, k := range supportedConfigKeys {
		b.WriteString("  ")
		b.WriteString(k)
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}
