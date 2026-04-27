package infra

import (
	"context"
	"fmt"
	"os/exec"
)

// GitCloner は git コマンドを使って git clone を実行する。
type GitCloner struct{}

// Clone は repoURL を destDir へ git clone --depth=1 で clone する。
func (g *GitCloner) Clone(ctx context.Context, repoURL string, destDir string) error {
	cmd := exec.CommandContext(ctx, "git", "clone", "--depth=1", repoURL, destDir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clone failed: %w\n%s", err, out)
	}
	return nil
}
