package infra

import (
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
)

// ErrGitNotFound は git コマンドが PATH 上に見つからない場合のエラー。
var ErrGitNotFound = errors.New("gitコマンドが見つかりません")

// ErrNotInGitRepo はカレントディレクトリが git リポジトリ外であることを示すエラー。
var ErrNotInGitRepo = errors.New("カレントディレクトリはgitリポジトリではありません")

// gitRunner はテスト差し替え可能な git コマンド実行関数。
var gitRunner = exec.Command

// ResolveQueuePath は `git rev-parse --path-format=absolute --git-common-dir` の
// 結果の親ディレクトリに `.vox-actor/queue` を結合したパスを返す。
//
// worktree 上で実行しても `--git-common-dir` により本体リポジトリ直下の
// `.vox-actor/queue` が選ばれる。ディレクトリの作成は行わない。
func ResolveQueuePath() (string, error) {
	cmd := gitRunner("git", "rev-parse", "--path-format=absolute", "--git-common-dir")
	out, err := cmd.Output()
	if err != nil {
		// ExitError（非0終了）は git 外で rev-parse を呼んだ想定に寄せて
		// ErrNotInGitRepo に分類する。それ以外（バイナリ不在など）は ErrGitNotFound。
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return "", ErrNotInGitRepo
		}
		return "", ErrGitNotFound
	}

	gitCommonDir := strings.TrimSpace(string(out))
	if gitCommonDir == "" {
		return "", ErrNotInGitRepo
	}

	repoRoot := filepath.Dir(gitCommonDir)
	return filepath.Join(repoRoot, ".vox-actor", "queue"), nil
}
