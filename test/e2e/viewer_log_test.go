//go:build e2e

package e2e

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// TestViewerE2E_Log_SpeakersLoaded は VOICEVOX 接続成功時に "speakers loaded" ログが
// 出力されることを検証する。
func TestViewerE2E_Log_SpeakersLoaded(t *testing.T) {
	t.Parallel()

	fakeVox := startFakeVoicevox(t)
	port := freePort(t)
	homeDir := t.TempDir()
	workspace := t.TempDir()

	vp := startViewer(t, map[string]string{
		"HOME":                homeDir,
		"VOX_ACTOR_WORKSPACE": workspace,
		"VOX_ENGINE_URL":      fakeVox.URL,
	}, "--port", fmt.Sprintf("%d", port))

	line := vp.waitForStderr("speakers loaded", 10*time.Second)
	if !strings.Contains(line, "speakerCount") && !strings.Contains(line, "styleCount") {
		t.Errorf("expected speakers loaded log to include speaker counts\nline: %s", line)
	}

	vp.assertCleanExit(3 * time.Second)
}

// TestViewerE2E_Log_SilentModeWarn は VOICEVOX 不在時に silent モード移行 WARN ログの
// 内容が正しいことを検証する。
func TestViewerE2E_Log_SilentModeWarn(t *testing.T) {
	t.Parallel()

	port := freePort(t)
	homeDir := t.TempDir()
	workspace := t.TempDir()

	vp := startViewer(t, map[string]string{
		"HOME":                homeDir,
		"VOX_ACTOR_WORKSPACE": workspace,
		"VOX_ENGINE_URL":      "http://127.0.0.1:1",
	}, "--port", fmt.Sprintf("%d", port))

	line := vp.waitForStderr("VOICEVOX engine unreachable", 10*time.Second)

	// HumanHandler の形式: "WARN ... VOICEVOX engine unreachable, continuing in silent mode..."
	if !strings.Contains(line, "silent mode") {
		t.Errorf("expected warn log to contain 'silent mode'\nline: %s", line)
	}

	vp.assertCleanExit(3 * time.Second)
}
