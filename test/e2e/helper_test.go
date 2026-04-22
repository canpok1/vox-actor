//go:build e2e

package e2e

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
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
