//go:build e2e

package e2e

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// runCLI はビルド済みの vox-actor バイナリを指定の引数・環境変数で実行し、
// stdout / stderr / 終了コードを返す。
//
// env に指定したキーは現在のプロセスの環境変数に上書きされる形で渡される。
// env が nil または空の場合は現在の環境変数をそのまま引き継ぐ。
// 呼び出し側は終了コード以外の失敗（バイナリが見つからない等）に対しては
// t.Fatalf で停止する想定。
func runCLI(t *testing.T, env map[string]string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()

	cmd := exec.Command(binaryPath, args...)
	if len(env) > 0 {
		base := os.Environ()
		for k, v := range env {
			base = append(base, k+"="+v)
		}
		cmd.Env = base
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err := cmd.Run()
	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("failed to run CLI: %v\nstdout:\n%s\nstderr:\n%s",
				err, stdoutBuf.String(), stderrBuf.String())
		}
		exitCode = exitErr.ExitCode()
	}

	return stdoutBuf.String(), stderrBuf.String(), exitCode
}

// extractLineContaining は s の中から needle を含む最初の行を返す。
// 見つからない場合は t.Fatalf でテストを停止する。
func extractLineContaining(t *testing.T, s, needle string) string {
	t.Helper()
	for _, line := range strings.Split(s, "\n") {
		if strings.Contains(line, needle) {
			return line
		}
	}
	t.Fatalf("expected output to contain a line with %q\noutput:\n%s", needle, s)
	return ""
}

// writeTempFile は dir 配下に name のファイルを作成して絶対パスを返す。
func writeTempFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
	return path
}

// countNonEmptyLines は s の空行を除いた行数を返す。
func countNonEmptyLines(s string) int {
	n := 0
	for _, line := range strings.Split(s, "\n") {
		if strings.TrimSpace(line) != "" {
			n++
		}
	}
	return n
}
