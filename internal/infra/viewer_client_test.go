package infra_test

// テストリスト: viewer 検出 / ViewerAPIClient
//
// 受け入れ条件 vs テスト関数のマッピング:
// - lock ファイルなし → DetectViewer が false を返す               → TestDetectViewer_NoLockFile
// - lock に addr なし → DetectViewer が false を返す               → TestDetectViewer_NoAddr
// - lock に addr あり HTTP 疎通 OK → DetectViewer が addr と true  → TestDetectViewer_Running
// - lock に addr あり HTTP 疎通失敗 → DetectViewer が false を返す → TestDetectViewer_HTTPFails
// - POST /api/play 成功 → ViewerAPIClient.Play が PlayResponse を返す      → TestViewerAPIClient_Play_Success
// - POST /api/play 無音モード応答 → ViewerAPIClient.Play が silent=true    → TestViewerAPIClient_Play_Silent
// - POST /api/play で非 200 応答 → ViewerAPIClient.Play がエラーを返す     → TestViewerAPIClient_Play_Error
// - POST /api/play で 400 + body → エラーメッセージに body を含む         → TestViewerAPIClient_Play_ErrorWithBody
// #481 --viewer-url:
// - NewViewerAPIClientFromURL でフル URL 指定 → POST /api/play が成功する  → TestViewerAPIClientFromURL_Play_Success
// - NewViewerAPIClientFromURL でフル URL に trailing slash → POST が成功   → TestViewerAPIClientFromURL_Play_TrailingSlash
// - GET /api/playback/{id} で非 200 + body → エラーメッセージに body を含む → TestViewerAPIClient_GetPlayback_ErrorWithBody

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/canpok1/vox-actor/internal/infra"
)

func TestDetectViewer_NoLockFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	lockPath := filepath.Join(dir, "viewer.lock")

	_, ok := infra.DetectViewer(lockPath)
	if ok {
		t.Error("expected DetectViewer to return false when lock file is missing")
	}
}

func TestDetectViewer_NoAddr(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	lockPath := filepath.Join(dir, "viewer.lock")

	vl, err := infra.AcquireViewerLock(lockPath)
	if err != nil {
		t.Fatalf("AcquireViewerLock: %v", err)
	}
	defer vl.Release() //nolint:errcheck

	_, ok := infra.DetectViewer(lockPath)
	if ok {
		t.Error("expected DetectViewer to return false when addr is empty")
	}
}

func TestDetectViewer_Running(t *testing.T) {
	t.Parallel()
	// Start a real HTTP test server that responds to /api/status
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/status" {
			w.WriteHeader(http.StatusOK)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	dir := t.TempDir()
	lockPath := filepath.Join(dir, "viewer.lock")

	vl, err := infra.AcquireViewerLock(lockPath)
	if err != nil {
		t.Fatalf("AcquireViewerLock: %v", err)
	}
	defer vl.Release() //nolint:errcheck

	// srv.Listener.Addr().String() returns e.g. "127.0.0.1:XXXXX"
	addr := srv.Listener.Addr().String()
	if err := vl.WriteAddr(addr); err != nil {
		t.Fatalf("WriteAddr: %v", err)
	}

	got, ok := infra.DetectViewer(lockPath)
	if !ok {
		t.Fatal("expected DetectViewer to return true when viewer is running")
	}
	if got != addr {
		t.Errorf("expected addr=%s, got %s", addr, got)
	}
}

func TestDetectViewer_HTTPFails(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	lockPath := filepath.Join(dir, "viewer.lock")

	vl, err := infra.AcquireViewerLock(lockPath)
	if err != nil {
		t.Fatalf("AcquireViewerLock: %v", err)
	}
	defer vl.Release() //nolint:errcheck

	// Use an addr that is not listening
	if err := vl.WriteAddr("127.0.0.1:1"); err != nil {
		t.Fatalf("WriteAddr: %v", err)
	}

	_, ok := infra.DetectViewer(lockPath)
	if ok {
		t.Error("expected DetectViewer to return false when HTTP fails")
	}
}

func TestViewerAPIClient_Play_Success(t *testing.T) {
	t.Parallel()
	var capturedReq infra.ViewerPlayRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&capturedReq)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"silent": false})
	}))
	defer srv.Close()

	speed := 1.2
	pitch := 0.05
	intonation := 1.3
	client := infra.NewViewerAPIClient(srv.Listener.Addr().String())
	req := infra.ViewerPlayRequest{Clips: []infra.ViewerClip{{
		Text:       "こんにちは",
		SpeakerID:  2,
		Speed:      &speed,
		Pitch:      &pitch,
		Intonation: &intonation,
	}}}
	resp, err := client.Play(t.Context(), req)
	if err != nil {
		t.Fatalf("Play: %v", err)
	}
	if resp.Silent {
		t.Errorf("expected silent=false, got true")
	}
	if len(capturedReq.Clips) == 0 {
		t.Fatalf("expected clips to be captured")
	}
	clip := capturedReq.Clips[0]
	if clip.Text != "こんにちは" {
		t.Errorf("expected text=%q, got %q", "こんにちは", clip.Text)
	}
	if clip.SpeakerID != 2 {
		t.Errorf("expected speaker_id=2, got %d", clip.SpeakerID)
	}
	if clip.Speed == nil || *clip.Speed != speed {
		t.Errorf("expected speed=%v, got %v", speed, clip.Speed)
	}
	if clip.Pitch == nil || *clip.Pitch != pitch {
		t.Errorf("expected pitch=%v, got %v", pitch, clip.Pitch)
	}
	if clip.Intonation == nil || *clip.Intonation != intonation {
		t.Errorf("expected intonation=%v, got %v", intonation, clip.Intonation)
	}
}

func TestViewerAPIClient_Play_Silent(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"silent":        true,
			"silent_reason": "VOICEVOX接続失敗",
		})
	}))
	defer srv.Close()

	client := infra.NewViewerAPIClient(srv.Listener.Addr().String())
	resp, err := client.Play(t.Context(), infra.ViewerPlayRequest{Clips: []infra.ViewerClip{{Text: "test", SpeakerID: 2}}})
	if err != nil {
		t.Fatalf("Play: %v", err)
	}
	if !resp.Silent {
		t.Errorf("expected silent=true")
	}
	if resp.SilentReason == "" {
		t.Errorf("expected non-empty SilentReason")
	}
}

func TestViewerAPIClient_Play_Error(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "synthesis failed", http.StatusBadGateway)
	}))
	defer srv.Close()

	client := infra.NewViewerAPIClient(srv.Listener.Addr().String())
	_, err := client.Play(t.Context(), infra.ViewerPlayRequest{Clips: []infra.ViewerClip{{Text: "test", SpeakerID: 2}}})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestViewerAPIClient_Play_ErrorWithBody(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("clips[0].speaker_id: unknown speaker id 9999"))
	}))
	defer srv.Close()

	client := infra.NewViewerAPIClient(srv.Listener.Addr().String())
	_, err := client.Play(t.Context(), infra.ViewerPlayRequest{Clips: []infra.ViewerClip{{Text: "test", SpeakerID: 9999}}})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "unknown speaker id 9999") {
		t.Errorf("expected error to contain body, got: %v", err)
	}
}

func TestViewerAPIClientFromURL_Play_Success(t *testing.T) {
	t.Parallel()
	var capturedReq infra.ViewerPlayRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&capturedReq)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"silent": false})
	}))
	defer srv.Close()

	client := infra.NewViewerAPIClientFromURL("http://" + srv.Listener.Addr().String())
	resp, err := client.Play(t.Context(), infra.ViewerPlayRequest{Clips: []infra.ViewerClip{{Text: "テスト", SpeakerID: 3}}})
	if err != nil {
		t.Fatalf("Play: %v", err)
	}
	if resp.Silent {
		t.Errorf("expected silent=false")
	}
	if len(capturedReq.Clips) == 0 || capturedReq.Clips[0].Text != "テスト" {
		t.Errorf("expected text=%q, got %q", "テスト", func() string {
			if len(capturedReq.Clips) > 0 {
				return capturedReq.Clips[0].Text
			}
			return ""
		}())
	}
}

func TestViewerAPIClientFromURL_Play_TrailingSlash(t *testing.T) {
	t.Parallel()
	var called bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"silent": false})
	}))
	defer srv.Close()

	client := infra.NewViewerAPIClientFromURL("http://" + srv.Listener.Addr().String() + "/")
	_, err := client.Play(t.Context(), infra.ViewerPlayRequest{Clips: []infra.ViewerClip{{Text: "テスト", SpeakerID: 3}}})
	if err != nil {
		t.Fatalf("Play: %v", err)
	}
	if !called {
		t.Error("expected POST /api/play to be called")
	}
}
func TestViewerAPIClient_GetPlayback_Success(t *testing.T) {
	t.Parallel()
	id := "00000000-0000-4000-8000-000000000001"
	startedAt := int64(1000000000)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/playback/"+id {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":              id,
			"status":          "completed",
			"clip_count":      2,
			"completed_clips": 2,
			"started_at":      startedAt,
			"finished_at":     startedAt + 5000,
			"failed_reason":   "",
		})
	}))
	defer srv.Close()

	client := infra.NewViewerAPIClient(srv.Listener.Addr().String())
	resp, err := client.GetPlayback(t.Context(), id)
	if err != nil {
		t.Fatalf("GetPlayback: %v", err)
	}
	if resp.Status != "completed" {
		t.Errorf("expected status=completed, got %q", resp.Status)
	}
	if resp.ClipCount != 2 {
		t.Errorf("expected clip_count=2, got %d", resp.ClipCount)
	}
	if resp.CompletedClips != 2 {
		t.Errorf("expected completed_clips=2, got %d", resp.CompletedClips)
	}
	if resp.StartedAt == nil || *resp.StartedAt != startedAt {
		t.Errorf("expected started_at=%d, got %v", startedAt, resp.StartedAt)
	}
}

func TestViewerAPIClient_GetPlayback_Error(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	defer srv.Close()

	client := infra.NewViewerAPIClient(srv.Listener.Addr().String())
	_, err := client.GetPlayback(t.Context(), "00000000-0000-4000-8000-000000000001")
	if err == nil {
		t.Fatal("expected error for non-200 response")
	}
}

func TestViewerAPIClient_GetPlayback_ErrorWithBody(t *testing.T) {
	t.Parallel()
	id := "00000000-0000-4000-8000-000000000002"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("playback not found"))
	}))
	defer srv.Close()

	client := infra.NewViewerAPIClient(srv.Listener.Addr().String())
	_, err := client.GetPlayback(t.Context(), id)
	if err == nil {
		t.Fatal("expected error for non-200 response")
	}
	if !strings.Contains(err.Error(), "playback not found") {
		t.Errorf("expected error to contain body, got: %v", err)
	}
}
