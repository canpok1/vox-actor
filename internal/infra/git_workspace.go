package infra

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ErrGitNotFound は git コマンドが PATH 上に見つからない場合のエラー。
var ErrGitNotFound = errors.New("gitコマンドが見つかりません")

// ErrNotInGitRepo はカレントディレクトリが git リポジトリ外であることを示すエラー。
var ErrNotInGitRepo = errors.New("カレントディレクトリはgitリポジトリではありません")

// envWorkspaceKey はワークスペースルートを明示指定するための環境変数名。
const envWorkspaceKey = "VOX_ACTOR_WORKSPACE"

// gitRunner はテスト差し替え可能な git コマンド実行関数。
var gitRunner = exec.Command

// ResolveWorkspacePath は vox-actor のワークスペースルートの絶対パスを返す。
//
// 解決順:
//  1. 環境変数 VOX_ACTOR_WORKSPACE が設定されていればその値
//  2. git リポジトリ内であれば `<repoRoot>/.vox-actor`
//  3. それ以外は `<cwd>/.vox-actor`
func ResolveWorkspacePath() (string, error) {
	if v := os.Getenv(envWorkspaceKey); v != "" {
		return v, nil
	}

	cmd := gitRunner("git", "rev-parse", "--path-format=absolute", "--git-common-dir")
	out, err := cmd.Output()
	if err != nil {
		// ExitError（非0終了）は git 管理外で rev-parse を呼んだ想定に寄せて
		// cwd/.vox-actor にフォールバックする。それ以外（バイナリ不在など）は ErrGitNotFound。
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return cwdDotVoxActor()
		}
		return "", ErrGitNotFound
	}

	gitCommonDir := strings.TrimSpace(string(out))
	if gitCommonDir == "" {
		return cwdDotVoxActor()
	}

	repoRoot := filepath.Dir(gitCommonDir)
	return filepath.Join(repoRoot, ".vox-actor"), nil
}

func cwdDotVoxActor() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Join(cwd, ".vox-actor"), nil
}

// ResolveQueuePath はワークスペースルート配下の queue ディレクトリ絶対パスを返す。
//
// 実装は ResolveWorkspacePath の結果に `queue` を結合するだけで、環境変数/git解決は
// ResolveWorkspacePath に一元化する。ディレクトリの作成は行わない。
func ResolveQueuePath() (string, error) {
	workspace, err := ResolveWorkspacePath()
	if err != nil {
		return "", err
	}
	return filepath.Join(workspace, "queue"), nil
}

func resolveHomeViewerDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".vox-actor", "viewer"), nil
}

// ResolveHomeViewerHistoryPath はホームスコープの viewer 履歴ディレクトリパス (~/.vox-actor/viewer/history/) を返す。
func ResolveHomeViewerHistoryPath() (string, error) {
	base, err := resolveHomeViewerDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "history"), nil
}

// ResolveHomeViewerLockPath はホームスコープの viewer ロックファイルパス (~/.vox-actor/viewer/viewer.lock) を返す。
func ResolveHomeViewerLockPath() (string, error) {
	base, err := resolveHomeViewerDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "viewer.lock"), nil
}

// ResolveHomeAssetsPath はホームスコープのアセットルートパス (~/.vox-actor/assets/) を返す。
func ResolveHomeAssetsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".vox-actor", "assets"), nil
}

// ResolveProjectAssetsPath はプロジェクトスコープのアセットルートパスを返す。
// git リポジトリ内なら <repoRoot>/.vox-actor/assets/、それ以外は <cwd>/.vox-actor/assets/。
func ResolveProjectAssetsPath() (string, error) {
	path, err := ResolveWorkspacePath()
	if err != nil {
		return "", err
	}
	return filepath.Join(path, "assets"), nil
}
