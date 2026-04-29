package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/canpok1/vox-actor/internal/app"
	"github.com/canpok1/vox-actor/internal/infra"
	"github.com/spf13/cobra"
)

var resolveWorkspaceFunc = resolveWorkspaceWithFallback
var resolveHomeAssetsFunc = infra.ResolveHomeAssetsPath

const (
	scopeHome    = "home"
	scopeProject = "project"
)

// AssetsDownloadDeps はassets downloadコマンドの依存を保持する。
type AssetsDownloadDeps struct {
	Cloner app.GitCloner
	// AssetsDirFunc はアセットの配置先ルートディレクトリを返す。nil の場合はワークスペース解決を使う。
	AssetsDirFunc func() (string, error)
}

func makeAssetsDownloadCmd(deps *AssetsDownloadDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "download <repo-url>",
		Short: "git リポジトリからキャラクター画像をダウンロードする",
		Long:  "指定した git リポジトリから vox-actor-assets.json を読み取り、キャラクター画像を .vox-actor/assets/ へコピーする。",
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.ExactArgs(1)(cmd, args); err != nil {
				return fmt.Errorf("%w: %s", ErrUsage, err)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAssetsDownload(cmd, args, deps)
		},
	}

	cmd.Flags().StringArray("speaker", nil, "ダウンロード対象の speaker 名（リピート可、カンマ区切り可）")
	cmd.Flags().Bool("force", false, "ローカルに同名 speaker が存在する場合に上書き")
	cmd.Flags().String("scope", "project", "配置先スコープ（home: ~/.vox-actor/assets/、project: プロジェクト配下）")
	cmd.Flags().Bool("verbose", false, "詳細ログを出力")

	return cmd
}

func runAssetsDownload(cmd *cobra.Command, args []string, deps *AssetsDownloadDeps) error {
	if deps == nil || deps.Cloner == nil {
		return fmt.Errorf("assets download command dependencies are not initialized")
	}

	repoURL := args[0]

	speakerRaw, _ := cmd.Flags().GetStringArray("speaker")
	speakers := splitSpeakers(speakerRaw)

	force, _ := cmd.Flags().GetBool("force")
	scope, _ := cmd.Flags().GetString("scope")

	if scope != scopeHome && scope != scopeProject {
		return fmt.Errorf("%w: --scope must be %q or %q, got %q", ErrUsage, scopeHome, scopeProject, scope)
	}

	assetsDir, err := resolveAssetsDir(deps, scope)
	if err != nil {
		return err
	}

	logger := buildLoggerFromFlags(cmd)

	uc := app.NewAssetsDownloadUsecase(deps.Cloner, app.WithAssetsDownloadLogger(logger))
	return uc.Run(cmd.Context(), app.AssetsDownloadParams{
		RepoURL:   repoURL,
		Speakers:  speakers,
		Force:     force,
		AssetsDir: assetsDir,
	})
}

// splitSpeakers はカンマ区切りを含む speaker 文字列スライスを個々の speaker 名に展開する。
func splitSpeakers(raw []string) []string {
	var result []string
	for _, s := range raw {
		for _, name := range strings.Split(s, ",") {
			name = strings.TrimSpace(name)
			if name != "" {
				result = append(result, name)
			}
		}
	}
	return result
}

func resolveAssetsDir(deps *AssetsDownloadDeps, scope string) (string, error) {
	if deps.AssetsDirFunc != nil {
		return deps.AssetsDirFunc()
	}
	if scope == scopeHome {
		return resolveHomeAssetsFunc()
	}
	ws, err := resolveWorkspaceFunc()
	if err != nil {
		return "", err
	}
	return filepath.Join(ws, "assets"), nil
}
