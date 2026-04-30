//go:build e2e

package e2e

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// TestViewerE2E_Help_ExitZero は `viewer --help` が exit 0 で終了し、
// 主要フラグが表示されることを検証する。
func TestViewerE2E_Help_ExitZero(t *testing.T) {
	t.Parallel()

	stdout, _, exitCode := runCLI(t, nil, "viewer", "--help")

	if exitCode != 0 {
		t.Fatalf("expected exit code 0 for viewer --help, got %d", exitCode)
	}

	wantFlags := []string{"--port", "--host", "--engine-url", "--speaker"}
	for _, flag := range wantFlags {
		if !strings.Contains(stdout, flag) {
			t.Errorf("expected --help output to contain flag %q\nstdout:\n%s", flag, stdout)
		}
	}

	noWantFlags := []string{"--watch", "--watch-queue", "--delete"}
	for _, flag := range noWantFlags {
		if strings.Contains(stdout, flag) {
			t.Errorf("expected --help output to NOT contain flag %q\nstdout:\n%s", flag, stdout)
		}
	}
}

// TestViewerE2E_Env_VOXEngineURL_Applied は VOX_ENGINE_URL 環境変数で指定したサーバーに
// viewer が接続することを検証する（speakers loaded ログで確認）。
func TestViewerE2E_Env_VOXEngineURL_Applied(t *testing.T) {
	t.Parallel()

	fakeVox := startFakeVoicevox(t)
	port := freePort(t)
	homeDir := t.TempDir()
	workspace := t.TempDir()

	// VOX_ENGINE_URL を環境変数で指定（--engine-url フラグは使わない）
	vp := startViewer(t, map[string]string{
		"HOME":                homeDir,
		"VOX_ACTOR_WORKSPACE": workspace,
		"VOX_ENGINE_URL":      fakeVox.URL,
	}, "--port", fmt.Sprintf("%d", port))

	// VOICEVOX に接続成功 → "speakers loaded" ログが出る
	vp.waitForStderr("speakers loaded", 10*time.Second)

	vp.assertCleanExit(3 * time.Second)
}

// TestViewerE2E_Host_0000_Binds は --host 0.0.0.0 を指定したとき
// 127.0.0.1 でも接続できることを検証する。
// Go の net.Listen("tcp", "0.0.0.0:N") は IPv6 dual-stack では [::] として表示されるため
// ログ内の addr 表現は 0.0.0.0 または [::] のいずれかを許容する。
func TestViewerE2E_Host_0000_Binds(t *testing.T) {
	t.Parallel()

	port := freePort(t)
	homeDir := t.TempDir()
	workspace := t.TempDir()

	vp := startViewer(t, map[string]string{
		"HOME":                homeDir,
		"VOX_ACTOR_WORKSPACE": workspace,
		"VOX_ENGINE_URL":      "http://127.0.0.1:1",
	}, "--port", fmt.Sprintf("%d", port), "--host", "0.0.0.0")

	vp.waitForStderr("stream server listening", 10*time.Second)

	// 127.0.0.1 でアクセス可能か確認（0.0.0.0 バインドの主目的）
	url := fmt.Sprintf("http://127.0.0.1:%d/api/status", port)
	resp := getURLWithRetry(t, url)
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("expected 200 from 127.0.0.1 with --host 0.0.0.0, got %d", resp.StatusCode)
	}

	vp.assertCleanExit(3 * time.Second)
}
