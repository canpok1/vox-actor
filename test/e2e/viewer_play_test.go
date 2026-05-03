//go:build e2e

package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// startFakeVoicevoxCreateQueryFails は /audio_query を 500 で返す fake VOICEVOX サーバーを返す。
func startFakeVoicevoxCreateQueryFails(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/speakers", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"name":"ずんだもん","styles":[{"name":"ノーマル","id":3}]}]`))
	})
	mux.HandleFunc("/audio_query", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	})
	mux.HandleFunc("/synthesis", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "audio/wav")
		_, _ = w.Write(bytes.Repeat([]byte{0x00}, 64))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// startFakeVoicevoxSynthesisFails は /synthesis を 500 で返す fake VOICEVOX サーバーを返す。
func startFakeVoicevoxSynthesisFails(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/speakers", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"name":"ずんだもん","styles":[{"name":"ノーマル","id":3}]}]`))
	})
	mux.HandleFunc("/audio_query", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"accent_phrases":[],"speedScale":1.0,"pitchScale":0.0,"intonationScale":1.0,"volumeScale":1.0,"prePhonemeLength":0.1,"postPhonemeLength":0.1,"outputSamplingRate":24000,"outputStereo":false,"kana":""}`))
	})
	mux.HandleFunc("/synthesis", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestViewerE2E_APIPlay_Success(t *testing.T) {
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

	resp := postAPIPlay(t, addr, "テスト", 3)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	ct := resp.Header.Get("Content-Type")
	if ct != "application/json; charset=utf-8" {
		t.Errorf("Content-Type: expected %q, got %q", "application/json; charset=utf-8", ct)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	var result struct {
		Silent bool `json:"silent"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("decode response: %v\nbody: %s", err, body)
	}
	if result.Silent {
		t.Error("expected silent=false in normal mode response")
	}
}

func TestViewerE2E_APIPlay_SilentModeResponse(t *testing.T) {
	t.Parallel()

	port := freePort(t)
	homeDir := t.TempDir()
	workspace := t.TempDir()

	_, addr := startViewerGetAddr(t, map[string]string{
		"HOME":                homeDir,
		"VOX_ACTOR_WORKSPACE": workspace,
		"VOX_ENGINE_URL":      "http://127.0.0.1:1",
	}, port)

	resp := postAPIPlay(t, addr, "サイレントテスト", 3)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	ct := resp.Header.Get("Content-Type")
	if ct != "application/json; charset=utf-8" {
		t.Errorf("Content-Type: expected %q, got %q", "application/json; charset=utf-8", ct)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	var result struct {
		Silent       bool   `json:"silent"`
		SilentReason string `json:"silent_reason"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("decode response: %v\nbody: %s", err, body)
	}
	if !result.Silent {
		t.Error("expected silent=true in silent mode response")
	}
	if result.SilentReason == "" {
		t.Error("expected non-empty silentReason in silent mode response")
	}
}

func TestViewerE2E_APIPlay_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	port := freePort(t)
	homeDir := t.TempDir()
	workspace := t.TempDir()

	_, addr := startViewerGetAddr(t, map[string]string{
		"HOME":                homeDir,
		"VOX_ACTOR_WORKSPACE": workspace,
		"VOX_ENGINE_URL":      "http://127.0.0.1:1",
	}, port)

	url := fmt.Sprintf("http://%s/api/play", addr)
	resp, err := http.Get(url) //nolint:noctx
	if err != nil {
		t.Fatalf("GET /api/play: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", resp.StatusCode)
	}
}

func TestViewerE2E_APIPlay_InvalidJSON(t *testing.T) {
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

	url := fmt.Sprintf("http://%s/api/play", addr)
	resp, err := http.Post(url, "application/json", bytes.NewBufferString(`not-json`)) //nolint:noctx
	if err != nil {
		t.Fatalf("POST /api/play: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestViewerE2E_APIPlay_CreateQueryFails(t *testing.T) {
	t.Parallel()

	fakeVox := startFakeVoicevoxCreateQueryFails(t)
	port := freePort(t)
	homeDir := t.TempDir()
	workspace := t.TempDir()

	_, addr := startViewerGetAddr(t, map[string]string{
		"HOME":                homeDir,
		"VOX_ACTOR_WORKSPACE": workspace,
		"VOX_ENGINE_URL":      fakeVox.URL,
	}, port)

	resp := postAPIPlay(t, addr, "テスト", 3)
	defer resp.Body.Close()

	// 非同期 worker 化により、リクエストは即座に 200 で返る。
	// worker が synthesize エラーを検知し playback state を failed に更新する。
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 (async), got %d", resp.StatusCode)
	}
}

func TestViewerE2E_APIPlay_SynthesisFails(t *testing.T) {
	t.Parallel()

	fakeVox := startFakeVoicevoxSynthesisFails(t)
	port := freePort(t)
	homeDir := t.TempDir()
	workspace := t.TempDir()

	vp, addr := startViewerGetAddr(t, map[string]string{
		"HOME":                homeDir,
		"VOX_ACTOR_WORKSPACE": workspace,
		"VOX_ENGINE_URL":      fakeVox.URL,
	}, port)

	resp := postAPIPlay(t, addr, "テスト", 3)
	defer resp.Body.Close()

	// 非同期 worker 化により、リクエストは即座に 200 で返る。
	// worker が synthesize エラーを検知し playback state を failed に更新する。
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 (async), got %d", resp.StatusCode)
	}

	vp.assertCleanExit(3 * time.Second)
}
