//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

type viewerProcess struct {
	t        *testing.T
	cmd      *exec.Cmd
	cancel   context.CancelFunc
	stderr   *lineAccumulator
	done     chan struct{}
	waitErr  error
	stopOnce sync.Once
}

// startViewer はビルド済み vox-actor を別プロセスで起動し、viewer サブコマンドを実行する。
func startViewer(t *testing.T, env map[string]string, args ...string) *viewerProcess {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	fullArgs := append([]string{"viewer"}, args...)
	cmd := exec.CommandContext(ctx, binaryPath, fullArgs...)
	if len(env) > 0 {
		base := os.Environ()
		for k, v := range env {
			base = append(base, k+"="+v)
		}
		cmd.Env = base
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		t.Fatalf("failed to create stderr pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		cancel()
		t.Fatalf("failed to start viewer: %v", err)
	}

	acc := &lineAccumulator{}
	go readLinesInto(stderrPipe, acc)

	vp := &viewerProcess{
		t:      t,
		cmd:    cmd,
		cancel: cancel,
		stderr: acc,
		done:   make(chan struct{}),
	}
	go func() {
		vp.waitErr = cmd.Wait()
		close(vp.done)
	}()

	t.Cleanup(func() {
		vp.stop(3 * time.Second)
	})

	return vp
}

func (v *viewerProcess) waitForStderr(needle string, timeout time.Duration) string {
	v.t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		for _, line := range v.stderr.snapshot() {
			if strings.Contains(line, needle) {
				return line
			}
		}
		select {
		case <-v.done:
			v.t.Fatalf("viewer exited before %q appeared: err=%v\nstderr:\n%s",
				needle, v.waitErr, v.stderr.String())
		default:
		}
		if time.Now().After(deadline) {
			v.t.Fatalf("timed out after %s waiting for stderr to contain %q\nstderr:\n%s",
				timeout, needle, v.stderr.String())
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func (v *viewerProcess) stop(timeout time.Duration) (exitCode int, err error) {
	v.stopOnce.Do(func() {
		_ = v.cmd.Process.Signal(syscall.SIGTERM)
		select {
		case <-v.done:
		case <-time.After(timeout):
			_ = v.cmd.Process.Kill()
			<-v.done
		}
		v.cancel()
	})
	return exitCodeFromErr(v.waitErr), v.waitErr
}

func (v *viewerProcess) assertCleanExit(timeout time.Duration) {
	v.t.Helper()
	exitCode, _ := v.stop(timeout)
	if exitCode != 0 {
		v.t.Fatalf("expected clean exit (0), got %d\nstderr:\n%s",
			exitCode, v.stderr.String())
	}
}

// freePort は使用可能なTCPポート番号を返す。
func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to get free port: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return port
}

// extractAddrFromLog は "stream server listening" ログ行からアドレスを抽出する。
// HumanHandler の形式: "... stream server listening (addr=127.0.0.1:8080, silent=...)"
func extractAddrFromLog(line string) string {
	for _, field := range strings.Fields(line) {
		f := strings.TrimPrefix(field, "(")
		f = strings.TrimSuffix(f, ")")
		f = strings.TrimSuffix(f, ",")
		if strings.HasPrefix(f, "addr=") {
			return strings.TrimPrefix(f, "addr=")
		}
	}
	return ""
}

func TestViewerE2E_StatusEndpoint(t *testing.T) {
	t.Parallel()

	port := freePort(t)
	vp := startViewer(t, nil, "--port", fmt.Sprintf("%d", port))

	line := vp.waitForStderr("stream server listening", 10*time.Second)
	addr := extractAddrFromLog(line)
	if addr == "" {
		t.Fatalf("failed to extract addr from log line: %q", line)
	}

	url := fmt.Sprintf("http://%s/api/status", addr)
	var resp *http.Response
	var err error
	for i := 0; i < 20; i++ {
		resp, err = http.Get(url) //nolint:noctx
		if err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("failed to GET /api/status: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("expected valid JSON response, got: %s", body)
	}
	if _, ok := result["silent"]; !ok {
		t.Errorf("expected 'silent' field in /api/status response, got: %s", body)
	}

	vp.assertCleanExit(3 * time.Second)
}

func TestViewerE2E_WithWatchDir_WatchesDirectory(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	port := freePort(t)
	workspace := t.TempDir()

	vp := startViewer(t, map[string]string{"VOX_ACTOR_WORKSPACE": workspace},
		"--port", fmt.Sprintf("%d", port),
		"--watch", dir,
	)

	vp.waitForStderr("watching directory", 10*time.Second)
	vp.assertCleanExit(3 * time.Second)
}

func TestViewerE2E_WatchQueue_StartsWatching(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	port := freePort(t)

	vp := startViewer(t,
		map[string]string{"VOX_ACTOR_WORKSPACE": workspace},
		"--port", fmt.Sprintf("%d", port),
		"--watch-queue",
	)

	vp.waitForStderr("watching directory", 10*time.Second)

	// ResolveQueuePath: VOX_ACTOR_WORKSPACE/queue
	queueDir := workspace + "/queue"
	info, err := os.Stat(queueDir)
	if err != nil {
		t.Fatalf("expected queue dir to be auto-created at %s: %v", queueDir, err)
	}
	if !info.IsDir() {
		t.Fatalf("expected %s to be a directory", queueDir)
	}

	vp.assertCleanExit(3 * time.Second)
}

func TestViewerE2E_InvalidPort_ReturnsUsageError(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		port string
	}{
		{"port 0", "0"},
		{"port 99999", "99999"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, stderr, exitCode := runCLI(t, nil, "viewer", "--port", tc.port)
			if exitCode != 2 {
				t.Fatalf("expected exit code 2 (usage error) for port %s, got %d\nstderr:\n%s",
					tc.port, exitCode, stderr)
			}
			if strings.TrimSpace(stderr) == "" {
				t.Error("expected non-empty stderr for usage error")
			}
		})
	}
}

func TestViewerE2E_WatchNonDirectory_ReturnsUsageError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	file := writeTempFile(t, dir, "notadir.txt", "hello")

	_, stderr, exitCode := runCLI(t, nil, "viewer", "--watch", file)
	if exitCode != 2 {
		t.Fatalf("expected exit code 2 for non-directory --watch, got %d\nstderr:\n%s", exitCode, stderr)
	}
}
