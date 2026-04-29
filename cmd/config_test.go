package cmd

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// withTempGitRepo はテスト実行中だけカレントディレクトリを一時的なgitリポジトリに切り替える。
func withTempGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	// 一時ディレクトリ直下に .git を作って git-common-dir が返せるようにする。
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatalf("failed to create .git: %v", err)
	}
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("git init failed, skipping: %v\n%s", err, out)
	}
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(orig)
	})
	return dir
}

// withTempNonGitDir はテスト実行中だけカレントディレクトリを一時的な非gitディレクトリに切り替える。
func withTempNonGitDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(orig)
	})
	return dir
}

func TestConfigCmd_RegisteredAsSubcommand(t *testing.T) {
	rootCmd := makeRootCmd()

	found := false
	for _, c := range rootCmd.Commands() {
		if c.Name() == "config" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'config' subcommand to be registered")
	}
}

func TestConfigCmd_PathQueue_InsideGitRepo_PrintsAbsolutePath(t *testing.T) {
	t.Setenv("VOX_ACTOR_WORKSPACE", "")
	repoRoot := withTempGitRepo(t)

	rootCmd := makeRootCmd()
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rootCmd.SetOut(outBuf)
	rootCmd.SetErr(errBuf)
	rootCmd.SetArgs([]string{"config", "path.queue"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v, stderr: %s", err, errBuf.String())
	}

	got := strings.TrimRight(outBuf.String(), "\n")
	want := filepath.Join(repoRoot, ".vox-actor", "queue")
	// macOS の /var → /private/var シンボリックリンクを考慮して EvalSymlinks で比較する。
	wantEval, _ := filepath.EvalSymlinks(filepath.Dir(want))
	gotEval, _ := filepath.EvalSymlinks(filepath.Dir(got))
	if gotEval != wantEval || filepath.Base(got) != filepath.Base(want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestConfigCmd_PathWorkspace_InsideGitRepo_PrintsAbsolutePath(t *testing.T) {
	t.Setenv("VOX_ACTOR_WORKSPACE", "")
	repoRoot := withTempGitRepo(t)

	rootCmd := makeRootCmd()
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rootCmd.SetOut(outBuf)
	rootCmd.SetErr(errBuf)
	rootCmd.SetArgs([]string{"config", "path.workspace"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v, stderr: %s", err, errBuf.String())
	}

	got := strings.TrimRight(outBuf.String(), "\n")
	want := filepath.Join(repoRoot, ".vox-actor")
	wantEval, _ := filepath.EvalSymlinks(filepath.Dir(want))
	gotEval, _ := filepath.EvalSymlinks(filepath.Dir(got))
	if gotEval != wantEval || filepath.Base(got) != filepath.Base(want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestConfigCmd_PathWorkspace_WithEnv_PrintsEnvValue(t *testing.T) {
	workspace := t.TempDir()
	t.Setenv("VOX_ACTOR_WORKSPACE", workspace)

	rootCmd := makeRootCmd()
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rootCmd.SetOut(outBuf)
	rootCmd.SetErr(errBuf)
	rootCmd.SetArgs([]string{"config", "path.workspace"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v, stderr: %s", err, errBuf.String())
	}

	got := strings.TrimRight(outBuf.String(), "\n")
	if got != workspace {
		t.Errorf("got %q, want %q", got, workspace)
	}
}

func TestConfigCmd_PathWorkspace_OutsideGitRepo_PrintsVoxActorSubdir(t *testing.T) {
	t.Setenv("VOX_ACTOR_WORKSPACE", "")
	cwd := withTempNonGitDir(t)

	rootCmd := makeRootCmd()
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rootCmd.SetOut(outBuf)
	rootCmd.SetErr(errBuf)
	rootCmd.SetArgs([]string{"config", "path.workspace"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v, stderr: %s", err, errBuf.String())
	}

	got := strings.TrimRight(outBuf.String(), "\n")
	want := filepath.Join(cwd, ".vox-actor")
	// macOS の /var → /private/var シンボリックリンクを考慮して比較する。
	gotEval, _ := filepath.EvalSymlinks(filepath.Dir(got))
	wantEval, _ := filepath.EvalSymlinks(filepath.Dir(want))
	if gotEval != wantEval || filepath.Base(got) != filepath.Base(want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestConfigCmd_PathQueue_WithEnv_PrintsEnvValueJoinedWithQueue(t *testing.T) {
	workspace := t.TempDir()
	t.Setenv("VOX_ACTOR_WORKSPACE", workspace)

	rootCmd := makeRootCmd()
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rootCmd.SetOut(outBuf)
	rootCmd.SetErr(errBuf)
	rootCmd.SetArgs([]string{"config", "path.queue"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v, stderr: %s", err, errBuf.String())
	}

	got := strings.TrimRight(outBuf.String(), "\n")
	want := filepath.Join(workspace, "queue")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestConfigCmd_PathQueue_OutsideGitRepo_PrintsCwdQueue(t *testing.T) {
	t.Setenv("VOX_ACTOR_WORKSPACE", "")
	cwd := withTempNonGitDir(t)

	rootCmd := makeRootCmd()
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rootCmd.SetOut(outBuf)
	rootCmd.SetErr(errBuf)
	rootCmd.SetArgs([]string{"config", "path.queue"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v, stderr: %s", err, errBuf.String())
	}

	got := strings.TrimRight(outBuf.String(), "\n")
	want := filepath.Join(cwd, ".vox-actor", "queue")
	wantEval, _ := filepath.EvalSymlinks(filepath.Dir(want))
	gotEval, _ := filepath.EvalSymlinks(filepath.Dir(got))
	if gotEval != wantEval || filepath.Base(got) != filepath.Base(want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestConfigCmd_PathTmp_InsideGitRepo_PrintsAbsolutePath(t *testing.T) {
	t.Setenv("VOX_ACTOR_WORKSPACE", "")
	repoRoot := withTempGitRepo(t)

	rootCmd := makeRootCmd()
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rootCmd.SetOut(outBuf)
	rootCmd.SetErr(errBuf)
	rootCmd.SetArgs([]string{"config", "path.tmp"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v, stderr: %s", err, errBuf.String())
	}

	got := strings.TrimRight(outBuf.String(), "\n")
	want := filepath.Join(repoRoot, ".vox-actor", "tmp")
	wantEval, _ := filepath.EvalSymlinks(filepath.Dir(want))
	gotEval, _ := filepath.EvalSymlinks(filepath.Dir(got))
	if gotEval != wantEval || filepath.Base(got) != filepath.Base(want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestConfigCmd_PathTmp_OutsideGitRepo_PrintsCwdTmp(t *testing.T) {
	t.Setenv("VOX_ACTOR_WORKSPACE", "")
	cwd := withTempNonGitDir(t)

	rootCmd := makeRootCmd()
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rootCmd.SetOut(outBuf)
	rootCmd.SetErr(errBuf)
	rootCmd.SetArgs([]string{"config", "path.tmp"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v, stderr: %s", err, errBuf.String())
	}

	got := strings.TrimRight(outBuf.String(), "\n")
	want := filepath.Join(cwd, ".vox-actor", "tmp")
	wantEval, _ := filepath.EvalSymlinks(filepath.Dir(want))
	gotEval, _ := filepath.EvalSymlinks(filepath.Dir(got))
	if gotEval != wantEval || filepath.Base(got) != filepath.Base(want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestConfigCmd_PathTmp_WithEnv_PrintsEnvValueJoinedWithTmp(t *testing.T) {
	workspace := t.TempDir()
	t.Setenv("VOX_ACTOR_WORKSPACE", workspace)

	rootCmd := makeRootCmd()
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rootCmd.SetOut(outBuf)
	rootCmd.SetErr(errBuf)
	rootCmd.SetArgs([]string{"config", "path.tmp"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v, stderr: %s", err, errBuf.String())
	}

	got := strings.TrimRight(outBuf.String(), "\n")
	want := filepath.Join(workspace, "tmp")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestConfigCmd_UnknownKey_ReturnsErrUsageAndListsSupportedKeys(t *testing.T) {
	rootCmd := makeRootCmd()
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rootCmd.SetOut(outBuf)
	rootCmd.SetErr(errBuf)
	rootCmd.SetArgs([]string{"config", "path.unknown"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for unknown key, got nil")
	}
	if !errors.Is(err, ErrUsage) {
		t.Errorf("expected ErrUsage, got: %v", err)
	}
	msg := err.Error()
	if !strings.Contains(msg, "unknown key: path.unknown") {
		t.Errorf("expected error message to contain 'unknown key: path.unknown', got: %v", err)
	}
	if !strings.Contains(msg, "path.queue") {
		t.Errorf("expected error message to list supported key 'path.queue', got: %v", err)
	}
}

func TestConfigCmd_NoArgs_ReturnsError(t *testing.T) {
	rootCmd := makeRootCmd()
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rootCmd.SetOut(outBuf)
	rootCmd.SetErr(errBuf)
	rootCmd.SetArgs([]string{"config"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when no key is given, got nil")
	}
	if !errors.Is(err, ErrUsage) {
		t.Errorf("expected ErrUsage, got: %v", err)
	}
}
