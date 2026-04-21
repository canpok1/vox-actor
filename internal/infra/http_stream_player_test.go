package infra

// テストリスト: HTTPStreamPlayer
//
// インターフェース:
// DONE: HTTPStreamPlayer が app.AudioPlayer を実装すること
// DONE: HTTPStreamPlayer が app.StreamPlayer を実装すること
//
// ライフサイクル:
// DONE: Start() で指定アドレスにバインドし Addr() で実アドレスを取得できる
// DONE: ":0" でバインドすると動的ポートが割り当てられる
// DONE: Start 前に Play() を呼ぶとエラー
// DONE: Shutdown() でサーバーが停止する
// DONE: 不正アドレスで Start するとエラー
//
// エンドポイント:
// DONE: GET / が index.html を返す
// DONE: GET /app.js が JS を返す
// DONE: GET /app.css が CSS を返す
// DONE: 未登録の /clips/{id}.wav は 404
//
// Play / キュー / SSE:
// DONE: Play() で WAV がキューに登録され GET /clips/{id}.wav で配信される
// DONE: クリップID は 1 から単調増加
// DONE: 容量を超えて Play() すると古いクリップが破棄されて 404 になる
// DONE: SSE 購読者に clip イベントがブロードキャストされる
// DONE: 複数 SSE 購読者に同じイベントが届く
// DONE: 空 WAV で Play() するとエラー
// DONE: キャンセル済みコンテキストで Play() するとエラー
//
// #222 タイムラインUI / 履歴サイズ:
// DONE: clipEvent の JSON に text フィールドが含まれる
// DONE: PlayMeta.Text が空文字の場合も text フィールドが空文字で含まれる
//
// #226 backpressure:
// DONE: Play() が WAV 推定再生時間ぶんブロックしてから return する
// DONE: ctx キャンセル時は sleep を中断し ctx.Err() を返す
// DONE: WAVヘッダ不正時は warning を出して sleep をスキップしつつ正常 return する
//
// #228 話者名/スタイル名/timestamp:
// DONE: clipEvent に speakerName / styleName / timestamp フィールドが含まれる
// DONE: SpeakerLookup にヒットすれば speakerName / styleName が解決値で配信される
// DONE: SpeakerLookup に未ヒットの ID は speakerName が `話者#<ID>` にフォールバックする
// DONE: timestamp は nowFunc の戻り値の Unix ms で配信される

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/canpok1/vox-actor/internal/app"
	"github.com/canpok1/vox-actor/internal/domain/entity"
)

func newStartedPlayer(t *testing.T) *HTTPStreamPlayer {
	t.Helper()
	p, err := NewHTTPStreamPlayer("127.0.0.1:0")
	if err != nil {
		t.Fatalf("NewHTTPStreamPlayer: %v", err)
	}
	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = p.Shutdown(shutdownCtx)
	})
	return p
}

func TestHTTPStreamPlayer_ImplementsAudioPlayer(t *testing.T) {
	t.Parallel()
	var _ app.AudioPlayer = (*HTTPStreamPlayer)(nil)
}

func TestHTTPStreamPlayer_ImplementsStreamPlayer(t *testing.T) {
	t.Parallel()
	var _ app.StreamPlayer = (*HTTPStreamPlayer)(nil)
}

func TestHTTPStreamPlayer_StartAddr_DynamicPort(t *testing.T) {
	t.Parallel()
	p := newStartedPlayer(t)
	addr := p.Addr()
	if !strings.HasPrefix(addr, "127.0.0.1:") {
		t.Fatalf("unexpected addr: %s", addr)
	}
	if strings.HasSuffix(addr, ":0") {
		t.Fatalf("expected resolved port, got %s", addr)
	}
}

func TestHTTPStreamPlayer_Start_InvalidAddr(t *testing.T) {
	t.Parallel()
	p, err := NewHTTPStreamPlayer("not-a-valid-addr")
	if err != nil {
		t.Fatalf("NewHTTPStreamPlayer: %v", err)
	}
	if err := p.Start(context.Background()); err == nil {
		t.Fatal("expected Start to return an error for invalid addr")
	}
}

func TestHTTPStreamPlayer_Play_BeforeStart(t *testing.T) {
	t.Parallel()
	p, err := NewHTTPStreamPlayer("127.0.0.1:0")
	if err != nil {
		t.Fatalf("NewHTTPStreamPlayer: %v", err)
	}
	if err := p.Play(context.Background(), []byte("wav"), app.PlayMeta{}); err == nil {
		t.Fatal("expected error when Play() is called before Start()")
	}
}

func TestHTTPStreamPlayer_Shutdown_StopsServer(t *testing.T) {
	t.Parallel()
	p, err := NewHTTPStreamPlayer("127.0.0.1:0")
	if err != nil {
		t.Fatalf("NewHTTPStreamPlayer: %v", err)
	}
	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	addr := p.Addr()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := p.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}

	// Shutdown 後は HTTP リクエストが失敗する（接続拒否 or 受信エラー）
	client := &http.Client{Timeout: 500 * time.Millisecond}
	resp, err := client.Get("http://" + addr + "/")
	if err == nil {
		_ = resp.Body.Close()
		t.Fatal("expected HTTP request to fail after Shutdown")
	}
}

func TestHTTPStreamPlayer_Index(t *testing.T) {
	t.Parallel()
	p := newStartedPlayer(t)
	resp, err := http.Get("http://" + p.Addr() + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Errorf("unexpected Content-Type: %s", ct)
	}
}

func TestHTTPStreamPlayer_AppJS(t *testing.T) {
	t.Parallel()
	p := newStartedPlayer(t)
	resp, err := http.Get("http://" + p.Addr() + "/app.js")
	if err != nil {
		t.Fatalf("GET /app.js: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "javascript") {
		t.Errorf("unexpected Content-Type: %s", ct)
	}
}

func TestHTTPStreamPlayer_AppCSS(t *testing.T) {
	t.Parallel()
	p := newStartedPlayer(t)
	resp, err := http.Get("http://" + p.Addr() + "/app.css")
	if err != nil {
		t.Fatalf("GET /app.css: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "css") {
		t.Errorf("unexpected Content-Type: %s", ct)
	}
}

func TestHTTPStreamPlayer_Clip_NotFound(t *testing.T) {
	t.Parallel()
	p := newStartedPlayer(t)
	resp, err := http.Get("http://" + p.Addr() + "/clips/999.wav")
	if err != nil {
		t.Fatalf("GET /clips/999.wav: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: %d", resp.StatusCode)
	}
}

func TestHTTPStreamPlayer_Play_EmptyWAV(t *testing.T) {
	t.Parallel()
	p := newStartedPlayer(t)
	if err := p.Play(context.Background(), []byte{}, app.PlayMeta{}); err == nil {
		t.Fatal("expected error for empty WAV")
	}
}

func TestHTTPStreamPlayer_Play_CancelledContext(t *testing.T) {
	t.Parallel()
	p := newStartedPlayer(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := p.Play(ctx, []byte("RIFFdummy"), app.PlayMeta{}); err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

// subscribeSSE は /events に接続し、受信した clip イベントのdataをチャネルに流す。
// テスト終了時にクリーンアップされる。
func subscribeSSE(t *testing.T, baseURL string) <-chan string {
	t.Helper()

	req, err := http.NewRequest(http.MethodGet, baseURL+"/events", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	req = req.WithContext(ctx)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /events: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		t.Fatalf("events status: %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "text/event-stream") {
		_ = resp.Body.Close()
		t.Fatalf("events content-type: %s", ct)
	}

	events := make(chan string, 16)
	go func() {
		defer func() { _ = resp.Body.Close() }()
		defer close(events)

		scanner := bufio.NewScanner(resp.Body)
		var eventName string
		for scanner.Scan() {
			line := scanner.Text()
			switch {
			case strings.HasPrefix(line, "event:"):
				eventName = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			case strings.HasPrefix(line, "data:"):
				data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
				if eventName == "clip" {
					select {
					case events <- data:
					case <-ctx.Done():
						return
					}
				}
			case line == "":
				eventName = ""
			}
		}
	}()

	return events
}

func TestHTTPStreamPlayer_Play_DeliversSSEAndWAV(t *testing.T) {
	t.Parallel()
	p := newStartedPlayer(t)
	baseURL := "http://" + p.Addr()

	events := subscribeSSE(t, baseURL)
	// SSE購読が確立するまで少し待つ
	time.Sleep(50 * time.Millisecond)

	wav := []byte("RIFFwavdata")
	if err := p.Play(context.Background(), wav, app.PlayMeta{}); err != nil {
		t.Fatalf("Play: %v", err)
	}

	select {
	case data, ok := <-events:
		if !ok {
			t.Fatal("events channel closed before clip delivered")
		}
		if !strings.Contains(data, "\"id\":1") {
			t.Errorf("expected clip id=1, got: %s", data)
		}
		if !strings.Contains(data, "\"url\":\"/clips/1.wav\"") {
			t.Errorf("expected url=/clips/1.wav, got: %s", data)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for clip event")
	}

	resp, err := http.Get(baseURL + "/clips/1.wav")
	if err != nil {
		t.Fatalf("GET /clips/1.wav: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("clip status: %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "audio/wav") {
		t.Errorf("clip content-type: %s", ct)
	}
}

func TestHTTPStreamPlayer_ClipIDMonotonicallyIncreasing(t *testing.T) {
	t.Parallel()
	p := newStartedPlayer(t)
	baseURL := "http://" + p.Addr()

	events := subscribeSSE(t, baseURL)
	time.Sleep(50 * time.Millisecond)

	for range 3 {
		if err := p.Play(context.Background(), []byte("RIFFx"), app.PlayMeta{}); err != nil {
			t.Fatalf("Play: %v", err)
		}
	}

	for expected := 1; expected <= 3; expected++ {
		select {
		case data := <-events:
			needle := fmt.Sprintf("\"id\":%d", expected)
			if !strings.Contains(data, needle) {
				t.Errorf("expected %s in %s", needle, data)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("timeout waiting for event #%d", expected)
		}
	}
}

func TestHTTPStreamPlayer_MultipleSubscribers(t *testing.T) {
	t.Parallel()
	p := newStartedPlayer(t)
	baseURL := "http://" + p.Addr()

	const subscribers = 3
	channels := make([]<-chan string, subscribers)
	for i := range subscribers {
		channels[i] = subscribeSSE(t, baseURL)
	}
	time.Sleep(80 * time.Millisecond)

	if err := p.Play(context.Background(), []byte("RIFFx"), app.PlayMeta{}); err != nil {
		t.Fatalf("Play: %v", err)
	}

	for i, ch := range channels {
		select {
		case data := <-ch:
			if !strings.Contains(data, "\"id\":1") {
				t.Errorf("subscriber %d got %s", i, data)
			}
		case <-time.After(2 * time.Second):
			t.Errorf("subscriber %d timeout", i)
		}
	}
}

// --- #222 タイムラインUI / 履歴サイズ ---

func TestHTTPStreamPlayer_Play_ClipEventIncludesText(t *testing.T) {
	t.Parallel()
	p := newStartedPlayer(t)
	baseURL := "http://" + p.Addr()

	events := subscribeSSE(t, baseURL)
	time.Sleep(50 * time.Millisecond)

	if err := p.Play(context.Background(), []byte("RIFFx"), app.PlayMeta{Text: "こんにちは、ずんだもんなのだ"}); err != nil {
		t.Fatalf("Play: %v", err)
	}

	select {
	case data := <-events:
		if !strings.Contains(data, `"text":"こんにちは、ずんだもんなのだ"`) {
			t.Errorf("expected clip payload to contain text field, got: %s", data)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for clip event")
	}
}

func TestHTTPStreamPlayer_Play_ClipEventEmptyText(t *testing.T) {
	t.Parallel()
	p := newStartedPlayer(t)
	baseURL := "http://" + p.Addr()

	events := subscribeSSE(t, baseURL)
	time.Sleep(50 * time.Millisecond)

	if err := p.Play(context.Background(), []byte("RIFFx"), app.PlayMeta{}); err != nil {
		t.Fatalf("Play: %v", err)
	}

	select {
	case data := <-events:
		if !strings.Contains(data, `"text":""`) {
			t.Errorf("expected empty text field, got: %s", data)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for clip event")
	}
}

// --- #228 話者名/スタイル名/timestamp ---

func newStartedPlayerWithOpts(t *testing.T, opts ...HTTPStreamOption) *HTTPStreamPlayer {
	t.Helper()
	allOpts := append([]HTTPStreamOption{}, opts...)
	p, err := NewHTTPStreamPlayer("127.0.0.1:0", allOpts...)
	if err != nil {
		t.Fatalf("NewHTTPStreamPlayer: %v", err)
	}
	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = p.Shutdown(shutdownCtx)
	})
	return p
}

func TestHTTPStreamPlayer_Play_ClipEventResolvesSpeakerName(t *testing.T) {
	t.Parallel()
	lookup := map[int]entity.SpeakerStyleInfo{
		3: {SpeakerName: "ずんだもん", StyleName: "ノーマル"},
	}
	fixed := time.Date(2026, 4, 21, 12, 0, 0, 123_000_000, time.UTC)
	p := newStartedPlayerWithOpts(t,
		WithSpeakerLookup(lookup),
		withNowFunc(func() time.Time { return fixed }),
	)
	baseURL := "http://" + p.Addr()

	events := subscribeSSE(t, baseURL)
	time.Sleep(50 * time.Millisecond)

	if err := p.Play(context.Background(), []byte("RIFFx"), app.PlayMeta{Text: "やっほー", SpeakerID: 3}); err != nil {
		t.Fatalf("Play: %v", err)
	}

	select {
	case data := <-events:
		if !strings.Contains(data, `"speakerName":"ずんだもん"`) {
			t.Errorf("expected speakerName=ずんだもん, got: %s", data)
		}
		if !strings.Contains(data, `"styleName":"ノーマル"`) {
			t.Errorf("expected styleName=ノーマル, got: %s", data)
		}
		expectedTS := fmt.Sprintf(`"timestamp":%d`, fixed.UnixMilli())
		if !strings.Contains(data, expectedTS) {
			t.Errorf("expected %s, got: %s", expectedTS, data)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for clip event")
	}
}

func TestHTTPStreamPlayer_Play_ClipEventFallbackForUnknownSpeaker(t *testing.T) {
	t.Parallel()
	lookup := map[int]entity.SpeakerStyleInfo{
		3: {SpeakerName: "ずんだもん", StyleName: "ノーマル"},
	}
	p := newStartedPlayerWithOpts(t, WithSpeakerLookup(lookup))
	baseURL := "http://" + p.Addr()

	events := subscribeSSE(t, baseURL)
	time.Sleep(50 * time.Millisecond)

	if err := p.Play(context.Background(), []byte("RIFFx"), app.PlayMeta{Text: "未知", SpeakerID: 999}); err != nil {
		t.Fatalf("Play: %v", err)
	}

	select {
	case data := <-events:
		if !strings.Contains(data, `"speakerName":"話者#999"`) {
			t.Errorf("expected speakerName fallback to 話者#999, got: %s", data)
		}
		if !strings.Contains(data, `"styleName":""`) {
			t.Errorf("expected styleName='' for unknown id, got: %s", data)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for clip event")
	}
}

func TestHTTPStreamPlayer_Play_ClipEventNoLookup(t *testing.T) {
	t.Parallel()
	// lookup を設定しない場合、すべての SpeakerID が fallback される。
	p := newStartedPlayer(t)
	baseURL := "http://" + p.Addr()

	events := subscribeSSE(t, baseURL)
	time.Sleep(50 * time.Millisecond)

	if err := p.Play(context.Background(), []byte("RIFFx"), app.PlayMeta{SpeakerID: 3}); err != nil {
		t.Fatalf("Play: %v", err)
	}

	select {
	case data := <-events:
		if !strings.Contains(data, `"speakerName":"話者#3"`) {
			t.Errorf("expected speakerName fallback when no lookup, got: %s", data)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for clip event")
	}
}

// --- #226 backpressure ---

func TestHTTPStreamPlayer_Play_BlocksForEstimatedDuration(t *testing.T) {
	t.Parallel()
	p := newStartedPlayer(t)

	// byteRate=48000, dataSize=4800 → 100ms
	wavData := buildWAV(48000, 4800)

	start := time.Now()
	if err := p.Play(context.Background(), wavData, app.PlayMeta{}); err != nil {
		t.Fatalf("Play: %v", err)
	}
	elapsed := time.Since(start)

	const expected = 100 * time.Millisecond
	if elapsed < expected {
		t.Errorf("Play returned too quickly: elapsed=%v, expected at least %v", elapsed, expected)
	}
	if elapsed > expected+200*time.Millisecond {
		t.Errorf("Play blocked too long: elapsed=%v, expected ~%v", elapsed, expected)
	}
}

func TestHTTPStreamPlayer_Play_ContextCancelInterruptsSleep(t *testing.T) {
	t.Parallel()
	p := newStartedPlayer(t)

	// byteRate=48000, dataSize=480000 → 10秒
	wavData := buildWAV(48000, 480000)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	err := p.Play(ctx, wavData, app.PlayMeta{})
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected ctx.Err() to be returned, got nil")
	}
	if elapsed > 1*time.Second {
		t.Errorf("Play should have been interrupted quickly: elapsed=%v", elapsed)
	}
}

func TestHTTPStreamPlayer_Play_InvalidWAVHeaderReturnsWithoutSleep(t *testing.T) {
	t.Parallel()

	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelWarn}))
	p, err := NewHTTPStreamPlayer("127.0.0.1:0", WithHTTPStreamLogger(logger))
	if err != nil {
		t.Fatalf("NewHTTPStreamPlayer: %v", err)
	}
	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = p.Shutdown(shutdownCtx)
	})

	// 不正な WAV (RIFFヘッダなし) でも Play 自体はエラーにならず即 return する。
	start := time.Now()
	if err := p.Play(context.Background(), []byte("not-a-valid-wav-data"), app.PlayMeta{}); err != nil {
		t.Fatalf("Play: %v", err)
	}
	elapsed := time.Since(start)

	if elapsed > 100*time.Millisecond {
		t.Errorf("Play should return quickly when WAV header is invalid: elapsed=%v", elapsed)
	}

	// 警告ログが出ていることを確認する。
	warnLine := findLogLine(logBuf.String(), "failed to estimate WAV duration")
	if warnLine == "" {
		t.Errorf("expected warning log for invalid WAV header, got: %s", logBuf.String())
	}
	if !strings.Contains(warnLine, "level=WARN") {
		t.Errorf("expected WARN level log, got: %s", warnLine)
	}
}

// findLogLine は logs に含まれる行のうち needle を含む最初の行を返す。
// 見つからない場合は空文字を返す。
func findLogLine(logs, needle string) string {
	for line := range strings.SplitSeq(logs, "\n") {
		if strings.Contains(line, needle) {
			return line
		}
	}
	return ""
}

// --- #226 履歴サイズ固定値（20）の検証 ---

func TestHTTPStreamPlayer_RingBuffer_FixedSize(t *testing.T) {
	t.Parallel()
	p := newStartedPlayer(t)
	baseURL := "http://" + p.Addr()

	// 21件積むと1件目だけが押し出される（容量20）
	const total = 21
	for range total {
		if err := p.Play(context.Background(), []byte("RIFFx"), app.PlayMeta{}); err != nil {
			t.Fatalf("Play: %v", err)
		}
	}

	resp1, err := http.Get(baseURL + "/clips/1.wav")
	if err != nil {
		t.Fatalf("GET clip1: %v", err)
	}
	_ = resp1.Body.Close()
	if resp1.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 for evicted clip 1, got %d", resp1.StatusCode)
	}

	// 2件目（最古の生存クリップ）と21件目（最新）は取得できる
	resp2, err := http.Get(baseURL + "/clips/2.wav")
	if err != nil {
		t.Fatalf("GET clip2: %v", err)
	}
	_ = resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Errorf("expected 200 for clip 2 (oldest surviving), got %d", resp2.StatusCode)
	}

	resp21, err := http.Get(baseURL + "/clips/21.wav")
	if err != nil {
		t.Fatalf("GET clip21: %v", err)
	}
	_ = resp21.Body.Close()
	if resp21.StatusCode != http.StatusOK {
		t.Errorf("expected 200 for clip 21, got %d", resp21.StatusCode)
	}
}
