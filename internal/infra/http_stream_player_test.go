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
//
// #229 音量localStorage保存 / 消音チェックボックス:
// DONE: index.html に音量スライダー初期値 50 と消音チェックボックス(checked) が含まれる
//
// #237 音声テストタブ / 接続バッジ / speakers.json / test-clip:
// DONE: index.html に配信/音声テストのタブ要素と接続バッジ要素が含まれる
// DONE: index.html から `.status` 行が削除されている
// DONE: /test-clip が VoicevoxClient 未注入時に 503 を返す
// DONE: /test-clip が speaker パラメータ欠落・不正・未登録IDで 400 を返す
// DONE: /test-clip が成功時に 200 WAV を返す
// DONE: /test-clip が 2 回目の呼び出しで VoicevoxClient を呼ばずキャッシュから返す
// DONE: /test-clip が VOICEVOX 合成失敗時に 502 を返す
// DONE: /test-clip は WithTestPhrase のフレーズを CreateQuery に渡す
//
// #284 無音モードフォールバック / /api/status:
// DONE: /speakers.json は 404 (エンドポイント廃止)
// DONE: /api/status が speakerLookup を id 昇順で配列化し silent=false で返す
// DONE: /api/status は lookup 未設定時に speakers=[] silent=false を返す
// DONE: SetSilent + Start 後の /api/status が silent=true, silentReason 付き, speakers=[] を返す
// DONE: PlayText が url="" で clipEvent を配信し silentInterval ぶん待機する
// DONE: PlayText はキャンセル済み ctx で即エラー
// DONE: PlayText は Start 前だとエラー
//
// #285 サーバー側エラー配信:
// TODO: BroadcastError が SSE "error" イベントで errorEvent を配信する（synthesis カテゴリ）
// TODO: synthesis カテゴリは path/text/speakerName/styleName を含み lookup から解決される
// TODO: file カテゴリは path を含み text/speakerName/styleName を省略する
// TODO: connection カテゴリは message のみを含む
// TODO: errorEvent.id はクリップIDと独立に 1 から単調増加する
// TODO: BroadcastError は複数購読者にブロードキャストされる

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/canpok1/vox-actor/internal/app"
	"github.com/canpok1/vox-actor/internal/domain/entity"
)

// testIndexHTML は Vite build 後の dist/index.html を模した fixture。
// 配信画面テストが検証する UI マーカーをすべて含み、ハッシュ付きアセットへの参照を持つ。
const testIndexHTML = `<!DOCTYPE html>
<html lang="ja">
<head>
<meta charset="UTF-8">
<title>vox-actor stream</title>
<script type="module" crossorigin src="/assets/index-test.js"></script>
<link rel="stylesheet" crossorigin href="/assets/index-test.css">
</head>
<body>
<main class="container">
  <div class="header">
    <h1>vox-actor stream</h1>
    <span id="status-badge" class="badge badge-connected">● 接続中</span>
  </div>
  <div class="volume-row">
    <label class="volume-slider">
      音量
      <input type="range" id="volume" min="0" max="100" value="50">
      <span id="volume-icon">🔇</span>
    </label>
    <label class="toggle"><input type="checkbox" id="mute" checked>消音</label>
  </div>
  <div class="tabs">
    <button type="button" id="tab-stream" class="tab active">配信</button>
    <button type="button" id="tab-test" class="tab">音声テスト</button>
  </div>
  <section id="panel-stream" class="panel">
    <div class="timeline-controls">
      <label class="toggle"><input type="checkbox" id="show-speaker-name" checked>話者名</label>
      <label class="toggle"><input type="checkbox" id="show-style-name" checked>スタイル</label>
      <label class="toggle"><input type="checkbox" id="show-timestamp" checked>時刻</label>
    </div>
  </section>
  <section id="panel-test" class="panel hidden">
    <select id="test-speaker"></select>
    <button type="button" id="test-play">▶ テスト再生</button>
  </section>
  <audio id="player" muted></audio>
</main>
</body>
</html>
`

// newTestStreamAssets は Vite dist ライクな fs.FS を返す。
// index.html はマーカー検証用に共通の testIndexHTML を使う。
func newTestStreamAssets() fstest.MapFS {
	return fstest.MapFS{
		"index.html":            {Data: []byte(testIndexHTML)},
		"assets/index-test.js":  {Data: []byte("export const a=1;\n")},
		"assets/index-test.css": {Data: []byte("body{}\n")},
	}
}

func newStartedPlayer(t *testing.T) *HTTPStreamPlayer {
	t.Helper()
	p, err := NewHTTPStreamPlayer("127.0.0.1:0", newTestStreamAssets())
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
	p, err := NewHTTPStreamPlayer("not-a-valid-addr", newTestStreamAssets())
	if err != nil {
		t.Fatalf("NewHTTPStreamPlayer: %v", err)
	}
	if err := p.Start(context.Background()); err == nil {
		t.Fatal("expected Start to return an error for invalid addr")
	}
}

func TestHTTPStreamPlayer_Play_BeforeStart(t *testing.T) {
	t.Parallel()
	p, err := NewHTTPStreamPlayer("127.0.0.1:0", newTestStreamAssets())
	if err != nil {
		t.Fatalf("NewHTTPStreamPlayer: %v", err)
	}
	if err := p.Play(context.Background(), []byte("wav"), app.PlayMeta{}); err == nil {
		t.Fatal("expected error when Play() is called before Start()")
	}
}

func TestHTTPStreamPlayer_Shutdown_StopsServer(t *testing.T) {
	t.Parallel()
	p, err := NewHTTPStreamPlayer("127.0.0.1:0", newTestStreamAssets())
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

// fetchIndexAssetPaths は GET / から Vite が書き換えた /assets/<hash>.{js,css} のパスを抽出する。
// ハッシュ付きファイル名はテスト側で決め打ちできないため、index.html からランタイムで取得する。
func fetchIndexAssetPaths(t *testing.T, baseURL string) (jsPath, cssPath string) {
	t.Helper()
	resp, err := http.Get(baseURL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read index: %v", err)
	}
	html := string(body)
	jsRe := regexp.MustCompile(`src="(/assets/[^"]+\.js)"`)
	cssRe := regexp.MustCompile(`href="(/assets/[^"]+\.css)"`)
	jsMatch := jsRe.FindStringSubmatch(html)
	cssMatch := cssRe.FindStringSubmatch(html)
	if len(jsMatch) < 2 {
		t.Fatalf("no /assets/*.js reference in index: %s", html)
	}
	if len(cssMatch) < 2 {
		t.Fatalf("no /assets/*.css reference in index: %s", html)
	}
	return jsMatch[1], cssMatch[1]
}

func TestHTTPStreamPlayer_Assets_JSAndCSS(t *testing.T) {
	t.Parallel()
	p := newStartedPlayer(t)
	baseURL := "http://" + p.Addr()
	jsPath, cssPath := fetchIndexAssetPaths(t, baseURL)

	resp, err := http.Get(baseURL + jsPath)
	if err != nil {
		t.Fatalf("GET %s: %v", jsPath, err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("js status: %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "javascript") {
		t.Errorf("unexpected js Content-Type: %s", ct)
	}

	resp, err = http.Get(baseURL + cssPath)
	if err != nil {
		t.Fatalf("GET %s: %v", cssPath, err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("css status: %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "css") {
		t.Errorf("unexpected css Content-Type: %s", ct)
	}
}

func TestHTTPStreamPlayer_Index_CacheControlNoCache(t *testing.T) {
	t.Parallel()
	p := newStartedPlayer(t)
	resp, err := http.Get("http://" + p.Addr() + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	_ = resp.Body.Close()
	if got := resp.Header.Get("Cache-Control"); got != "no-cache" {
		t.Errorf("expected Cache-Control=no-cache on /, got %q", got)
	}
}

func TestHTTPStreamPlayer_Assets_CacheControlImmutable(t *testing.T) {
	t.Parallel()
	p := newStartedPlayer(t)
	baseURL := "http://" + p.Addr()
	jsPath, _ := fetchIndexAssetPaths(t, baseURL)
	resp, err := http.Get(baseURL + jsPath)
	if err != nil {
		t.Fatalf("GET %s: %v", jsPath, err)
	}
	_ = resp.Body.Close()
	got := resp.Header.Get("Cache-Control")
	if !strings.Contains(got, "immutable") || !strings.Contains(got, "max-age=31536000") {
		t.Errorf("expected Cache-Control to include immutable and max-age=31536000 on /assets/*, got %q", got)
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
	return subscribeSSEByEvent(t, baseURL, "clip")
}

// subscribeSSEByEvent は /events に接続し、指定したイベント名の data のみをチャネルに流す。
func subscribeSSEByEvent(t *testing.T, baseURL, wantEvent string) <-chan string {
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
				if eventName == wantEvent {
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
	p, err := NewHTTPStreamPlayer("127.0.0.1:0", newTestStreamAssets(), opts...)
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

func TestHTTPStreamPlayer_Index_ContainsToggleControls(t *testing.T) {
	t.Parallel()
	p := newStartedPlayer(t)
	resp, err := http.Get("http://" + p.Addr() + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	html := string(body)
	for _, needle := range []string{
		`id="show-speaker-name"`,
		`id="show-style-name"`,
		`id="show-timestamp"`,
		"話者名",
		"スタイル",
		"時刻",
	} {
		if !strings.Contains(html, needle) {
			t.Errorf("expected index.html to contain %q", needle)
		}
	}
}

func TestHTTPStreamPlayer_Index_ContainsVolumeControls(t *testing.T) {
	t.Parallel()
	p := newStartedPlayer(t)
	resp, err := http.Get("http://" + p.Addr() + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	html := string(body)
	for _, needle := range []string{
		`id="volume"`,
		`value="50"`,
		`id="mute" checked`,
		"消音",
	} {
		if !strings.Contains(html, needle) {
			t.Errorf("expected index.html to contain %q", needle)
		}
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
	p, err := NewHTTPStreamPlayer("127.0.0.1:0", newTestStreamAssets(), WithHTTPStreamLogger(logger))
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

// --- #237 音声テストタブ / 接続バッジ / speakers.json / test-clip ---

func TestHTTPStreamPlayer_Index_ContainsTabElements(t *testing.T) {
	t.Parallel()
	p := newStartedPlayer(t)
	resp, err := http.Get("http://" + p.Addr() + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	html := string(body)
	for _, needle := range []string{
		`id="tab-stream"`,
		`id="tab-test"`,
		`id="panel-stream"`,
		`id="panel-test"`,
		`id="test-speaker"`,
		`id="test-play"`,
		"配信",
		"音声テスト",
	} {
		if !strings.Contains(html, needle) {
			t.Errorf("expected index.html to contain %q", needle)
		}
	}
}

func TestHTTPStreamPlayer_Index_ContainsConnectionBadge(t *testing.T) {
	t.Parallel()
	p := newStartedPlayer(t)
	resp, err := http.Get("http://" + p.Addr() + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	html := string(body)
	if !strings.Contains(html, `id="status-badge"`) {
		t.Errorf("expected index.html to contain status-badge element")
	}
}

func TestHTTPStreamPlayer_Index_NoStatusRow(t *testing.T) {
	t.Parallel()
	p := newStartedPlayer(t)
	resp, err := http.Get("http://" + p.Addr() + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	html := string(body)
	if strings.Contains(html, `class="status"`) {
		t.Errorf("expected index.html to not contain class=\"status\", got: %s", html)
	}
}

type apiStatusBody struct {
	Silent       bool   `json:"silent"`
	SilentReason string `json:"silentReason"`
	Speakers     []struct {
		ID          int    `json:"id"`
		SpeakerName string `json:"speakerName"`
		StyleName   string `json:"styleName"`
	} `json:"speakers"`
}

func fetchAPIStatus(t *testing.T, p *HTTPStreamPlayer) apiStatusBody {
	t.Helper()
	resp, err := http.Get("http://" + p.Addr() + "/api/status")
	if err != nil {
		t.Fatalf("GET /api/status: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Errorf("unexpected Content-Type: %s", ct)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var got apiStatusBody
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("unmarshal: %v body=%s", err, string(body))
	}
	return got
}

func TestHTTPStreamPlayer_APIStatus_ReturnsSortedSpeakers(t *testing.T) {
	t.Parallel()
	lookup := map[int]entity.SpeakerStyleInfo{
		7: {SpeakerName: "四国めたん", StyleName: "ノーマル"},
		3: {SpeakerName: "ずんだもん", StyleName: "ノーマル"},
		1: {SpeakerName: "四国めたん", StyleName: "あまあま"},
	}
	p := newStartedPlayerWithOpts(t, WithSpeakerLookup(lookup))
	got := fetchAPIStatus(t, p)
	if got.Silent {
		t.Errorf("expected silent=false, got true")
	}
	if got.SilentReason != "" {
		t.Errorf("expected silentReason empty, got %q", got.SilentReason)
	}
	if len(got.Speakers) != 3 {
		t.Fatalf("expected 3 speakers, got %d: %+v", len(got.Speakers), got)
	}
	wantIDs := []int{1, 3, 7}
	for i, w := range wantIDs {
		if got.Speakers[i].ID != w {
			t.Errorf("entry %d: expected id=%d, got %d", i, w, got.Speakers[i].ID)
		}
	}
	if got.Speakers[0].SpeakerName != "四国めたん" || got.Speakers[0].StyleName != "あまあま" {
		t.Errorf("entry 0: %+v", got.Speakers[0])
	}
	if got.Speakers[1].SpeakerName != "ずんだもん" || got.Speakers[1].StyleName != "ノーマル" {
		t.Errorf("entry 1: %+v", got.Speakers[1])
	}
}

func TestHTTPStreamPlayer_APIStatus_ReturnsOrderedSpeakers(t *testing.T) {
	t.Parallel()
	lookup := map[int]entity.SpeakerStyleInfo{
		1: {SpeakerName: "四国めたん", StyleName: "あまあま"},
		7: {SpeakerName: "四国めたん", StyleName: "ノーマル"},
		3: {SpeakerName: "ずんだもん", StyleName: "ノーマル"},
	}
	orderedIDs := []int{7, 3, 1}
	p := newStartedPlayerWithOpts(t,
		WithSpeakerLookup(lookup),
		WithOrderedSpeakerIDs(orderedIDs),
	)
	got := fetchAPIStatus(t, p)
	if got.Silent {
		t.Errorf("expected silent=false, got true")
	}
	if got.SilentReason != "" {
		t.Errorf("expected silentReason empty, got %q", got.SilentReason)
	}
	if len(got.Speakers) != 3 {
		t.Fatalf("expected 3 speakers, got %d: %+v", len(got.Speakers), got)
	}
	wantIDs := []int{7, 3, 1}
	for i, w := range wantIDs {
		if got.Speakers[i].ID != w {
			t.Errorf("entry %d: expected id=%d, got %d", i, w, got.Speakers[i].ID)
		}
	}
	if got.Speakers[0].SpeakerName != "四国めたん" || got.Speakers[0].StyleName != "ノーマル" {
		t.Errorf("entry 0: expected 四国めたん/ノーマル, got %+v", got.Speakers[0])
	}
	if got.Speakers[1].SpeakerName != "ずんだもん" || got.Speakers[1].StyleName != "ノーマル" {
		t.Errorf("entry 1: expected ずんだもん/ノーマル, got %+v", got.Speakers[1])
	}
	if got.Speakers[2].SpeakerName != "四国めたん" || got.Speakers[2].StyleName != "あまあま" {
		t.Errorf("entry 2: expected 四国めたん/あまあま, got %+v", got.Speakers[2])
	}
}

func TestHTTPStreamPlayer_APIStatus_NoLookupReturnsEmptySpeakers(t *testing.T) {
	t.Parallel()
	p := newStartedPlayer(t)
	got := fetchAPIStatus(t, p)
	if got.Silent {
		t.Errorf("expected silent=false, got true")
	}
	if got.SilentReason != "" {
		t.Errorf("expected silentReason empty, got %q", got.SilentReason)
	}
	if len(got.Speakers) != 0 {
		t.Errorf("expected empty speakers, got %+v", got.Speakers)
	}
}

func TestHTTPStreamPlayer_APIStatus_SilentModeReturnsReasonAndEmpty(t *testing.T) {
	t.Parallel()
	lookup := map[int]entity.SpeakerStyleInfo{
		3: {SpeakerName: "ずんだもん", StyleName: "ノーマル"},
	}
	reason := "VOICEVOXに接続できないため無音モードで起動しました。\n音を再生したい場合はVOICEVOXに接続できる状態で起動し直してください。"
	p, err := NewHTTPStreamPlayer("127.0.0.1:0", newTestStreamAssets(), WithSpeakerLookup(lookup))
	if err != nil {
		t.Fatalf("NewHTTPStreamPlayer: %v", err)
	}
	p.SetSilent(reason)
	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = p.Shutdown(shutdownCtx)
	})

	got := fetchAPIStatus(t, p)
	if !got.Silent {
		t.Errorf("expected silent=true, got false")
	}
	if got.SilentReason != reason {
		t.Errorf("silentReason mismatch\nwant: %q\ngot:  %q", reason, got.SilentReason)
	}
	if len(got.Speakers) != 0 {
		t.Errorf("expected speakers=[] in silent mode, got %+v", got.Speakers)
	}
}

// /speakers.json は廃止されたので 404 を返すことを検証する。
func TestHTTPStreamPlayer_SpeakersJSON_Removed(t *testing.T) {
	t.Parallel()
	p := newStartedPlayer(t)
	resp, err := http.Get("http://" + p.Addr() + "/speakers.json")
	if err != nil {
		t.Fatalf("GET /speakers.json: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 for deprecated /speakers.json, got %d", resp.StatusCode)
	}
}

// PlayText は clipEvent を url="" で配信し、silentInterval ぶん待機してから return する。
func TestHTTPStreamPlayer_PlayText_EmptyURLAndSilentInterval(t *testing.T) {
	t.Parallel()
	lookup := map[int]entity.SpeakerStyleInfo{
		3: {SpeakerName: "ずんだもん", StyleName: "ノーマル"},
	}
	p, err := NewHTTPStreamPlayer("127.0.0.1:0", newTestStreamAssets(),
		WithSpeakerLookup(lookup),
	)
	if err != nil {
		t.Fatalf("NewHTTPStreamPlayer: %v", err)
	}
	p.SetSilent("reason")
	p.silentInterval = 40 * time.Millisecond // テスト高速化
	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = p.Shutdown(shutdownCtx)
	})

	baseURL := "http://" + p.Addr()
	events := subscribeSSE(t, baseURL)
	time.Sleep(50 * time.Millisecond) // 購読確立待ち

	start := time.Now()
	if err := p.PlayText(context.Background(), app.PlayMeta{Text: "こんにちは", SpeakerID: 3}); err != nil {
		t.Fatalf("PlayText: %v", err)
	}
	elapsed := time.Since(start)
	if elapsed < 30*time.Millisecond {
		t.Errorf("PlayText returned too early: %v < 30ms", elapsed)
	}

	select {
	case data := <-events:
		var ev clipEvent
		if err := json.Unmarshal([]byte(data), &ev); err != nil {
			t.Fatalf("unmarshal: %v data=%s", err, data)
		}
		if ev.URL != "" {
			t.Errorf("expected url empty in silent mode, got %q", ev.URL)
		}
		if ev.Text != "こんにちは" {
			t.Errorf("unexpected Text: %q", ev.Text)
		}
		if ev.SpeakerName != "ずんだもん" {
			t.Errorf("unexpected SpeakerName: %q", ev.SpeakerName)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no clip event delivered within 2s")
	}
}

func TestHTTPStreamPlayer_PlayText_CtxCancelled(t *testing.T) {
	t.Parallel()
	p := newStartedPlayer(t)
	p.silentInterval = 2 * time.Second

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := p.PlayText(ctx, app.PlayMeta{Text: "x", SpeakerID: 3}); err == nil {
		t.Fatal("expected error for cancelled ctx")
	}
}

func TestHTTPStreamPlayer_PlayText_BeforeStart(t *testing.T) {
	t.Parallel()
	p, err := NewHTTPStreamPlayer("127.0.0.1:0", newTestStreamAssets())
	if err != nil {
		t.Fatalf("NewHTTPStreamPlayer: %v", err)
	}
	if err := p.PlayText(context.Background(), app.PlayMeta{}); err == nil {
		t.Fatal("expected error when PlayText is called before Start()")
	}
}

// PlayText で未登録 SpeakerID は `話者#<ID>` にフォールバックする。
func TestHTTPStreamPlayer_PlayText_UnknownSpeakerFallback(t *testing.T) {
	t.Parallel()
	p, err := NewHTTPStreamPlayer("127.0.0.1:0", newTestStreamAssets())
	if err != nil {
		t.Fatalf("NewHTTPStreamPlayer: %v", err)
	}
	p.SetSilent("reason")
	p.silentInterval = 10 * time.Millisecond
	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = p.Shutdown(ctx)
	})

	baseURL := "http://" + p.Addr()
	events := subscribeSSE(t, baseURL)
	time.Sleep(50 * time.Millisecond)
	if err := p.PlayText(context.Background(), app.PlayMeta{Text: "hello", SpeakerID: 42}); err != nil {
		t.Fatalf("PlayText: %v", err)
	}
	select {
	case data := <-events:
		var ev clipEvent
		if err := json.Unmarshal([]byte(data), &ev); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if ev.SpeakerName != "話者#42" {
			t.Errorf("expected fallback SpeakerName '話者#42', got %q", ev.SpeakerName)
		}
		if ev.StyleName != "" {
			t.Errorf("expected StyleName empty, got %q", ev.StyleName)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no event")
	}
}

// testVoicevoxClient is a minimal stub implementing app.VoicevoxClient for /test-clip tests.
type testVoicevoxClient struct {
	createQueryCall int
	synthesizeCall  int
	createQueryErr  error
	synthesizeErr   error
	wav             []byte
	capturedText    string
	capturedSpeaker int
}

func (c *testVoicevoxClient) HealthCheck(_ context.Context) error { return nil }
func (c *testVoicevoxClient) CreateQuery(_ context.Context, text string, speakerID int) (*entity.AudioQuery, error) {
	c.createQueryCall++
	c.capturedText = text
	c.capturedSpeaker = speakerID
	if c.createQueryErr != nil {
		return nil, c.createQueryErr
	}
	return &entity.AudioQuery{}, nil
}
func (c *testVoicevoxClient) Synthesize(_ context.Context, _ *entity.AudioQuery, _ int) ([]byte, error) {
	c.synthesizeCall++
	if c.synthesizeErr != nil {
		return nil, c.synthesizeErr
	}
	return c.wav, nil
}
func (c *testVoicevoxClient) GetSpeakers(_ context.Context) ([]entity.Speaker, error) {
	return nil, nil
}

func TestHTTPStreamPlayer_TestClip_NoClientReturns503(t *testing.T) {
	t.Parallel()
	lookup := map[int]entity.SpeakerStyleInfo{3: {SpeakerName: "ずんだもん", StyleName: "ノーマル"}}
	p := newStartedPlayerWithOpts(t, WithSpeakerLookup(lookup))
	resp, err := http.Get("http://" + p.Addr() + "/test-clip?speaker=3")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", resp.StatusCode)
	}
}

func TestHTTPStreamPlayer_TestClip_BadRequest(t *testing.T) {
	t.Parallel()
	lookup := map[int]entity.SpeakerStyleInfo{3: {SpeakerName: "ずんだもん", StyleName: "ノーマル"}}
	stub := &testVoicevoxClient{wav: []byte("RIFFxxx")}
	p := newStartedPlayerWithOpts(t, WithSpeakerLookup(lookup), WithVoicevoxClient(stub))
	baseURL := "http://" + p.Addr()

	cases := []struct {
		name string
		url  string
	}{
		{"missing", baseURL + "/test-clip"},
		{"empty", baseURL + "/test-clip?speaker="},
		{"non-numeric", baseURL + "/test-clip?speaker=abc"},
		{"unknown-id", baseURL + "/test-clip?speaker=999"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := http.Get(tc.url)
			if err != nil {
				t.Fatalf("GET: %v", err)
			}
			_ = resp.Body.Close()
			if resp.StatusCode != http.StatusBadRequest {
				t.Errorf("expected 400, got %d", resp.StatusCode)
			}
		})
	}
	if stub.createQueryCall != 0 || stub.synthesizeCall != 0 {
		t.Errorf("expected VoicevoxClient not called on bad requests, got createQuery=%d synth=%d",
			stub.createQueryCall, stub.synthesizeCall)
	}
}

func TestHTTPStreamPlayer_TestClip_SuccessAndCached(t *testing.T) {
	t.Parallel()
	lookup := map[int]entity.SpeakerStyleInfo{3: {SpeakerName: "ずんだもん", StyleName: "ノーマル"}}
	stub := &testVoicevoxClient{wav: []byte("RIFFtestwav")}
	p := newStartedPlayerWithOpts(t,
		WithSpeakerLookup(lookup),
		WithVoicevoxClient(stub),
		WithTestPhrase("テスト発話"),
	)
	baseURL := "http://" + p.Addr()

	// 1回目: VoicevoxClient を呼び WAV を返す
	resp1, err := http.Get(baseURL + "/test-clip?speaker=3")
	if err != nil {
		t.Fatalf("first GET: %v", err)
	}
	body1, _ := io.ReadAll(resp1.Body)
	_ = resp1.Body.Close()
	if resp1.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp1.StatusCode)
	}
	if ct := resp1.Header.Get("Content-Type"); !strings.Contains(ct, "audio/wav") {
		t.Errorf("unexpected Content-Type: %s", ct)
	}
	if !bytes.Equal(body1, []byte("RIFFtestwav")) {
		t.Errorf("unexpected body: %q", body1)
	}
	if stub.capturedText != "テスト発話" {
		t.Errorf("expected captured text=テスト発話, got %q", stub.capturedText)
	}
	if stub.capturedSpeaker != 3 {
		t.Errorf("expected captured speaker=3, got %d", stub.capturedSpeaker)
	}
	if stub.createQueryCall != 1 || stub.synthesizeCall != 1 {
		t.Errorf("expected single synth on first call: createQuery=%d synth=%d",
			stub.createQueryCall, stub.synthesizeCall)
	}

	// 2回目: キャッシュから返る
	resp2, err := http.Get(baseURL + "/test-clip?speaker=3")
	if err != nil {
		t.Fatalf("second GET: %v", err)
	}
	body2, _ := io.ReadAll(resp2.Body)
	_ = resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp2.StatusCode)
	}
	if !bytes.Equal(body2, []byte("RIFFtestwav")) {
		t.Errorf("cached body mismatch: %q", body2)
	}
	if stub.createQueryCall != 1 || stub.synthesizeCall != 1 {
		t.Errorf("expected cache hit, VoicevoxClient should not be called again: createQuery=%d synth=%d",
			stub.createQueryCall, stub.synthesizeCall)
	}
}

func TestHTTPStreamPlayer_TestClip_SynthesizeFailureReturns502(t *testing.T) {
	t.Parallel()
	lookup := map[int]entity.SpeakerStyleInfo{3: {SpeakerName: "ずんだもん", StyleName: "ノーマル"}}
	stub := &testVoicevoxClient{synthesizeErr: errors.New("engine down")}
	p := newStartedPlayerWithOpts(t, WithSpeakerLookup(lookup), WithVoicevoxClient(stub))
	resp, err := http.Get("http://" + p.Addr() + "/test-clip?speaker=3")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusBadGateway {
		t.Errorf("expected 502, got %d", resp.StatusCode)
	}
}

func TestHTTPStreamPlayer_TestClip_CreateQueryFailureReturns502(t *testing.T) {
	t.Parallel()
	lookup := map[int]entity.SpeakerStyleInfo{3: {SpeakerName: "ずんだもん", StyleName: "ノーマル"}}
	stub := &testVoicevoxClient{createQueryErr: errors.New("bad")}
	p := newStartedPlayerWithOpts(t, WithSpeakerLookup(lookup), WithVoicevoxClient(stub))
	resp, err := http.Get("http://" + p.Addr() + "/test-clip?speaker=3")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusBadGateway {
		t.Errorf("expected 502, got %d", resp.StatusCode)
	}
}

// --- #285 サーバー側エラー配信 ---

func TestHTTPStreamPlayer_BroadcastError_SynthesisCategory(t *testing.T) {
	t.Parallel()
	lookup := map[int]entity.SpeakerStyleInfo{
		3: {SpeakerName: "ずんだもん", StyleName: "ノーマル"},
	}
	fixed := time.Date(2026, 4, 24, 12, 0, 0, 456_000_000, time.UTC)
	p := newStartedPlayerWithOpts(t,
		WithSpeakerLookup(lookup),
		withNowFunc(func() time.Time { return fixed }),
	)
	baseURL := "http://" + p.Addr()

	events := subscribeSSEByEvent(t, baseURL, "error")
	time.Sleep(50 * time.Millisecond)

	p.BroadcastError(app.StreamError{
		Category:  app.StreamErrorCategorySynthesis,
		Message:   "synthesize failed",
		Path:      "/tmp/script.txt",
		Text:      "こんにちは",
		SpeakerID: 3,
	})

	select {
	case data := <-events:
		var ev errorEvent
		if err := json.Unmarshal([]byte(data), &ev); err != nil {
			t.Fatalf("unmarshal: %v data=%s", err, data)
		}
		if ev.ID != 1 {
			t.Errorf("expected id=1, got %d", ev.ID)
		}
		if ev.Category != "synthesis" {
			t.Errorf("expected category=synthesis, got %q", ev.Category)
		}
		if ev.Message != "synthesize failed" {
			t.Errorf("unexpected message: %q", ev.Message)
		}
		if ev.Path != "/tmp/script.txt" {
			t.Errorf("unexpected path: %q", ev.Path)
		}
		if ev.Text != "こんにちは" {
			t.Errorf("unexpected text: %q", ev.Text)
		}
		if ev.SpeakerName != "ずんだもん" {
			t.Errorf("expected speakerName=ずんだもん, got %q", ev.SpeakerName)
		}
		if ev.StyleName != "ノーマル" {
			t.Errorf("expected styleName=ノーマル, got %q", ev.StyleName)
		}
		if ev.Timestamp != fixed.UnixMilli() {
			t.Errorf("expected timestamp=%d, got %d", fixed.UnixMilli(), ev.Timestamp)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for error event")
	}
}

func TestHTTPStreamPlayer_BroadcastError_FileCategory(t *testing.T) {
	t.Parallel()
	p := newStartedPlayer(t)
	baseURL := "http://" + p.Addr()

	events := subscribeSSEByEvent(t, baseURL, "error")
	time.Sleep(50 * time.Millisecond)

	p.BroadcastError(app.StreamError{
		Category: app.StreamErrorCategoryFile,
		Message:  "failed to read script",
		Path:     "/tmp/script.txt",
	})

	select {
	case data := <-events:
		var ev errorEvent
		if err := json.Unmarshal([]byte(data), &ev); err != nil {
			t.Fatalf("unmarshal: %v data=%s", err, data)
		}
		if ev.Category != "file" {
			t.Errorf("expected category=file, got %q", ev.Category)
		}
		if ev.Path != "/tmp/script.txt" {
			t.Errorf("unexpected path: %q", ev.Path)
		}
		if ev.Text != "" {
			t.Errorf("expected empty text, got %q", ev.Text)
		}
		if ev.SpeakerName != "" {
			t.Errorf("expected empty speakerName, got %q", ev.SpeakerName)
		}
		if ev.StyleName != "" {
			t.Errorf("expected empty styleName, got %q", ev.StyleName)
		}
		// 省略フィールドは JSON 上に出現しないこと（omitempty の確認）
		if strings.Contains(data, `"text":`) {
			t.Errorf("expected text to be omitted for file category, got: %s", data)
		}
		if strings.Contains(data, `"speakerName":`) {
			t.Errorf("expected speakerName to be omitted for file category, got: %s", data)
		}
		if strings.Contains(data, `"styleName":`) {
			t.Errorf("expected styleName to be omitted for file category, got: %s", data)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for error event")
	}
}

func TestHTTPStreamPlayer_BroadcastError_ConnectionCategory(t *testing.T) {
	t.Parallel()
	p := newStartedPlayer(t)
	baseURL := "http://" + p.Addr()

	events := subscribeSSEByEvent(t, baseURL, "error")
	time.Sleep(50 * time.Millisecond)

	p.BroadcastError(app.StreamError{
		Category: app.StreamErrorCategoryConnection,
		Message:  "VOICEVOX unreachable",
	})

	select {
	case data := <-events:
		var ev errorEvent
		if err := json.Unmarshal([]byte(data), &ev); err != nil {
			t.Fatalf("unmarshal: %v data=%s", err, data)
		}
		if ev.Category != "connection" {
			t.Errorf("expected category=connection, got %q", ev.Category)
		}
		if ev.Message != "VOICEVOX unreachable" {
			t.Errorf("unexpected message: %q", ev.Message)
		}
		if strings.Contains(data, `"path":`) {
			t.Errorf("expected path to be omitted for connection category, got: %s", data)
		}
		if strings.Contains(data, `"text":`) {
			t.Errorf("expected text to be omitted for connection category, got: %s", data)
		}
		if strings.Contains(data, `"speakerName":`) {
			t.Errorf("expected speakerName to be omitted for connection category, got: %s", data)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for error event")
	}
}

func TestHTTPStreamPlayer_BroadcastError_IDMonotonicAndIndependentFromClip(t *testing.T) {
	t.Parallel()
	p := newStartedPlayer(t)
	baseURL := "http://" + p.Addr()

	clipEvents := subscribeSSEByEvent(t, baseURL, "clip")
	errorEvents := subscribeSSEByEvent(t, baseURL, "error")
	time.Sleep(80 * time.Millisecond)

	// clip を 2 回、error を 3 回交互に配信して、それぞれが独立に 1 始まりで連番になること
	if err := p.Play(context.Background(), []byte("RIFFa"), app.PlayMeta{}); err != nil {
		t.Fatalf("Play: %v", err)
	}
	p.BroadcastError(app.StreamError{Category: app.StreamErrorCategoryFile, Message: "e1"})
	p.BroadcastError(app.StreamError{Category: app.StreamErrorCategoryFile, Message: "e2"})
	if err := p.Play(context.Background(), []byte("RIFFb"), app.PlayMeta{}); err != nil {
		t.Fatalf("Play: %v", err)
	}
	p.BroadcastError(app.StreamError{Category: app.StreamErrorCategoryFile, Message: "e3"})

	for i := 1; i <= 2; i++ {
		select {
		case data := <-clipEvents:
			needle := fmt.Sprintf(`"id":%d`, i)
			if !strings.Contains(data, needle) {
				t.Errorf("clip event #%d: expected %s, got %s", i, needle, data)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("timeout waiting for clip #%d", i)
		}
	}
	for i := 1; i <= 3; i++ {
		select {
		case data := <-errorEvents:
			needle := fmt.Sprintf(`"id":%d`, i)
			if !strings.Contains(data, needle) {
				t.Errorf("error event #%d: expected %s, got %s", i, needle, data)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("timeout waiting for error #%d", i)
		}
	}
}

func TestHTTPStreamPlayer_BroadcastError_MultipleSubscribers(t *testing.T) {
	t.Parallel()
	p := newStartedPlayer(t)
	baseURL := "http://" + p.Addr()

	const subscribers = 3
	channels := make([]<-chan string, subscribers)
	for i := range subscribers {
		channels[i] = subscribeSSEByEvent(t, baseURL, "error")
	}
	time.Sleep(80 * time.Millisecond)

	p.BroadcastError(app.StreamError{Category: app.StreamErrorCategoryFile, Message: "oh"})

	for i, ch := range channels {
		select {
		case data := <-ch:
			if !strings.Contains(data, `"id":1`) {
				t.Errorf("subscriber %d got %s", i, data)
			}
		case <-time.After(2 * time.Second):
			t.Errorf("subscriber %d timeout", i)
		}
	}
}

// HTTPStreamPlayer は app.ErrorBroadcaster を実装する。
func TestHTTPStreamPlayer_ImplementsErrorBroadcaster(t *testing.T) {
	t.Parallel()
	var _ app.ErrorBroadcaster = (*HTTPStreamPlayer)(nil)
}

// #323 キャラクタータブ / 口パク:
// DONE: GET /api/characters が enabled=false, characters=[] を返す（workspacePath 未設定の場合）
// DONE: GET /api/characters が有効な場合に enabled=true と characters 配列を返す
// DONE: loadErr != nil のとき workspacePath と assetsDir を含む Warn ログが出る
// DONE: 読み込み成功時に件数を含む Info ログが出る
// TODO: settings.json が存在しない場合に enabled=false を返す
// TODO: settings.json のパース失敗時に enabled=false を返す
// TODO: 画像パスに `..` を含む場合は該当エントリを無視し enabled=false になる
// TODO: 画像ファイル不在の場合は該当エントリを無視する
// TODO: 全エントリが無効な場合は enabled=false を返す
// TODO: GET /assets/images/<relative-path> が画像ファイルを配信する
// TODO: GET /assets/images/<relative-path> でパス検証（..や先頭/拒否）
// TODO: speakerName + styleName が重複する場合はエラー（startup 時）
// TODO: ミュート中でも画像キャッシュは読み込まれる（lazy load on start）

func TestHTTPStreamPlayer_APICharacters_DisabledWhenNoWorkspacePath(t *testing.T) {
	t.Parallel()
	p := newStartedPlayer(t)
	resp, err := http.Get("http://" + p.Addr() + "/api/characters")
	if err != nil {
		t.Fatalf("GET /api/characters: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: %d", resp.StatusCode)
	}
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	enabled, ok := result["enabled"].(bool)
	if !ok || enabled {
		t.Errorf("expected enabled=false, got %v", result)
	}
	chars, ok := result["characters"].([]interface{})
	if !ok || len(chars) != 0 {
		t.Errorf("expected characters=[], got %v", result)
	}
}

func TestLoadCharacterSettingsFromSpeakerJSON_SingleValidSpeaker(t *testing.T) {
	t.Parallel()
	// Create a temporary workspace with assets/speaker-name/speaker.json
	fsys := fstest.MapFS{
		"assets/zundamon/speaker.json": {
			Data: []byte(`{
				"speakerName": "ずんだもん",
				"styles": [
					{
						"styleName": "ノーマル",
						"mouthClosed": "zundamon/normal_closed.png",
						"mouthOpened": "zundamon/normal_opened.png"
					}
				]
			}`),
		},
		"assets/zundamon/normal_closed.png": {Data: []byte("")},
		"assets/zundamon/normal_opened.png": {Data: []byte("")},
	}

	entries, err := loadCharacterSettingsFromSpeakerJSON(fsys, "/", "assets", slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("loadCharacterSettingsFromSpeakerJSON failed: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].SpeakerName != "ずんだもん" {
		t.Errorf("expected speakerName='ずんだもん', got %q", entries[0].SpeakerName)
	}
	if entries[0].StyleName != "ノーマル" {
		t.Errorf("expected styleName='ノーマル', got %q", entries[0].StyleName)
	}
	if entries[0].MouthClosed != "zundamon/normal_closed.png" {
		t.Errorf("expected mouthClosed='zundamon/normal_closed.png', got %q", entries[0].MouthClosed)
	}
	if entries[0].MouthOpen != "zundamon/normal_opened.png" {
		t.Errorf("expected mouthOpen='zundamon/normal_opened.png', got %q", entries[0].MouthOpen)
	}
}

func TestLoadCharacterSettingsFromSpeakerJSON_MultipleStyles(t *testing.T) {
	t.Parallel()
	fsys := fstest.MapFS{
		"assets/zundamon/speaker.json": {
			Data: []byte(`{
				"speakerName": "ずんだもん",
				"styles": [
					{
						"styleName": "ノーマル",
						"mouthClosed": "zundamon/normal_closed.png",
						"mouthOpened": "zundamon/normal_opened.png"
					},
					{
						"styleName": "喜び",
						"mouthClosed": "zundamon/happy_closed.png",
						"mouthOpened": "zundamon/happy_opened.png"
					}
				]
			}`),
		},
		"assets/zundamon/normal_closed.png": {Data: []byte("")},
		"assets/zundamon/normal_opened.png": {Data: []byte("")},
		"assets/zundamon/happy_closed.png":  {Data: []byte("")},
		"assets/zundamon/happy_opened.png":  {Data: []byte("")},
	}

	entries, err := loadCharacterSettingsFromSpeakerJSON(fsys, "/", "assets", slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("loadCharacterSettingsFromSpeakerJSON failed: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].StyleName != "ノーマル" {
		t.Errorf("expected first style to be ノーマル, got %q", entries[0].StyleName)
	}
	if entries[1].StyleName != "喜び" {
		t.Errorf("expected second style to be 喜び, got %q", entries[1].StyleName)
	}
}

func TestLoadCharacterSettingsFromSpeakerJSON_NoSpeakerJSON(t *testing.T) {
	t.Parallel()
	fsys := fstest.MapFS{}

	entries, err := loadCharacterSettingsFromSpeakerJSON(fsys, "/", "assets", slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err == nil {
		t.Fatal("expected error when no speaker.json found")
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestLoadCharacterSettingsFromSpeakerJSON_SkipInvalidSpeakerJSON(t *testing.T) {
	t.Parallel()
	fsys := fstest.MapFS{
		"assets/invalid/speaker.json": {
			Data: []byte(`{invalid json}`),
		},
		"assets/zundamon/speaker.json": {
			Data: []byte(`{
				"speakerName": "ずんだもん",
				"styles": [
					{
						"styleName": "ノーマル",
						"mouthClosed": "zundamon/normal_closed.png",
						"mouthOpened": "zundamon/normal_opened.png"
					}
				]
			}`),
		},
		"assets/zundamon/normal_closed.png": {Data: []byte("")},
		"assets/zundamon/normal_opened.png": {Data: []byte("")},
	}

	entries, err := loadCharacterSettingsFromSpeakerJSON(fsys, "/", "assets", slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("loadCharacterSettingsFromSpeakerJSON failed: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry (valid one), got %d", len(entries))
	}
	if entries[0].SpeakerName != "ずんだもん" {
		t.Errorf("expected valid speaker to be loaded")
	}
}

func TestLoadCharacterSettingsFromSpeakerJSON_SkipMissingImageFiles(t *testing.T) {
	t.Parallel()
	fsys := fstest.MapFS{
		"assets/zundamon/speaker.json": {
			Data: []byte(`{
				"speakerName": "ずんだもん",
				"styles": [
					{
						"styleName": "ノーマル",
						"mouthClosed": "zundamon/normal_closed.png",
						"mouthOpened": "zundamon/normal_opened.png"
					}
				]
			}`),
		},
	}

	entries, err := loadCharacterSettingsFromSpeakerJSON(fsys, "/", "assets", slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err == nil {
		t.Fatal("expected error when image files don't exist")
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries (invalid entry), got %d", len(entries))
	}
}

func newValidAssetsWorkspace(t *testing.T) string {
	t.Helper()
	workspaceDir := t.TempDir()
	speakerDir := filepath.Join(workspaceDir, "assets", "zundamon")
	if err := os.MkdirAll(speakerDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	speakerJSON := `{"speakerName":"ずんだもん","styles":[{"styleName":"ノーマル","mouthClosed":"zundamon/normal_closed.png","mouthOpened":"zundamon/normal_opened.png"}]}`
	if err := os.WriteFile(filepath.Join(speakerDir, "speaker.json"), []byte(speakerJSON), 0o644); err != nil {
		t.Fatalf("WriteFile speaker.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(speakerDir, "normal_closed.png"), []byte(""), 0o644); err != nil {
		t.Fatalf("WriteFile normal_closed.png: %v", err)
	}
	if err := os.WriteFile(filepath.Join(speakerDir, "normal_opened.png"), []byte(""), 0o644); err != nil {
		t.Fatalf("WriteFile normal_opened.png: %v", err)
	}
	return workspaceDir
}

func TestHTTPStreamPlayer_APICharacters_EnabledWithValidAssets(t *testing.T) {
	t.Parallel()
	workspaceDir := newValidAssetsWorkspace(t)
	p, err := NewHTTPStreamPlayer("127.0.0.1:0", newTestStreamAssets(), WithWorkspacePath(workspaceDir))
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

	resp, err := http.Get("http://" + p.Addr() + "/api/characters")
	if err != nil {
		t.Fatalf("GET /api/characters: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: %d", resp.StatusCode)
	}
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	enabled, ok := result["enabled"].(bool)
	if !ok || !enabled {
		t.Errorf("expected enabled=true, got %v", result)
	}
	chars, ok := result["characters"].([]interface{})
	if !ok || len(chars) == 0 {
		t.Errorf("expected non-empty characters, got %v", result)
	}
}

func TestBuildAPICharactersJSON_LogsWarnOnLoadError(t *testing.T) {
	t.Parallel()
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	workspaceDir := t.TempDir() // no assets/ subdir

	p, err := NewHTTPStreamPlayer("127.0.0.1:0", newTestStreamAssets(),
		WithHTTPStreamLogger(logger),
		WithWorkspacePath(workspaceDir),
	)
	if err != nil {
		t.Fatalf("NewHTTPStreamPlayer: %v", err)
	}
	if err := p.buildAPICharactersJSON(); err != nil {
		t.Fatalf("buildAPICharactersJSON: %v", err)
	}

	logOutput := logBuf.String()
	if !strings.Contains(logOutput, "WARN") {
		t.Errorf("expected WARN log, got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "failed to load character settings") {
		t.Errorf("expected warn message in log, got: %s", logOutput)
	}
	if !strings.Contains(logOutput, workspaceDir) {
		t.Errorf("expected workspacePath in log, got: %s", logOutput)
	}
}

func TestBuildAPICharactersJSON_LogsInfoOnSuccess(t *testing.T) {
	t.Parallel()
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	workspaceDir := newValidAssetsWorkspace(t)
	p, err := NewHTTPStreamPlayer("127.0.0.1:0", newTestStreamAssets(),
		WithHTTPStreamLogger(logger),
		WithWorkspacePath(workspaceDir),
	)
	if err != nil {
		t.Fatalf("NewHTTPStreamPlayer: %v", err)
	}
	if err := p.buildAPICharactersJSON(); err != nil {
		t.Fatalf("buildAPICharactersJSON: %v", err)
	}

	logOutput := logBuf.String()
	if !strings.Contains(logOutput, "INFO") {
		t.Errorf("expected INFO log, got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "character settings loaded") {
		t.Errorf("expected 'character settings loaded' in log, got: %s", logOutput)
	}
}
