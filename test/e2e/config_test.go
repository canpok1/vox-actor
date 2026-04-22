//go:build e2e

package e2e

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// initTempGitRepo は t.TempDir() 配下に `git init` で新規リポジトリを作成し、
// symlink を解決したリポジトリ絶対パスを返す。git が利用できない環境では Skip する。
func initTempGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("git init failed, skipping: %v\n%s", err, out)
	}
	// macOS の /var ↔ /private/var のような差異を吸収するため、
	// 比較側も子プロセス側も同じく symlink 解決後のパスで揃える。
	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatalf("failed to eval symlinks for %q: %v", dir, err)
	}
	return resolved
}

// assertSinglePathLine は stdout がパス1行＋末尾LFの形式になっていることを検証し、
// 末尾の改行を除いた文字列を返す。
func assertSinglePathLine(t *testing.T, stdout string) string {
	t.Helper()
	if !strings.HasSuffix(stdout, "\n") {
		t.Errorf("expected stdout to end with newline, got %q", stdout)
	}
	if n := countNonEmptyLines(stdout); n != 1 {
		t.Errorf("expected exactly 1 non-empty line in stdout, got %d\nstdout:\n%s", n, stdout)
	}
	return strings.TrimRight(stdout, "\n")
}

func TestConfigE2E_PathWorkspace_WithEnv(t *testing.T) {
	workspace := t.TempDir()
	env := map[string]string{"VOX_ACTOR_WORKSPACE": workspace}

	stdout, stderr, exitCode := runCLI(t, env, "config", "path.workspace")
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d\nstderr:\n%s", exitCode, stderr)
	}

	got := assertSinglePathLine(t, stdout)
	if got != workspace {
		t.Errorf("got %q, want %q", got, workspace)
	}
}

func TestConfigE2E_PathQueue_WithEnv(t *testing.T) {
	workspace := t.TempDir()
	env := map[string]string{"VOX_ACTOR_WORKSPACE": workspace}

	stdout, stderr, exitCode := runCLI(t, env, "config", "path.queue")
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d\nstderr:\n%s", exitCode, stderr)
	}

	got := assertSinglePathLine(t, stdout)
	want := filepath.Join(workspace, "queue")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestConfigE2E_PathWorkspace_InsideGitRepo(t *testing.T) {
	repoRoot := initTempGitRepo(t)
	// 既存の VOX_ACTOR_WORKSPACE を明示的に空で上書きし、git解決経路をテストする。
	env := map[string]string{"VOX_ACTOR_WORKSPACE": ""}

	stdout, stderr, exitCode := runCLIInDir(t, repoRoot, env, "config", "path.workspace")
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d\nstderr:\n%s", exitCode, stderr)
	}

	got := assertSinglePathLine(t, stdout)
	want := filepath.Join(repoRoot, ".vox-actor")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestConfigE2E_PathQueue_InsideGitRepo(t *testing.T) {
	repoRoot := initTempGitRepo(t)
	env := map[string]string{"VOX_ACTOR_WORKSPACE": ""}

	stdout, stderr, exitCode := runCLIInDir(t, repoRoot, env, "config", "path.queue")
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d\nstderr:\n%s", exitCode, stderr)
	}

	got := assertSinglePathLine(t, stdout)
	want := filepath.Join(repoRoot, ".vox-actor", "queue")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestConfigE2E_UnknownKey_ExitCode2(t *testing.T) {
	stdout, stderr, exitCode := runCLI(t, nil, "config", "path.unknown")
	if exitCode != 2 {
		t.Fatalf("expected exit code 2, got %d\nstderr:\n%s", exitCode, stderr)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("expected empty stdout, got %q", stdout)
	}
	wants := []string{"unknown key: path.unknown", "path.queue", "path.workspace"}
	for _, w := range wants {
		if !strings.Contains(stderr, w) {
			t.Errorf("expected stderr to contain %q\nstderr:\n%s", w, stderr)
		}
	}
}

func TestConfigE2E_GitNotFound_ExitCode1(t *testing.T) {
	dir := t.TempDir()
	// PATH を空にして子プロセス内からの git コマンド探索を失敗させる。
	env := map[string]string{
		"PATH":                "",
		"VOX_ACTOR_WORKSPACE": "",
	}

	stdout, stderr, exitCode := runCLIInDir(t, dir, env, "config", "path.workspace")
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d\nstderr:\n%s", exitCode, stderr)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("expected empty stdout, got %q", stdout)
	}
	if strings.TrimSpace(stderr) == "" {
		t.Error("expected non-empty stderr for git-not-found")
	}
}

func TestConfigE2E_OutsideGitRepo_ExitCode1(t *testing.T) {
	dir := t.TempDir()
	env := map[string]string{"VOX_ACTOR_WORKSPACE": ""}

	stdout, stderr, exitCode := runCLIInDir(t, dir, env, "config", "path.workspace")
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d\nstderr:\n%s", exitCode, stderr)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("expected empty stdout, got %q", stdout)
	}
	if strings.TrimSpace(stderr) == "" {
		t.Error("expected non-empty stderr for outside-git-repo")
	}
}
