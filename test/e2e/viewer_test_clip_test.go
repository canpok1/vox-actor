//go:build e2e

package e2e

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"testing"
)

// getTestClip は addr の /test-clip?speaker={speakerID} に GET してレスポンスを返す。
func getTestClip(t *testing.T, addr string, speakerID int) *http.Response {
	t.Helper()
	url := fmt.Sprintf("http://%s/test-clip?speaker=%d", addr, speakerID)
	resp, err := http.Get(url) //nolint:noctx
	if err != nil {
		t.Fatalf("GET /test-clip: %v", err)
	}
	return resp
}

func TestViewerE2E_TestClip_Success(t *testing.T) {
	t.Parallel()

	fakeVox := startFakeVoicevox(t)
	port := freePort(t)
	homeDir := t.TempDir()
	workspace := t.TempDir()

	_, addr := startViewerGetAddr(t, map[string]string{
		"HOME":                homeDir,
		"VOX_ACTOR_WORKSPACE": workspace,
		"VOX_ENGINE_URL":      fakeVox.URL,
	}, port)

	resp := getTestClip(t, addr, 3)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	if len(body) == 0 {
		t.Error("expected non-empty WAV body")
	}
}

func TestViewerE2E_TestClip_ContentType(t *testing.T) {
	t.Parallel()

	fakeVox := startFakeVoicevox(t)
	port := freePort(t)
	homeDir := t.TempDir()
	workspace := t.TempDir()

	_, addr := startViewerGetAddr(t, map[string]string{
		"HOME":                homeDir,
		"VOX_ACTOR_WORKSPACE": workspace,
		"VOX_ENGINE_URL":      fakeVox.URL,
	}, port)

	resp := getTestClip(t, addr, 3)
	defer resp.Body.Close()

	ct := resp.Header.Get("Content-Type")
	if ct != "audio/wav" {
		t.Errorf("Content-Type: expected %q, got %q", "audio/wav", ct)
	}
}

func TestViewerE2E_TestClip_ContentLength(t *testing.T) {
	t.Parallel()

	fakeVox := startFakeVoicevox(t)
	port := freePort(t)
	homeDir := t.TempDir()
	workspace := t.TempDir()

	_, addr := startViewerGetAddr(t, map[string]string{
		"HOME":                homeDir,
		"VOX_ACTOR_WORKSPACE": workspace,
		"VOX_ENGINE_URL":      fakeVox.URL,
	}, port)

	resp := getTestClip(t, addr, 3)
	defer resp.Body.Close()

	cl := resp.Header.Get("Content-Length")
	if cl == "" {
		t.Fatal("expected non-empty Content-Length header")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}

	expected := fmt.Sprintf("%d", len(body))
	if cl != expected {
		t.Errorf("Content-Length: expected %q, got %q", expected, cl)
	}
}

func TestViewerE2E_TestClip_Idempotent(t *testing.T) {
	t.Parallel()

	fakeVox := startFakeVoicevox(t)
	port := freePort(t)
	homeDir := t.TempDir()
	workspace := t.TempDir()

	_, addr := startViewerGetAddr(t, map[string]string{
		"HOME":                homeDir,
		"VOX_ACTOR_WORKSPACE": workspace,
		"VOX_ENGINE_URL":      fakeVox.URL,
	}, port)

	resp1 := getTestClip(t, addr, 3)
	body1, err := io.ReadAll(resp1.Body)
	resp1.Body.Close()
	if err != nil {
		t.Fatalf("read first response body: %v", err)
	}

	resp2 := getTestClip(t, addr, 3)
	body2, err := io.ReadAll(resp2.Body)
	resp2.Body.Close()
	if err != nil {
		t.Fatalf("read second response body: %v", err)
	}

	if !bytes.Equal(body1, body2) {
		t.Errorf("expected same body on repeated requests: len1=%d, len2=%d", len(body1), len(body2))
	}
}

func TestViewerE2E_TestClip_SilentMode(t *testing.T) {
	t.Parallel()

	port := freePort(t)
	homeDir := t.TempDir()
	workspace := t.TempDir()

	// VOICEVOX に接続できない → silent モード（speakerLookup が空になる）
	_, addr := startViewerGetAddr(t, map[string]string{
		"HOME":                homeDir,
		"VOX_ACTOR_WORKSPACE": workspace,
		"VOX_ENGINE_URL":      "http://127.0.0.1:1",
	}, port)

	resp := getTestClip(t, addr, 3)
	defer resp.Body.Close()

	// silent モードでは speakerLookup が空のため speaker not found (400)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 in silent mode, got %d", resp.StatusCode)
	}
}
