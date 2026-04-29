//go:build e2e

package e2e

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"
)

// getClipURLFromSSE は SSE 接続後に /api/play を呼び出し、clip イベントの URL を返す。
func getClipURLFromSSE(t *testing.T, addr string) string {
	t.Helper()

	sseResp := connectSSE(t, addr)
	defer sseResp.Body.Close()

	// handleEvents は WriteHeader+Flush の後で subscriber を登録するため、
	// POST 前に少し待って登録完了を確保する。
	time.Sleep(100 * time.Millisecond)

	eventCh := make(chan sseEvent, 1)
	go func() {
		r := bufio.NewReader(sseResp.Body)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		for {
			ev, err := readSSEEvent(ctx, r)
			if err != nil {
				return
			}
			if ev.EventType == "clip" {
				eventCh <- ev
				return
			}
		}
	}()

	playResp := postAPIPlay(t, addr, "テスト発話", 3)
	defer playResp.Body.Close()
	if playResp.StatusCode != http.StatusOK {
		t.Fatalf("POST /api/play expected 200, got %d", playResp.StatusCode)
	}

	select {
	case ev := <-eventCh:
		var data struct {
			URL string `json:"url"`
		}
		if err := json.Unmarshal([]byte(ev.Data), &data); err != nil {
			t.Fatalf("unmarshal clip event: %v\nraw: %s", err, ev.Data)
		}
		if data.URL == "" {
			t.Fatal("expected non-empty URL in clip event")
		}
		return data.URL
	case <-time.After(10 * time.Second):
		t.Fatal("timeout waiting for clip SSE event")
		return ""
	}
}

// getClip は addr の clipURL に GET してレスポンスを返す。
func getClip(t *testing.T, addr, clipURL string) *http.Response {
	t.Helper()
	url := fmt.Sprintf("http://%s%s", addr, clipURL)
	resp, err := http.Get(url) //nolint:noctx
	if err != nil {
		t.Fatalf("GET %s: %v", clipURL, err)
	}
	return resp
}

func TestViewerE2E_Clip_Success(t *testing.T) {
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

	clipURL := getClipURLFromSSE(t, addr)
	resp := getClip(t, addr, clipURL)
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

func TestViewerE2E_Clip_ContentType(t *testing.T) {
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

	clipURL := getClipURLFromSSE(t, addr)
	resp := getClip(t, addr, clipURL)
	defer resp.Body.Close()

	ct := resp.Header.Get("Content-Type")
	if ct != "audio/wav" {
		t.Errorf("Content-Type: expected %q, got %q", "audio/wav", ct)
	}
}

func TestViewerE2E_Clip_CacheControl(t *testing.T) {
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

	clipURL := getClipURLFromSSE(t, addr)
	resp := getClip(t, addr, clipURL)
	defer resp.Body.Close()

	const want = "public, max-age=31536000, immutable"
	cc := resp.Header.Get("Cache-Control")
	if cc != want {
		t.Errorf("Cache-Control: expected %q, got %q", want, cc)
	}
}

func TestViewerE2E_Clip_InvalidID_404(t *testing.T) {
	t.Parallel()

	port := freePort(t)
	homeDir := t.TempDir()
	workspace := t.TempDir()

	_, addr := startViewerGetAddr(t, map[string]string{
		"HOME":                homeDir,
		"VOX_ACTOR_WORKSPACE": workspace,
		"VOX_ENGINE_URL":      "http://127.0.0.1:1",
	}, port)

	url := fmt.Sprintf("http://%s/clips/not-a-timestamp.wav", addr)
	resp, err := http.Get(url) //nolint:noctx
	if err != nil {
		t.Fatalf("GET /clips/not-a-timestamp.wav: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestViewerE2E_Clip_NonExistent_404(t *testing.T) {
	t.Parallel()

	port := freePort(t)
	homeDir := t.TempDir()
	workspace := t.TempDir()

	_, addr := startViewerGetAddr(t, map[string]string{
		"HOME":                homeDir,
		"VOX_ACTOR_WORKSPACE": workspace,
		"VOX_ENGINE_URL":      "http://127.0.0.1:1",
	}, port)

	// 存在しない timestamp（1 = 非常に過去）で 404 を期待する
	url := fmt.Sprintf("http://%s/clips/1.wav", addr)
	resp, err := http.Get(url) //nolint:noctx
	if err != nil {
		t.Fatalf("GET /clips/1.wav: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestViewerE2E_Clip_RestartCollisionAvoidance(t *testing.T) {
	t.Parallel()

	fakeVox := startFakeVoicevox(t)

	// 1回目の起動
	port1 := freePort(t)
	homeDir1 := t.TempDir()
	workspace1 := t.TempDir()

	vp1, addr1 := startViewerGetAddr(t, map[string]string{
		"HOME":                homeDir1,
		"VOX_ACTOR_WORKSPACE": workspace1,
		"VOX_ENGINE_URL":      fakeVox.URL,
	}, port1)

	url1 := getClipURLFromSSE(t, addr1)
	_, _ = vp1.stop(3 * time.Second)

	// 2回目の起動（別インスタンス）
	port2 := freePort(t)
	homeDir2 := t.TempDir()
	workspace2 := t.TempDir()

	_, addr2 := startViewerGetAddr(t, map[string]string{
		"HOME":                homeDir2,
		"VOX_ACTOR_WORKSPACE": workspace2,
		"VOX_ENGINE_URL":      fakeVox.URL,
	}, port2)

	url2 := getClipURLFromSSE(t, addr2)

	if url1 == url2 {
		t.Errorf("expected different clip URLs after restart, both returned: %s", url1)
	}
}
