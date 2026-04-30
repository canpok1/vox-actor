//go:build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestViewerE2E_SilentMode_WarnLog(t *testing.T) {
	t.Parallel()

	port := freePort(t)
	homeDir := t.TempDir()
	workspace := t.TempDir()

	vp := startViewer(t, map[string]string{
		"HOME":                homeDir,
		"VOX_ACTOR_WORKSPACE": workspace,
		"VOX_ENGINE_URL":      "http://127.0.0.1:1",
	}, "--port", fmt.Sprintf("%d", port))

	vp.waitForStderr("VOICEVOX engine unreachable", 10*time.Second)

	listenLine := vp.waitForStderr("stream server listening", 10*time.Second)
	if !strings.Contains(listenLine, "silent=true") {
		t.Errorf("expected silent=true in listening log\nline: %s", listenLine)
	}

	vp.assertCleanExit(3 * time.Second)
}

func TestViewerE2E_SilentMode_StatusFields(t *testing.T) {
	t.Parallel()

	port := freePort(t)
	homeDir := t.TempDir()
	workspace := t.TempDir()

	vp, addr := startViewerGetAddr(t, map[string]string{
		"HOME":                homeDir,
		"VOX_ACTOR_WORKSPACE": workspace,
		"VOX_ENGINE_URL":      "http://127.0.0.1:1",
	}, port)

	resp := getURLWithRetry(t, fmt.Sprintf("http://%s/api/status", addr))
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	var result struct {
		Silent       bool   `json:"silent"`
		SilentReason string `json:"silentReason"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("unmarshal: %v\nbody: %s", err, body)
	}
	if !result.Silent {
		t.Errorf("expected silent=true in /api/status, got false\nbody: %s", body)
	}
	if result.SilentReason == "" {
		t.Errorf("expected non-empty silentReason in /api/status\nbody: %s", body)
	}

	vp.assertCleanExit(3 * time.Second)
}
