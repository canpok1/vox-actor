//go:build e2e

package e2e

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// sseEvent は SSE の 1 イベントを表す。
type sseEvent struct {
	EventType string
	Data      string
}

// readSSEEvent は r から次の完全な SSE イベントを読み取る。
// "event: xxx\ndata: yyy\n\n" 形式を想定する。
// コンテキストが終了した場合は ctx.Err() を返す。EOF 到達時は io.EOF を返す。
func readSSEEvent(ctx context.Context, r *bufio.Reader) (sseEvent, error) {
	var ev sseEvent
	for {
		select {
		case <-ctx.Done():
			return sseEvent{}, ctx.Err()
		default:
		}
		line, err := r.ReadString('\n')
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			if ev.EventType != "" || ev.Data != "" {
				return ev, nil
			}
			if err != nil {
				return sseEvent{}, err
			}
			continue
		}
		if after, ok := strings.CutPrefix(line, "event: "); ok {
			ev.EventType = after
		} else if after, ok := strings.CutPrefix(line, "data: "); ok {
			ev.Data = after
		}
		if err != nil {
			return sseEvent{}, err
		}
	}
}

// connectSSE は addr の /events に SSE 接続して *http.Response を返す。
// 接続失敗時は最大 20 回リトライする。
func connectSSE(t *testing.T, addr string) *http.Response {
	t.Helper()
	client := &http.Client{
		Transport: &http.Transport{DisableCompression: true},
	}
	url := fmt.Sprintf("http://%s/events", addr)
	var (
		resp   *http.Response
		reqErr error
	)
	for i := 0; i < 20; i++ {
		req, err := http.NewRequest(http.MethodGet, url, nil) //nolint:noctx
		if err != nil {
			t.Fatalf("failed to create SSE request: %v", err)
		}
		resp, reqErr = client.Do(req)
		if reqErr == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if reqErr != nil {
		t.Fatalf("failed to connect to /events: %v", reqErr)
	}
	return resp
}

// startFakeVoicevox は最小限の VOICEVOX API を実装した httptest.Server を返す。
// HealthCheck / GetSpeakers / CreateQuery / Synthesize に対応する。
func startFakeVoicevox(t *testing.T) *httptest.Server {
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
		w.Header().Set("Content-Type", "audio/wav")
		// estimateWAVDuration が失敗しても Play は非 fatal で継続するため、最小 WAV バイトで十分。
		_, _ = w.Write(bytes.Repeat([]byte{0x00}, 64))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// postAPIPlay は addr の /api/play に POST してレスポンスを返す。
func postAPIPlay(t *testing.T, addr, text string, speakerID int) *http.Response {
	t.Helper()
	type clip struct {
		Text      string `json:"text"`
		SpeakerID int    `json:"speaker_id"`
	}
	type playRequest struct {
		Clips []clip `json:"clips"`
	}
	body, err := json.Marshal(playRequest{Clips: []clip{{Text: text, SpeakerID: speakerID}}})
	if err != nil {
		t.Fatalf("marshal play request: %v", err)
	}
	url := fmt.Sprintf("http://%s/api/play", addr)
	resp, err := http.Post(url, "application/json", bytes.NewReader(body)) //nolint:noctx
	if err != nil {
		t.Fatalf("POST /api/play: %v", err)
	}
	return resp
}

func TestViewerE2E_Events_ConnectionHeaders(t *testing.T) {
	t.Parallel()

	port := freePort(t)
	homeDir := t.TempDir()
	workspace := t.TempDir()

	// VOICEVOX に接続できない → silent モード。SSE 接続自体は成功する。
	vp, addr := startViewerGetAddr(t, map[string]string{
		"HOME":                homeDir,
		"VOX_ACTOR_WORKSPACE": workspace,
		"VOX_ENGINE_URL":      "http://127.0.0.1:1",
	}, port)

	resp := connectSSE(t, addr)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("Content-Type: expected %q, got %q", "text/event-stream", ct)
	}
	if cc := resp.Header.Get("Cache-Control"); cc != "no-cache" {
		t.Errorf("Cache-Control: expected %q, got %q", "no-cache", cc)
	}
	if conn := resp.Header.Get("Connection"); conn != "keep-alive" {
		t.Errorf("Connection: expected %q, got %q", "keep-alive", conn)
	}

	vp.assertCleanExit(3 * time.Second)
}

func TestViewerE2E_Events_ClipEventAfterAPIPlay(t *testing.T) {
	t.Parallel()

	fakeVox := startFakeVoicevox(t)
	port := freePort(t)
	homeDir := t.TempDir()
	workspace := t.TempDir()

	vp, addr := startViewerGetAddr(t, map[string]string{
		"HOME":                homeDir,
		"VOX_ACTOR_WORKSPACE": workspace,
		"VOX_ENGINE_URL":      fakeVox.URL,
	}, port)

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
			URL         string `json:"url"`
			Text        string `json:"text"`
			SpeakerName string `json:"speakerName"`
		}
		if err := json.Unmarshal([]byte(ev.Data), &data); err != nil {
			t.Fatalf("failed to unmarshal clip data: %v\nraw: %s", err, ev.Data)
		}
		if data.Text != "テスト発話" {
			t.Errorf("text: expected %q, got %q", "テスト発話", data.Text)
		}
		if data.SpeakerName == "" {
			t.Error("expected non-empty speakerName in clip event")
		}
		if data.URL == "" {
			t.Error("expected non-empty URL in non-silent mode clip event")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("timeout waiting for clip SSE event")
	}

	vp.assertCleanExit(3 * time.Second)
}

func TestViewerE2E_Events_MultipleClients(t *testing.T) {
	t.Parallel()

	fakeVox := startFakeVoicevox(t)
	port := freePort(t)
	homeDir := t.TempDir()
	workspace := t.TempDir()

	vp, addr := startViewerGetAddr(t, map[string]string{
		"HOME":                homeDir,
		"VOX_ACTOR_WORKSPACE": workspace,
		"VOX_ENGINE_URL":      fakeVox.URL,
	}, port)

	const numClients = 3
	eventChs := make([]chan sseEvent, numClients)

	for i := range numClients {
		sseResp := connectSSE(t, addr)
		ch := make(chan sseEvent, 1)
		eventChs[i] = ch
		go func(resp *http.Response, ch chan sseEvent) {
			defer resp.Body.Close()
			r := bufio.NewReader(resp.Body)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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
		}(sseResp, ch)
	}

	// 全クライアントの subscriber 登録を待つ。
	time.Sleep(200 * time.Millisecond)

	playResp := postAPIPlay(t, addr, "マルチキャストテスト", 3)
	defer playResp.Body.Close()
	if playResp.StatusCode != http.StatusOK {
		t.Fatalf("POST /api/play expected 200, got %d", playResp.StatusCode)
	}

	for i, ch := range eventChs {
		select {
		case ev := <-ch:
			if ev.EventType != "clip" {
				t.Errorf("client %d: expected event type clip, got %q", i, ev.EventType)
			}
		case <-time.After(10 * time.Second):
			t.Fatalf("client %d: timeout waiting for clip SSE event", i)
		}
	}

	vp.assertCleanExit(3 * time.Second)
}

func TestViewerE2E_Events_ServerShutdownClosesSSE(t *testing.T) {
	t.Parallel()

	port := freePort(t)
	homeDir := t.TempDir()
	workspace := t.TempDir()

	vp, addr := startViewerGetAddr(t, map[string]string{
		"HOME":                homeDir,
		"VOX_ACTOR_WORKSPACE": workspace,
		"VOX_ENGINE_URL":      "http://127.0.0.1:1",
	}, port)

	sseResp := connectSSE(t, addr)

	// SSE 切断を goroutine で検知する（EOF または read error）
	doneCh := make(chan struct{}, 1)
	go func() {
		r := bufio.NewReader(sseResp.Body)
		for {
			_, err := r.ReadByte()
			if err != nil {
				doneCh <- struct{}{}
				return
			}
		}
	}()

	// SIGTERM 送信 → Shutdown() → subscribers.closeAll() → SSE ハンドラが channel close を検知して終了
	_, _ = vp.stop(3 * time.Second)
	sseResp.Body.Close()

	select {
	case <-doneCh:
		// SSE 接続が正常に切断された
	case <-time.After(5 * time.Second):
		t.Fatal("timeout: SSE connection was not closed after server shutdown")
	}
}

func TestViewerE2E_Events_SilentMode_ConnectionAndNoClip(t *testing.T) {
	t.Parallel()

	port := freePort(t)
	homeDir := t.TempDir()
	workspace := t.TempDir()

	// VOICEVOX 接続不可 → silent モード
	vp, addr := startViewerGetAddr(t, map[string]string{
		"HOME":                homeDir,
		"VOX_ACTOR_WORKSPACE": workspace,
		"VOX_ENGINE_URL":      "http://127.0.0.1:1",
	}, port)

	// silent モードでも SSE 接続は成功する
	sseResp := connectSSE(t, addr)
	defer sseResp.Body.Close()

	if sseResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 in silent mode, got %d", sseResp.StatusCode)
	}

	time.Sleep(100 * time.Millisecond)

	// silent モードでの SSE clip イベント不在を goroutine で監視
	unexpectedClipCh := make(chan sseEvent, 1)
	go func() {
		r := bufio.NewReader(sseResp.Body)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		for {
			ev, err := readSSEEvent(ctx, r)
			if err != nil {
				return
			}
			if ev.EventType == "clip" {
				unexpectedClipCh <- ev
				return
			}
		}
	}()

	// silent モードの /api/play → {silent: true} を返し SSE には clip イベントを配信しない
	playResp := postAPIPlay(t, addr, "サイレントテスト", 3)
	defer playResp.Body.Close()
	if playResp.StatusCode != http.StatusOK {
		t.Fatalf("POST /api/play expected 200, got %d", playResp.StatusCode)
	}

	var playResult struct {
		Silent bool `json:"silent"`
	}
	body, err := io.ReadAll(playResp.Body)
	if err != nil {
		t.Fatalf("read /api/play response: %v", err)
	}
	if err := json.Unmarshal(body, &playResult); err != nil {
		t.Fatalf("decode /api/play response: %v\nbody: %s", err, body)
	}
	if !playResult.Silent {
		t.Error("expected silent=true in silent mode /api/play response")
	}

	// clip イベントが来ないことを確認（2 秒タイムアウト）
	select {
	case ev := <-unexpectedClipCh:
		t.Errorf("expected no clip event in silent mode, but got: event=%s data=%s", ev.EventType, ev.Data)
	case <-time.After(2 * time.Second):
		// 期待通り: clip イベントなし
	}

	vp.assertCleanExit(3 * time.Second)
}
