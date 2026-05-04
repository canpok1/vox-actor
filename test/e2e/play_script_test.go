//go:build e2e

package e2e

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// playScriptPath は play-script.sh の絶対パスを返す。
// テストは test/e2e/ から実行されるため ../../ がリポジトリルートになる。
func findPlayScriptPath(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	scriptPath := filepath.Join(wd, "..", "..", "plugins", "vox-actor-plugin", "skills", "talk", "scripts", "play-script.sh")
	if _, err := os.Stat(scriptPath); err != nil {
		t.Fatalf("play-script.sh not found at %s: %v", scriptPath, err)
	}
	return scriptPath
}

// runPlayScript は play-script.sh を指定の環境変数・引数で実行して stdout/stderr/終了コードを返す。
// env は現在のプロセス環境変数に上書きマージされる。
func runPlayScript(t *testing.T, scriptPath string, env []string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	cmdArgs := append([]string{scriptPath}, args...)
	cmd := exec.Command("bash", cmdArgs...)
	cmd.Env = env
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf
	err := cmd.Run()
	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("failed to run play-script.sh: %v\nstdout:\n%s\nstderr:\n%s",
				err, stdoutBuf.String(), stderrBuf.String())
		}
		exitCode = exitErr.ExitCode()
	}
	return stdoutBuf.String(), stderrBuf.String(), exitCode
}

// makeFakeVoxActor は fake vox-actor シェルスクリプトを binDir に作成してそのパスを返す。
// 各サブコマンドの挙動は以下の環境変数で制御する:
//   - FAKE_AUDIO_CHECK_EXIT: audio-check の終了コード（省略時 0）
//   - FAKE_VIEWER_CHECK_EXIT: viewer-check の終了コード（省略時 1）
//   - FAKE_QUEUE_DIR: config path.queue の返り値
//   - FAKE_WORKSPACE_DIR: config path.workspace の返り値
//   - FAKE_ACT_EXIT: act の終了コード（省略時 0）
//   - FAKE_ACT_CALLS_LOG: act 呼び出し引数の記録ファイルパス
func makeFakeVoxActor(t *testing.T, binDir string) string {
	t.Helper()
	script := `#!/bin/bash
SUBCOMMAND="$1"
case "$SUBCOMMAND" in
  audio-check)
    exit "${FAKE_AUDIO_CHECK_EXIT:-0}"
    ;;
  viewer-check)
    exit "${FAKE_VIEWER_CHECK_EXIT:-1}"
    ;;
  config)
    case "$2" in
      path.queue) echo "$FAKE_QUEUE_DIR" ;;
      path.workspace) echo "$FAKE_WORKSPACE_DIR" ;;
    esac
    ;;
  act)
    if [ -n "$FAKE_ACT_CALLS_LOG" ]; then
      echo "$@" >> "$FAKE_ACT_CALLS_LOG"
    fi
    echo "fake-playback-id"
    exit "${FAKE_ACT_EXIT:-0}"
    ;;
  playback)
    exit 0
    ;;
esac
`
	scriptPath := filepath.Join(binDir, "vox-actor")
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake vox-actor: %v", err)
	}
	return scriptPath
}

// makePlayScriptEnv はテスト用環境変数スライスを構築する。
// fakeBinDir が "" の場合は PATH から vox-actor を除いた環境を返す。
func makePlayScriptEnv(t *testing.T, fakeBinDir string, extras map[string]string) []string {
	t.Helper()
	overrides := make(map[string]string, len(extras))
	for k, v := range extras {
		overrides[k] = v
	}
	if fakeBinDir != "" {
		overrides["PATH"] = fakeBinDir + ":" + os.Getenv("PATH")
	} else {
		// vox-actor を含まない PATH を作る（system bindir のみ）
		overrides["PATH"] = "/usr/bin:/bin"
	}
	return mergeEnv(os.Environ(), overrides)
}

// ─────────────────────────────────────────
// A. 前提条件エラー
// ─────────────────────────────────────────

func TestPlayScriptE2E_NoVoxActor_ErrorExit(t *testing.T) {
	t.Parallel()
	scriptPath := findPlayScriptPath(t)

	dir := t.TempDir()
	jsonlPath := writeTempFile(t, dir, "test.jsonl", `{"text":"テスト"}`+"\n")

	env := makePlayScriptEnv(t, "", nil)
	stdout, stderr, exitCode := runPlayScript(t, scriptPath, env, jsonlPath)

	if exitCode == 0 {
		t.Errorf("expected non-zero exit, got 0\nstdout:\n%s\nstderr:\n%s", stdout, stderr)
	}
	if !strings.Contains(stderr, "[ERROR]") {
		t.Errorf("expected stderr to contain [ERROR]\nstderr:\n%s", stderr)
	}
}

func TestPlayScriptE2E_NoArgs_ErrorExit(t *testing.T) {
	t.Parallel()
	scriptPath := findPlayScriptPath(t)

	binDir := t.TempDir()
	queueDir := t.TempDir()
	workspaceDir := t.TempDir()
	makeFakeVoxActor(t, binDir)

	env := makePlayScriptEnv(t, binDir, map[string]string{
		"FAKE_QUEUE_DIR":     queueDir,
		"FAKE_WORKSPACE_DIR": workspaceDir,
	})
	stdout, stderr, exitCode := runPlayScript(t, scriptPath, env)

	if exitCode == 0 {
		t.Errorf("expected non-zero exit, got 0\nstdout:\n%s\nstderr:\n%s", stdout, stderr)
	}
	if !strings.Contains(stdout, "[ERROR]") {
		t.Errorf("expected stdout to contain [ERROR]\nstdout:\n%s", stdout)
	}
}

func TestPlayScriptE2E_NoFile_ErrorExit(t *testing.T) {
	t.Parallel()
	scriptPath := findPlayScriptPath(t)

	binDir := t.TempDir()
	queueDir := t.TempDir()
	workspaceDir := t.TempDir()
	makeFakeVoxActor(t, binDir)

	env := makePlayScriptEnv(t, binDir, map[string]string{
		"FAKE_QUEUE_DIR":     queueDir,
		"FAKE_WORKSPACE_DIR": workspaceDir,
	})
	stdout, stderr, exitCode := runPlayScript(t, scriptPath, env, "/nonexistent/path.jsonl")

	if exitCode == 0 {
		t.Errorf("expected non-zero exit, got 0\nstdout:\n%s\nstderr:\n%s", stdout, stderr)
	}
	if !strings.Contains(stdout, "[ERROR]") {
		t.Errorf("expected stdout to contain [ERROR]\nstdout:\n%s", stdout)
	}
}

// ─────────────────────────────────────────
// B. モード切替（direct / file）
// ─────────────────────────────────────────

func TestPlayScriptE2E_AudioCheckOK_DirectMode(t *testing.T) {
	t.Parallel()
	scriptPath := findPlayScriptPath(t)

	binDir := t.TempDir()
	queueDir := t.TempDir()
	workspaceDir := t.TempDir()
	makeFakeVoxActor(t, binDir)

	dir := t.TempDir()
	jsonlPath := writeTempFile(t, dir, "test.jsonl", `{"text":"テスト"}`+"\n")

	actCallsLog := filepath.Join(t.TempDir(), "act_calls.log")
	env := makePlayScriptEnv(t, binDir, map[string]string{
		"FAKE_AUDIO_CHECK_EXIT": "0",
		"FAKE_QUEUE_DIR":        queueDir,
		"FAKE_WORKSPACE_DIR":    workspaceDir,
		"FAKE_ACT_CALLS_LOG":    actCallsLog,
	})
	stdout, stderr, exitCode := runPlayScript(t, scriptPath, env, jsonlPath)

	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d\nstdout:\n%s\nstderr:\n%s", exitCode, stdout, stderr)
	}

	// act が1回呼び出されたことを確認
	logData, err := os.ReadFile(actCallsLog)
	if err != nil {
		t.Fatalf("failed to read act calls log: %v", err)
	}
	lines := strings.Split(strings.TrimRight(string(logData), "\n"), "\n")
	if len(lines) != 1 {
		t.Errorf("expected act to be called once, got %d calls\nlog:\n%s", len(lines), string(logData))
	}

	// JSONL ファイルが削除されたことを確認
	if _, err := os.Stat(jsonlPath); !os.IsNotExist(err) {
		t.Errorf("expected JSONL file to be deleted, but it still exists: %s", jsonlPath)
	}
}

func TestPlayScriptE2E_ViewerCheckOK_DirectMode(t *testing.T) {
	t.Parallel()
	scriptPath := findPlayScriptPath(t)

	binDir := t.TempDir()
	queueDir := t.TempDir()
	workspaceDir := t.TempDir()
	makeFakeVoxActor(t, binDir)

	dir := t.TempDir()
	jsonlPath := writeTempFile(t, dir, "test.jsonl", `{"text":"テスト"}`+"\n")

	actCallsLog := filepath.Join(t.TempDir(), "act_calls.log")
	env := makePlayScriptEnv(t, binDir, map[string]string{
		"FAKE_AUDIO_CHECK_EXIT":  "1",
		"FAKE_VIEWER_CHECK_EXIT": "0",
		"FAKE_QUEUE_DIR":         queueDir,
		"FAKE_WORKSPACE_DIR":     workspaceDir,
		"FAKE_ACT_CALLS_LOG":     actCallsLog,
	})
	stdout, stderr, exitCode := runPlayScript(t, scriptPath, env, jsonlPath)

	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d\nstdout:\n%s\nstderr:\n%s", exitCode, stdout, stderr)
	}

	// act が1回呼び出されたことを確認
	logData, err := os.ReadFile(actCallsLog)
	if err != nil {
		t.Fatalf("failed to read act calls log: %v", err)
	}
	lines := strings.Split(strings.TrimRight(string(logData), "\n"), "\n")
	if len(lines) != 1 {
		t.Errorf("expected act to be called once, got %d calls\nlog:\n%s", len(lines), string(logData))
	}

	// JSONL ファイルが削除されたことを確認
	if _, err := os.Stat(jsonlPath); !os.IsNotExist(err) {
		t.Errorf("expected JSONL file to be deleted, but it still exists: %s", jsonlPath)
	}
}

func TestPlayScriptE2E_NoDevice_FileMode(t *testing.T) {
	t.Parallel()
	scriptPath := findPlayScriptPath(t)

	binDir := t.TempDir()
	queueDir := t.TempDir()
	workspaceDir := t.TempDir()
	makeFakeVoxActor(t, binDir)

	dir := t.TempDir()
	jsonlPath := writeTempFile(t, dir, "test.jsonl", `{"text":"テスト"}`+"\n")
	jsonlBase := filepath.Base(jsonlPath)

	actCallsLog := filepath.Join(t.TempDir(), "act_calls.log")
	env := makePlayScriptEnv(t, binDir, map[string]string{
		"FAKE_AUDIO_CHECK_EXIT":  "1",
		"FAKE_VIEWER_CHECK_EXIT": "1",
		"FAKE_QUEUE_DIR":         queueDir,
		"FAKE_WORKSPACE_DIR":     workspaceDir,
		"FAKE_ACT_CALLS_LOG":     actCallsLog,
	})
	stdout, stderr, exitCode := runPlayScript(t, scriptPath, env, jsonlPath)

	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d\nstdout:\n%s\nstderr:\n%s", exitCode, stdout, stderr)
	}

	// JSONL がキューディレクトリに移動したことを確認
	destPath := filepath.Join(queueDir, jsonlBase)
	if _, err := os.Stat(destPath); err != nil {
		t.Errorf("expected JSONL to be moved to queue dir %s: %v", destPath, err)
	}

	// act が呼び出されていないことを確認
	if _, err := os.Stat(actCallsLog); !os.IsNotExist(err) {
		t.Errorf("expected act NOT to be called in file mode, but calls log exists: %s", actCallsLog)
	}
}

// ─────────────────────────────────────────
// C. エラーログのローテーション
// ─────────────────────────────────────────

func TestPlayScriptE2E_LogRotation(t *testing.T) {
	t.Parallel()
	scriptPath := findPlayScriptPath(t)

	binDir := t.TempDir()
	workspaceDir := t.TempDir()
	queueDir := t.TempDir()
	makeFakeVoxActor(t, binDir)

	// workspace に 201 行のエラーログを事前作成
	errorLogPath := filepath.Join(workspaceDir, "play-script-errors.log")
	var sb strings.Builder
	for i := 0; i < 201; i++ {
		fmt.Fprintf(&sb, "pre-existing log line %d\n", i+1)
	}
	if err := os.WriteFile(errorLogPath, []byte(sb.String()), 0o644); err != nil {
		t.Fatalf("write error log: %v", err)
	}

	dir := t.TempDir()
	jsonlPath := writeTempFile(t, dir, "test.jsonl", `{"text":"テスト"}`+"\n")

	env := makePlayScriptEnv(t, binDir, map[string]string{
		"FAKE_AUDIO_CHECK_EXIT": "0",
		"FAKE_QUEUE_DIR":        queueDir,
		"FAKE_WORKSPACE_DIR":    workspaceDir,
		"FAKE_ACT_EXIT":         "1", // act を失敗させてローテーションを発生させる
	})
	stdout, stderr, exitCode := runPlayScript(t, scriptPath, env, jsonlPath)

	if exitCode == 0 {
		t.Errorf("expected non-zero exit (act failed), got 0\nstdout:\n%s\nstderr:\n%s", stdout, stderr)
	}

	// エラーログが 200 行以内に切り詰められたことを確認
	logData, err := os.ReadFile(errorLogPath)
	if err != nil {
		t.Fatalf("failed to read error log: %v", err)
	}
	lineCount := len(strings.Split(strings.TrimRight(string(logData), "\n"), "\n"))
	if lineCount > 200 {
		t.Errorf("expected error log to be truncated to <= 200 lines, got %d", lineCount)
	}
}
