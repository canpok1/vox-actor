package infra

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// withGitRunner はテスト中の gitRunner を一時的に差し替えるヘルパー。
func withGitRunner(t *testing.T, runner func(name string, args ...string) *exec.Cmd) {
	t.Helper()
	orig := gitRunner
	gitRunner = runner
	t.Cleanup(func() {
		gitRunner = orig
	})
}

func TestResolveQueuePath_ReturnsGitCommonDirParentJoinedWithVoxActorQueue(t *testing.T) {
	t.Setenv("VOX_ACTOR_WORKSPACE", "")
	repoRoot := t.TempDir()
	gitCommonDir := filepath.Join(repoRoot, ".git")

	withGitRunner(t, func(name string, args ...string) *exec.Cmd {
		// echo コマンドで git-common-dir の出力を模倣する。
		return exec.Command("echo", gitCommonDir)
	})

	got, err := ResolveQueuePath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(repoRoot, ".vox-actor", "queue")
	if got != want {
		t.Errorf("ResolveQueuePath() = %q, want %q", got, want)
	}
}

func TestResolveQueuePath_ReturnsCwdVoxActorQueue_WhenGitCommonDirEmpty(t *testing.T) {
	t.Setenv("VOX_ACTOR_WORKSPACE", "")
	withGitRunner(t, func(name string, args ...string) *exec.Cmd {
		// 空文字列を標準出力に出すだけのコマンド（=gitリポジトリ外の想定）
		return exec.Command("true")
	})

	cwd, _ := os.Getwd()
	got, err := ResolveQueuePath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(cwd, ".vox-actor", "queue")
	if got != want {
		t.Errorf("ResolveQueuePath() = %q, want %q", got, want)
	}
}

func TestResolveQueuePath_ReturnsCwdVoxActorQueue_WhenGitRunnerFailsWithNonZeroExit(t *testing.T) {
	t.Setenv("VOX_ACTOR_WORKSPACE", "")
	withGitRunner(t, func(name string, args ...string) *exec.Cmd {
		// 非0終了（gitリポジトリ外で git rev-parse した時の挙動を模倣）
		return exec.Command("false")
	})

	cwd, _ := os.Getwd()
	got, err := ResolveQueuePath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(cwd, ".vox-actor", "queue")
	if got != want {
		t.Errorf("ResolveQueuePath() = %q, want %q", got, want)
	}
}

func TestResolveQueuePath_ReturnsErrGitNotFound_WhenRunnerReportsCommandMissing(t *testing.T) {
	t.Setenv("VOX_ACTOR_WORKSPACE", "")
	withGitRunner(t, func(name string, args ...string) *exec.Cmd {
		// PATHに存在しないバイナリを指定することで exec 失敗を再現する。
		return exec.Command("/nonexistent/definitely-no-such-binary-xyz")
	})

	_, err := ResolveQueuePath()
	if !errors.Is(err, ErrGitNotFound) {
		t.Errorf("expected ErrGitNotFound, got: %v", err)
	}
}

func TestResolveQueuePath_TrimsTrailingWhitespaceFromOutput(t *testing.T) {
	t.Setenv("VOX_ACTOR_WORKSPACE", "")
	repoRoot := t.TempDir()
	gitCommonDir := filepath.Join(repoRoot, ".git")

	withGitRunner(t, func(name string, args ...string) *exec.Cmd {
		// printf に末尾改行と空白が含まれる出力を渡す（trim 確認）
		return exec.Command("printf", "%s\n", gitCommonDir)
	})

	got, err := ResolveQueuePath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(repoRoot, ".vox-actor", "queue")
	if got != want {
		t.Errorf("ResolveQueuePath() = %q, want %q", got, want)
	}
	if strings.Contains(got, "\n") {
		t.Errorf("ResolveQueuePath() should not contain newline: %q", got)
	}
}

func TestResolveQueuePath_ReturnsVoxActorWorkspaceQueue_WhenEnvIsSet(t *testing.T) {
	workspace := t.TempDir()
	t.Setenv("VOX_ACTOR_WORKSPACE", workspace)

	// gitRunner は呼ばれない想定だが、保険として非0終了するものを設定しておく。
	withGitRunner(t, func(name string, args ...string) *exec.Cmd {
		return exec.Command("false")
	})

	got, err := ResolveQueuePath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(workspace, "queue")
	if got != want {
		t.Errorf("ResolveQueuePath() = %q, want %q", got, want)
	}
}

func TestResolveTmpPath_ReturnsGitCommonDirParentJoinedWithVoxActorTmp(t *testing.T) {
	t.Setenv("VOX_ACTOR_WORKSPACE", "")
	repoRoot := t.TempDir()
	gitCommonDir := filepath.Join(repoRoot, ".git")

	withGitRunner(t, func(name string, args ...string) *exec.Cmd {
		return exec.Command("echo", gitCommonDir)
	})

	got, err := ResolveTmpPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(repoRoot, ".vox-actor", "tmp")
	if got != want {
		t.Errorf("ResolveTmpPath() = %q, want %q", got, want)
	}
}

func TestResolveTmpPath_ReturnsCwdVoxActorTmp_WhenGitRunnerFailsWithNonZeroExit(t *testing.T) {
	t.Setenv("VOX_ACTOR_WORKSPACE", "")
	withGitRunner(t, func(name string, args ...string) *exec.Cmd {
		return exec.Command("false")
	})

	cwd, _ := os.Getwd()
	got, err := ResolveTmpPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(cwd, ".vox-actor", "tmp")
	if got != want {
		t.Errorf("ResolveTmpPath() = %q, want %q", got, want)
	}
}

func TestResolveTmpPath_ReturnsVoxActorWorkspaceTmp_WhenEnvIsSet(t *testing.T) {
	workspace := t.TempDir()
	t.Setenv("VOX_ACTOR_WORKSPACE", workspace)

	withGitRunner(t, func(name string, args ...string) *exec.Cmd {
		return exec.Command("false")
	})

	got, err := ResolveTmpPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(workspace, "tmp")
	if got != want {
		t.Errorf("ResolveTmpPath() = %q, want %q", got, want)
	}
}

func TestResolveWorkspacePath_ReturnsVoxActorWorkspace_WhenEnvIsSet(t *testing.T) {
	workspace := t.TempDir()
	t.Setenv("VOX_ACTOR_WORKSPACE", workspace)

	// gitRunner は呼ばれない想定だが、保険として非0終了するものを設定しておく。
	withGitRunner(t, func(name string, args ...string) *exec.Cmd {
		return exec.Command("false")
	})

	got, err := ResolveWorkspacePath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != workspace {
		t.Errorf("ResolveWorkspacePath() = %q, want %q", got, workspace)
	}
}

func TestResolveWorkspacePath_ReturnsGitCommonDirParentJoinedWithVoxActor(t *testing.T) {
	t.Setenv("VOX_ACTOR_WORKSPACE", "")
	repoRoot := t.TempDir()
	gitCommonDir := filepath.Join(repoRoot, ".git")

	withGitRunner(t, func(name string, args ...string) *exec.Cmd {
		return exec.Command("echo", gitCommonDir)
	})

	got, err := ResolveWorkspacePath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(repoRoot, ".vox-actor")
	if got != want {
		t.Errorf("ResolveWorkspacePath() = %q, want %q", got, want)
	}
}

func TestResolveWorkspacePath_ReturnsCwdVoxActor_WhenGitCommonDirEmpty(t *testing.T) {
	t.Setenv("VOX_ACTOR_WORKSPACE", "")
	withGitRunner(t, func(name string, args ...string) *exec.Cmd {
		return exec.Command("true")
	})

	cwd, _ := os.Getwd()
	got, err := ResolveWorkspacePath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(cwd, ".vox-actor")
	if got != want {
		t.Errorf("ResolveWorkspacePath() = %q, want %q", got, want)
	}
}

func TestResolveWorkspacePath_ReturnsCwdVoxActor_WhenGitRunnerFailsWithNonZeroExit(t *testing.T) {
	t.Setenv("VOX_ACTOR_WORKSPACE", "")
	withGitRunner(t, func(name string, args ...string) *exec.Cmd {
		return exec.Command("false")
	})

	cwd, _ := os.Getwd()
	got, err := ResolveWorkspacePath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(cwd, ".vox-actor")
	if got != want {
		t.Errorf("ResolveWorkspacePath() = %q, want %q", got, want)
	}
}

func TestResolveWorkspacePath_ReturnsErrGitNotFound_WhenRunnerReportsCommandMissing(t *testing.T) {
	t.Setenv("VOX_ACTOR_WORKSPACE", "")
	withGitRunner(t, func(name string, args ...string) *exec.Cmd {
		return exec.Command("/nonexistent/definitely-no-such-binary-xyz")
	})

	_, err := ResolveWorkspacePath()
	if !errors.Is(err, ErrGitNotFound) {
		t.Errorf("expected ErrGitNotFound, got: %v", err)
	}
}

func TestResolveHomeAssetsPath_ReturnsDotVoxActorAssetsUnderHome(t *testing.T) {
	got, err := ResolveHomeAssetsPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".vox-actor", "assets")
	if got != want {
		t.Errorf("ResolveHomeAssetsPath() = %q, want %q", got, want)
	}
}

func TestResolveProjectAssetsPath_ReturnsGitRepoAssetsDir_WhenInGitRepo(t *testing.T) {
	t.Setenv("VOX_ACTOR_WORKSPACE", "")
	repoRoot := t.TempDir()
	gitCommonDir := filepath.Join(repoRoot, ".git")

	withGitRunner(t, func(name string, args ...string) *exec.Cmd {
		return exec.Command("echo", gitCommonDir)
	})

	got, err := ResolveProjectAssetsPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(repoRoot, ".vox-actor", "assets")
	if got != want {
		t.Errorf("ResolveProjectAssetsPath() = %q, want %q", got, want)
	}
}

func TestResolveProjectAssetsPath_ReturnsCwdAssetsDir_WhenOutsideGitRepo(t *testing.T) {
	t.Setenv("VOX_ACTOR_WORKSPACE", "")
	withGitRunner(t, func(name string, args ...string) *exec.Cmd {
		return exec.Command("false")
	})

	cwd, _ := os.Getwd()
	got, err := ResolveProjectAssetsPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(cwd, ".vox-actor", "assets")
	if got != want {
		t.Errorf("ResolveProjectAssetsPath() = %q, want %q", got, want)
	}
}

func TestResolveProjectAssetsPath_RespectsVoxActorWorkspaceEnv(t *testing.T) {
	workspace := t.TempDir()
	t.Setenv("VOX_ACTOR_WORKSPACE", workspace)

	withGitRunner(t, func(name string, args ...string) *exec.Cmd {
		return exec.Command("false")
	})

	got, err := ResolveProjectAssetsPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(workspace, "assets")
	if got != want {
		t.Errorf("ResolveProjectAssetsPath() = %q, want %q", got, want)
	}
}

func TestResolveHomeViewerHistoryPath_ReturnsDotVoxActorViewerHistoryUnderHome(t *testing.T) {
	got, err := ResolveHomeViewerHistoryPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".vox-actor", "viewer", "history")
	if got != want {
		t.Errorf("ResolveHomeViewerHistoryPath() = %q, want %q", got, want)
	}
}

func TestResolveHomeViewerHistoryPath_IgnoresVoxActorWorkspaceEnv(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("VOX_ACTOR_WORKSPACE", dir)

	got, err := ResolveHomeViewerHistoryPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".vox-actor", "viewer", "history")
	if got != want {
		t.Errorf("ResolveHomeViewerHistoryPath() = %q, want %q (VOX_ACTOR_WORKSPACE should not affect home-scoped path)", got, want)
	}
	if strings.Contains(got, dir) {
		t.Errorf("ResolveHomeViewerHistoryPath() = %q, should not contain workspace dir %q", got, dir)
	}
}

func TestResolveHomeViewerLockPath_ReturnsDotVoxActorViewerLockUnderHome(t *testing.T) {
	got, err := ResolveHomeViewerLockPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".vox-actor", "viewer", "viewer.lock")
	if got != want {
		t.Errorf("ResolveHomeViewerLockPath() = %q, want %q", got, want)
	}
}

func TestResolveHomeViewerLockPath_IgnoresVoxActorWorkspaceEnv(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("VOX_ACTOR_WORKSPACE", dir)

	got, err := ResolveHomeViewerLockPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".vox-actor", "viewer", "viewer.lock")
	if got != want {
		t.Errorf("ResolveHomeViewerLockPath() = %q, want %q (VOX_ACTOR_WORKSPACE should not affect home-scoped path)", got, want)
	}
	if strings.Contains(got, dir) {
		t.Errorf("ResolveHomeViewerLockPath() = %q, should not contain workspace dir %q", got, dir)
	}
}
