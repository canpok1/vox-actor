//go:build e2e

package e2e

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
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
	return runCLIInDir(t, "", env, args...)
}

// runCLIInDir は runCLI と同等だが、子プロセスのワーキングディレクトリを
// dir に設定して実行する。dir が空文字列の場合は現在のワーキングディレクトリを
// そのまま引き継ぐ（runCLI と同じ挙動）。
//
// config サブコマンドのように gitリポジトリ有無でパス解決が変わるケースを
// 明示的に制御したいテストから利用する。
func runCLIInDir(t *testing.T, dir string, env map[string]string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()

	cmd := exec.Command(binaryPath, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	if len(env) > 0 {
		cmd.Env = mergeEnv(os.Environ(), env)
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

// mergeEnv は base の env スライスに overrides を上書きマージして返す。
// overrides に含まれるキーは base から除去してから末尾に追加するため、確実に上書きされる。
func mergeEnv(base []string, overrides map[string]string) []string {
	keys := make(map[string]bool, len(overrides))
	for k := range overrides {
		keys[k] = true
	}
	filtered := make([]string, 0, len(base))
	for _, e := range base {
		key, _, _ := strings.Cut(e, "=")
		if !keys[key] {
			filtered = append(filtered, e)
		}
	}
	for k, v := range overrides {
		filtered = append(filtered, k+"="+v)
	}
	return filtered
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

// getURLWithRetry は url に GET して成功するまで最大 20 回リトライする。
func getURLWithRetry(t *testing.T, url string) *http.Response {
	t.Helper()
	var (
		resp   *http.Response
		getErr error
	)
	for i := 0; i < 20; i++ {
		resp, getErr = http.Get(url) //nolint:noctx
		if getErr == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if getErr != nil {
		t.Fatalf("GET %s: %v", url, getErr)
	}
	return resp
}

// waitForSSEClipEvent は resp から SSE clip イベントが届くまで最大 timeout 待機して返す。
func waitForSSEClipEvent(t *testing.T, resp *http.Response, timeout time.Duration) sseEvent {
	t.Helper()
	ch := make(chan sseEvent, 1)
	go func() {
		r := bufio.NewReader(resp.Body)
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		for {
			ev, err := readSSEEvent(ctx, r)
			if err != nil {
				return
			}
			if ev.EventType == "clip" {
				ch <- ev
				return
			}
		}
	}()
	select {
	case ev := <-ch:
		return ev
	case <-time.After(timeout):
		t.Fatalf("timeout (%s) waiting for SSE clip event", timeout)
		return sseEvent{}
	}
}

// assertSSEClipText は SSE clip イベントの data.text が want と一致することを検証する。
func assertSSEClipText(t *testing.T, ev sseEvent, want string) {
	t.Helper()
	var data struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal([]byte(ev.Data), &data); err != nil {
		t.Fatalf("unmarshal clip data: %v\nraw: %s", err, ev.Data)
	}
	if data.Text != want {
		t.Errorf("clip text: expected %q, got %q", want, data.Text)
	}
}

// waitForFileInDir は dir 内に prefix で始まるファイルが現れるまで最大 timeout 待機して
// そのパスを返す。タイムアウト時は t.Errorf を呼ぶ。
func waitForFileInDir(t *testing.T, dir, prefix string, timeout time.Duration) string {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		entries, _ := os.ReadDir(dir)
		for _, e := range entries {
			if strings.HasPrefix(e.Name(), prefix) {
				return filepath.Join(dir, e.Name())
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Errorf("timeout (%s): no file with prefix %q found in %s\n%s", timeout, prefix, dir, dumpDir(t, dir))
	return ""
}

// waitForFileGone は path が消えるまで最大 timeout 待機する。タイムアウト時は t.Errorf を呼ぶ。
func waitForFileGone(t *testing.T, path string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Errorf("timeout (%s): file still exists: %s", timeout, path)
}

// dumpDir はディレクトリ内のファイル一覧を文字列で返す（デバッグ用）。
func dumpDir(t *testing.T, dir string) string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Sprintf("(ReadDir error: %v)", err)
	}
	var sb strings.Builder
	for _, e := range entries {
		fmt.Fprintf(&sb, "%s\n", e.Name())
		if e.IsDir() {
			sub, _ := os.ReadDir(filepath.Join(dir, e.Name()))
			for _, s := range sub {
				fmt.Fprintf(&sb, "  %s/%s\n", e.Name(), s.Name())
			}
		}
	}
	return sb.String()
}

// writeViewerLockFile は homeDir 配下に viewer ロックファイルを書き込む。
// addr は "host:port" 形式で指定する。
// ロックファイルパスは $HOME/.vox-actor/viewer/viewer.lock。
func writeViewerLockFile(t *testing.T, homeDir, addr string) {
	t.Helper()
	lockDir := filepath.Join(homeDir, ".vox-actor", "viewer")
	if err := os.MkdirAll(lockDir, 0o755); err != nil {
		t.Fatalf("MkdirAll lock dir: %v", err)
	}
	lockPath := filepath.Join(lockDir, "viewer.lock")
	content := fmt.Sprintf("pid=%d\nstarted_at=%s\naddr=%s\n",
		os.Getpid(), time.Now().Format(time.RFC3339), addr)
	if err := os.WriteFile(lockPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write lock file: %v", err)
	}
}
