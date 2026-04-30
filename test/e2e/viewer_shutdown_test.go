//go:build e2e

package e2e

import (
	"syscall"
	"testing"
	"time"
)

// TestViewerE2E_SIGINT_GracefulShutdown は SIGINT 受信時に viewer が
// graceful shutdown（exit code 0）することを検証する。
func TestViewerE2E_SIGINT_GracefulShutdown(t *testing.T) {
	t.Parallel()

	port := freePort(t)
	homeDir := t.TempDir()
	workspace := t.TempDir()

	vp, _ := startViewerGetAddr(t, map[string]string{
		"HOME":                homeDir,
		"VOX_ACTOR_WORKSPACE": workspace,
		"VOX_ENGINE_URL":      "http://127.0.0.1:1",
	}, port)

	// SIGINT を送信
	if err := vp.cmd.Process.Signal(syscall.SIGINT); err != nil {
		t.Fatalf("failed to send SIGINT: %v", err)
	}

	select {
	case <-vp.done:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout: viewer did not exit after SIGINT")
	}

	exitCode := exitCodeFromErr(vp.waitErr)
	if exitCode != 0 {
		t.Errorf("expected exit code 0 after SIGINT, got %d\nstderr:\n%s",
			exitCode, vp.stderr.String())
	}
}

// TestViewerE2E_SIGTERM_GracefulShutdown は SIGTERM 受信時に viewer が
// graceful shutdown（exit code 0）することを検証する。
func TestViewerE2E_SIGTERM_GracefulShutdown(t *testing.T) {
	t.Parallel()

	port := freePort(t)
	homeDir := t.TempDir()
	workspace := t.TempDir()

	vp, _ := startViewerGetAddr(t, map[string]string{
		"HOME":                homeDir,
		"VOX_ACTOR_WORKSPACE": workspace,
		"VOX_ENGINE_URL":      "http://127.0.0.1:1",
	}, port)

	vp.assertCleanExit(3 * time.Second)
}
