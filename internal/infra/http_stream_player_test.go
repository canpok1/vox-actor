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
// DONE: 未登録の /clips/{timestamp}.wav は 404
//
// Play / キュー / SSE:
// DONE: Play() で WAV がキューに登録され GET /clips/{timestamp}.wav で配信される
// DONE: clipEvent の timestamp は一意かつ非ゼロ
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
// #226 backpressure (廃止: #487 で worker に移動):
// DELETED: Play() が WAV 推定再生時間ぶんブロックしてから return する → worker が担当
// DELETED: ctx キャンセル時は sleep を中断し ctx.Err() を返す → worker が担当
// DONE: WAVヘッダ不正時は warning を出して正常 return する
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
//
// #418 /api/play エンドポイント:
// DONE: GET /api/play は 405 を返す                               → TestHTTPStreamPlayer_APIPlay_MethodNotAllowed
// DONE: voicevoxClient 未設定時は 503 を返す                      → TestHTTPStreamPlayer_APIPlay_NoClient
// DONE: 不正 JSON ボディで 400 を返す                             → TestHTTPStreamPlayer_APIPlay_InvalidJSON
// DONE: 正常リクエストで合成→Play して 200 {silent:false} を返す → TestHTTPStreamPlayer_APIPlay_Success
// DONE: 無音モード時は合成せず 200 {silent:true,silent_reason} を返す → TestHTTPStreamPlayer_APIPlay_SilentMode
// DONE: 合成失敗時に 502 を返す                                   → TestHTTPStreamPlayer_APIPlay_SynthesisFails
//
// #408 再生履歴ファイル保存・起動時復元:
// TODO: Play() で clip が YYYY-MM-DD.jsonl に追記される          → TestHTTPStreamPlayer_History_WritesClipOnPlay
// TODO: PlayText() で clip が YYYY-MM-DD.jsonl に追記される      → TestHTTPStreamPlayer_History_WritesClipOnPlayText
// TODO: 日付変更時に新ファイルへ切り替わる                       → TestHTTPStreamPlayer_History_RotatesOnDateChange
// TODO: viewer 起動時に 30日より古い *.jsonl が削除される        → TestHTTPStreamPlayer_History_PrunesOldFiles
// TODO: 当日ファイルなし時も正常起動                             → TestHTTPStreamPlayer_Start_NoHistoryFile
// TODO: /api/history が当日末尾 50 件を返す                      → TestHTTPStreamPlayer_APIHistory_ReturnsTodaysEntries
// TODO: /api/history がファイルなし時に空配列を返す              → TestHTTPStreamPlayer_APIHistory_EmptyWhenNoFile

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

	var clipURL string
	select {
	case data, ok := <-events:
		if !ok {
			t.Fatal("events channel closed before clip delivered")
		}
		var ev clipEvent
		if err := json.Unmarshal([]byte(data), &ev); err != nil {
			t.Fatalf("unmarshal clipEvent: %v, data=%s", err, data)
		}
		if ev.Timestamp == 0 {
			t.Errorf("expected non-zero timestamp, got: %s", data)
		}
		if !strings.HasPrefix(ev.URL, clipPathPrefix) || !strings.HasSuffix(ev.URL, clipPathSuffix) {
			t.Errorf("expected url like /clips/<timestamp>.wav, got: %s", ev.URL)
		}
		clipURL = ev.URL
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for clip event")
	}

	resp, err := http.Get(baseURL + clipURL)
	if err != nil {
		t.Fatalf("GET %s: %v", clipURL, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("clip status: %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "audio/wav") {
		t.Errorf("clip content-type: %s", ct)
	}
}

func TestHTTPStreamPlayer_ClipTimestampIsNonZeroAndNoIDField(t *testing.T) {
	t.Parallel()
	fixedBase := time.Date(2026, 4, 28, 10, 0, 0, 0, time.Local)
	callCount := 0
	p, err := NewHTTPStreamPlayer("127.0.0.1:0", newTestStreamAssets(),
		withNowFunc(func() time.Time {
			callCount++
			return fixedBase.Add(time.Duration(callCount) * time.Millisecond)
		}),
	)
	if err != nil {
		t.Fatalf("NewHTTPStreamPlayer: %v", err)
	}
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

	for range 3 {
		if err := p.Play(context.Background(), []byte("RIFFx"), app.PlayMeta{}); err != nil {
			t.Fatalf("Play: %v", err)
		}
	}

	seen := make(map[int64]bool)
	for i := range 3 {
		select {
		case data := <-events:
			var ev clipEvent
			if err := json.Unmarshal([]byte(data), &ev); err != nil {
				t.Fatalf("event #%d unmarshal: %v, data=%s", i+1, err, data)
			}
			if ev.Timestamp == 0 {
				t.Errorf("event #%d: expected non-zero timestamp, got %s", i+1, data)
			}
			if seen[ev.Timestamp] {
				t.Errorf("event #%d: duplicate timestamp %d", i+1, ev.Timestamp)
			}
			seen[ev.Timestamp] = true
			// id フィールドが JSON に含まれないことを確認する
			if strings.Contains(data, `"id":`) {
				t.Errorf("event #%d: unexpected id field in clip payload: %s", i+1, data)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("timeout waiting for event #%d", i+1)
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
			if !strings.Contains(data, `"timestamp":`) {
				t.Errorf("subscriber %d: expected timestamp field, got %s", i, data)
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

// --- #226 backpressure (backpressure は #487 で worker に移動) ---

func TestHTTPStreamPlayer_Play_InvalidWAVHeaderSucceeds(t *testing.T) {
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
	// (#487 以降 backpressure は worker に移動したため、Play() はログを出さず即 return)
	if err := p.Play(context.Background(), []byte("not-a-valid-wav-data"), app.PlayMeta{}); err != nil {
		t.Fatalf("Play: %v", err)
	}
	_ = logBuf // logger は将来の WARN チェック追加のために保持
}

// --- #226 履歴サイズ固定値（20）の検証 ---

func TestHTTPStreamPlayer_RingBuffer_FixedSize(t *testing.T) {
	t.Parallel()
	fixedBase := time.Date(2026, 4, 28, 10, 0, 0, 0, time.Local)
	callCount := 0
	p, err := NewHTTPStreamPlayer("127.0.0.1:0", newTestStreamAssets(),
		withNowFunc(func() time.Time {
			callCount++
			return fixedBase.Add(time.Duration(callCount) * time.Millisecond)
		}),
	)
	if err != nil {
		t.Fatalf("NewHTTPStreamPlayer: %v", err)
	}
	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = p.Shutdown(ctx)
	})
	baseURL := "http://" + p.Addr()

	// 21件積むと1件目だけが押し出される（容量20）。
	// nowFunc は Play 毎に 1ms 増加するため、URL は /clips/<base+N>.wav で予測可能。
	const total = 21
	for range total {
		if err := p.Play(context.Background(), []byte("RIFFx"), app.PlayMeta{}); err != nil {
			t.Fatalf("Play: %v", err)
		}
	}
	baseMs := fixedBase.UnixMilli()
	clipURL := func(n int) string {
		return fmt.Sprintf("/clips/%d.wav", baseMs+int64(n))
	}

	// 1件目は押し出されて 404
	resp1, err := http.Get(baseURL + clipURL(1))
	if err != nil {
		t.Fatalf("GET clip1: %v", err)
	}
	_ = resp1.Body.Close()
	if resp1.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 for evicted clip 1, got %d", resp1.StatusCode)
	}

	// 2件目（最古の生存クリップ）と21件目（最新）は取得できる
	resp2, err := http.Get(baseURL + clipURL(2))
	if err != nil {
		t.Fatalf("GET clip2: %v", err)
	}
	_ = resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Errorf("expected 200 for clip 2 (oldest surviving), got %d", resp2.StatusCode)
	}

	resp21, err := http.Get(baseURL + clipURL(total))
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
	createQueryCall  int
	synthesizeCall   int
	createQueryErr   error
	synthesizeErr    error
	wav              []byte
	capturedText     string
	capturedSpeaker  int
	createQueryBlock chan struct{} // nil でなければ CreateQuery はこのチャネルが閉じるまでブロックする
}

func (c *testVoicevoxClient) HealthCheck(_ context.Context) error { return nil }
func (c *testVoicevoxClient) CreateQuery(ctx context.Context, text string, speakerID int) (*entity.AudioQuery, error) {
	c.createQueryCall++
	c.capturedText = text
	c.capturedSpeaker = speakerID
	if c.createQueryBlock != nil {
		select {
		case <-c.createQueryBlock:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
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

	// clip を 2 回、error を 3 回交互に配信して、error は独立に 1 始まりで連番になること
	if err := p.Play(context.Background(), []byte("RIFFa"), app.PlayMeta{}); err != nil {
		t.Fatalf("Play: %v", err)
	}
	p.BroadcastError(app.StreamError{Category: app.StreamErrorCategoryFile, Message: "e1"})
	p.BroadcastError(app.StreamError{Category: app.StreamErrorCategoryFile, Message: "e2"})
	if err := p.Play(context.Background(), []byte("RIFFb"), app.PlayMeta{}); err != nil {
		t.Fatalf("Play: %v", err)
	}
	p.BroadcastError(app.StreamError{Category: app.StreamErrorCategoryFile, Message: "e3"})

	for i := range 2 {
		select {
		case data := <-clipEvents:
			if !strings.Contains(data, `"timestamp":`) {
				t.Errorf("clip event #%d: expected timestamp field, got %s", i+1, data)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("timeout waiting for clip #%d", i+1)
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
	// Create a temporary workspace with assets/speaker-name/speaker.json.
	// speaker.json paths are filename-only (relative to the speaker directory),
	// matching the actual format used in .vox-actor/assets/*/speaker.json.
	fsys := fstest.MapFS{
		"assets/zundamon/speaker.json": {
			Data: []byte(`{
				"speakerName": "ずんだもん",
				"styles": [
					{
						"styleName": "ノーマル",
						"mouthClosed": "normal_closed.png",
						"mouthOpened": "normal_opened.png"
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
						"mouthClosed": "normal_closed.png",
						"mouthOpened": "normal_opened.png"
					},
					{
						"styleName": "喜び",
						"mouthClosed": "happy_closed.png",
						"mouthOpened": "happy_opened.png"
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
						"mouthClosed": "normal_closed.png",
						"mouthOpened": "normal_opened.png"
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
						"mouthClosed": "normal_closed.png",
						"mouthOpened": "normal_opened.png"
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

// TestLoadCharacterSettingsFromSpeakerJSON_PathRelativeToSpeakerDir verifies that
// mouthClosed/mouthOpened paths in speaker.json are treated as relative to the speaker
// directory, not the assets directory root. Real speaker.json files use filename-only
// paths (e.g. "ankomon_normal_close.png"), not paths prefixed with the speaker dir.
func TestLoadCharacterSettingsFromSpeakerJSON_PathRelativeToSpeakerDir(t *testing.T) {
	t.Parallel()
	// speaker.json uses filename-only paths (relative to its own directory),
	// matching the actual format used in .vox-actor/assets/*/speaker.json.
	fsys := fstest.MapFS{
		"assets/zundamon/speaker.json": {
			Data: []byte(`{
				"speakerName": "ずんだもん",
				"styles": [
					{
						"styleName": "ノーマル",
						"mouthClosed": "normal_closed.png",
						"mouthOpened": "normal_opened.png"
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
	if entries[0].MouthClosed != "zundamon/normal_closed.png" {
		t.Errorf("expected mouthClosed='zundamon/normal_closed.png', got %q", entries[0].MouthClosed)
	}
	if entries[0].MouthOpen != "zundamon/normal_opened.png" {
		t.Errorf("expected mouthOpen='zundamon/normal_opened.png', got %q", entries[0].MouthOpen)
	}
}

func newValidAssetsWorkspace(t *testing.T) string {
	t.Helper()
	workspaceDir := t.TempDir()
	speakerDir := filepath.Join(workspaceDir, "assets", "zundamon")
	if err := os.MkdirAll(speakerDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	speakerJSON := `{"speakerName":"ずんだもん","styles":[{"styleName":"ノーマル","mouthClosed":"normal_closed.png","mouthOpened":"normal_opened.png"}]}`
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

	// assets/ dir exists but is empty (no valid entries) → WARN "failed to load character settings"
	workspaceDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(workspaceDir, "assets"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

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

func TestBuildAPICharactersJSON_LogsInfoOnDirNotFound(t *testing.T) {
	t.Parallel()
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	workspaceDir := t.TempDir() // no assets/ subdir → dir not found

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
	if !strings.Contains(logOutput, "assets directory not found, skipping load") {
		t.Errorf("expected info message in log, got: %s", logOutput)
	}
	if !strings.Contains(logOutput, workspaceDir) {
		t.Errorf("expected workspaceDir in log, got: %s", logOutput)
	}
}

// newValidAssetsDir は assets ディレクトリを直接作成する（workspacePath なしで WithAssetsDirs で使用）。
func newValidAssetsDir(t *testing.T) string {
	t.Helper()
	assetsDir := t.TempDir()
	speakerDir := filepath.Join(assetsDir, "zundamon")
	if err := os.MkdirAll(speakerDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	speakerJSON := `{"speakerName":"ずんだもん","styles":[{"styleName":"ノーマル","mouthClosed":"normal_closed.png","mouthOpened":"normal_opened.png"}]}`
	if err := os.WriteFile(filepath.Join(speakerDir, "speaker.json"), []byte(speakerJSON), 0o644); err != nil {
		t.Fatalf("WriteFile speaker.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(speakerDir, "normal_closed.png"), []byte(""), 0o644); err != nil {
		t.Fatalf("WriteFile normal_closed.png: %v", err)
	}
	if err := os.WriteFile(filepath.Join(speakerDir, "normal_opened.png"), []byte(""), 0o644); err != nil {
		t.Fatalf("WriteFile normal_opened.png: %v", err)
	}
	return assetsDir
}

func TestHTTPStreamPlayer_WithAssetsDirs_EnablesCharacters(t *testing.T) {
	t.Parallel()
	assetsDir := newValidAssetsDir(t)
	p, err := NewHTTPStreamPlayer("127.0.0.1:0", newTestStreamAssets(), WithAssetsDirs([]string{assetsDir}))
	if err != nil {
		t.Fatalf("NewHTTPStreamPlayer: %v", err)
	}
	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = p.Shutdown(ctx)
	})

	resp, err := http.Get("http://" + p.Addr() + "/api/characters")
	if err != nil {
		t.Fatalf("GET /api/characters: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if enabled, _ := result["enabled"].(bool); !enabled {
		t.Errorf("expected enabled=true, got %v", result)
	}
}

func TestHTTPStreamPlayer_WithAssetsDirs_MergesProjectAndHome(t *testing.T) {
	t.Parallel()
	projectDir := t.TempDir()
	homeDir := t.TempDir()

	// project: charA
	charADir := filepath.Join(projectDir, "charA")
	if err := os.MkdirAll(charADir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	speakerA := `{"speakerName":"キャラA","styles":[{"styleName":"ノーマル","mouthClosed":"a_closed.png","mouthOpened":"a_opened.png"}]}`
	if err := os.WriteFile(filepath.Join(charADir, "speaker.json"), []byte(speakerA), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	for _, f := range []string{"a_closed.png", "a_opened.png"} {
		if err := os.WriteFile(filepath.Join(charADir, f), []byte(""), 0o644); err != nil {
			t.Fatalf("WriteFile %s: %v", f, err)
		}
	}

	// home: charB
	charBDir := filepath.Join(homeDir, "charB")
	if err := os.MkdirAll(charBDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	speakerB := `{"speakerName":"キャラB","styles":[{"styleName":"ノーマル","mouthClosed":"b_closed.png","mouthOpened":"b_opened.png"}]}`
	if err := os.WriteFile(filepath.Join(charBDir, "speaker.json"), []byte(speakerB), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	for _, f := range []string{"b_closed.png", "b_opened.png"} {
		if err := os.WriteFile(filepath.Join(charBDir, f), []byte(""), 0o644); err != nil {
			t.Fatalf("WriteFile %s: %v", f, err)
		}
	}

	p, err := NewHTTPStreamPlayer("127.0.0.1:0", newTestStreamAssets(),
		WithAssetsDirs([]string{projectDir, homeDir}),
	)
	if err != nil {
		t.Fatalf("NewHTTPStreamPlayer: %v", err)
	}
	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = p.Shutdown(ctx)
	})

	resp, err := http.Get("http://" + p.Addr() + "/api/characters")
	if err != nil {
		t.Fatalf("GET /api/characters: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	chars, _ := result["characters"].([]interface{})
	if len(chars) != 2 {
		t.Errorf("expected 2 characters (merged), got %d: %v", len(chars), chars)
	}
}

func TestHTTPStreamPlayer_WithAssetsDirs_ProjectPriorityOnDuplicateID(t *testing.T) {
	t.Parallel()
	projectDir := t.TempDir()
	homeDir := t.TempDir()

	// Both have "shared" but different speaker names
	for dir, speakerName := range map[string]string{projectDir: "プロジェクト版", homeDir: "ホーム版"} {
		d := filepath.Join(dir, "shared")
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		speakerJSON := `{"speakerName":"` + speakerName + `","styles":[{"styleName":"ノーマル","mouthClosed":"closed.png","mouthOpened":"opened.png"}]}`
		if err := os.WriteFile(filepath.Join(d, "speaker.json"), []byte(speakerJSON), 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
		for _, f := range []string{"closed.png", "opened.png"} {
			if err := os.WriteFile(filepath.Join(d, f), []byte(""), 0o644); err != nil {
				t.Fatalf("WriteFile %s: %v", f, err)
			}
		}
	}

	p, err := NewHTTPStreamPlayer("127.0.0.1:0", newTestStreamAssets(),
		WithAssetsDirs([]string{projectDir, homeDir}),
	)
	if err != nil {
		t.Fatalf("NewHTTPStreamPlayer: %v", err)
	}
	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = p.Shutdown(ctx)
	})

	resp, err := http.Get("http://" + p.Addr() + "/api/characters")
	if err != nil {
		t.Fatalf("GET /api/characters: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	chars, _ := result["characters"].([]interface{})
	if len(chars) != 1 {
		t.Fatalf("expected 1 character (deduplicated), got %d", len(chars))
	}
	char, _ := chars[0].(map[string]interface{})
	if char["speakerName"] != "プロジェクト版" {
		t.Errorf("expected speakerName 'プロジェクト版', got %v", char["speakerName"])
	}
}

func TestHTTPStreamPlayer_HandleCharacterImage_SearchesAcrossAssetsDirs(t *testing.T) {
	t.Parallel()
	projectDir := t.TempDir()
	homeDir := t.TempDir()

	// project: charA with image
	charADir := filepath.Join(projectDir, "charA")
	if err := os.MkdirAll(charADir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	imageContent := []byte("project-image-data")
	if err := os.WriteFile(filepath.Join(charADir, "icon.png"), imageContent, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	speakerA := `{"speakerName":"キャラA","styles":[{"styleName":"ノーマル","mouthClosed":"icon.png","mouthOpened":"icon.png"}]}`
	if err := os.WriteFile(filepath.Join(charADir, "speaker.json"), []byte(speakerA), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// home: charB with image
	charBDir := filepath.Join(homeDir, "charB")
	if err := os.MkdirAll(charBDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	imageBContent := []byte("home-image-data")
	if err := os.WriteFile(filepath.Join(charBDir, "icon.png"), imageBContent, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	speakerB := `{"speakerName":"キャラB","styles":[{"styleName":"ノーマル","mouthClosed":"icon.png","mouthOpened":"icon.png"}]}`
	if err := os.WriteFile(filepath.Join(charBDir, "speaker.json"), []byte(speakerB), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	p, err := NewHTTPStreamPlayer("127.0.0.1:0", newTestStreamAssets(),
		WithAssetsDirs([]string{projectDir, homeDir}),
	)
	if err != nil {
		t.Fatalf("NewHTTPStreamPlayer: %v", err)
	}
	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = p.Shutdown(ctx)
	})

	// image from homeDir's charB should be served
	resp, err := http.Get("http://" + p.Addr() + "/assets/images/charB/icon.png")
	if err != nil {
		t.Fatalf("GET /assets/images/charB/icon.png: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != string(imageBContent) {
		t.Errorf("expected home image content %q, got %q", imageBContent, body)
	}
}

func TestHTTPStreamPlayer_History_WritesClipOnPlay(t *testing.T) {
	historyDir := t.TempDir()
	fixedNow := time.Date(2026, 4, 28, 10, 0, 0, 0, time.Local)
	lookup := map[int]entity.SpeakerStyleInfo{
		3: {SpeakerName: "ずんだもん", StyleName: "ノーマル"},
	}
	p, err := NewHTTPStreamPlayer("127.0.0.1:0", newTestStreamAssets(),
		WithHistoryDir(historyDir),
		withNowFunc(func() time.Time { return fixedNow }),
		WithSpeakerLookup(lookup),
	)
	if err != nil {
		t.Fatalf("NewHTTPStreamPlayer: %v", err)
	}
	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = p.Shutdown(ctx)
	})

	if err := p.Play(context.Background(), []byte("RIFFwavdata"), app.PlayMeta{
		Text: "テストセリフ", SpeakerID: 3,
	}); err != nil {
		t.Fatalf("Play: %v", err)
	}

	filePath := filepath.Join(historyDir, "2026-04-28.jsonl")
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("expected history file at %s: %v", filePath, err)
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 line in history, got %d: %q", len(lines), data)
	}
	var rec historyRecord
	if err := json.Unmarshal([]byte(lines[0]), &rec); err != nil {
		t.Fatalf("failed to parse history line: %v", err)
	}
	if rec.Text != "テストセリフ" {
		t.Errorf("expected Text=%q, got %q", "テストセリフ", rec.Text)
	}
	if rec.SpeakerName != "ずんだもん" {
		t.Errorf("expected SpeakerName=%q, got %q", "ずんだもん", rec.SpeakerName)
	}
	if rec.StyleName != "ノーマル" {
		t.Errorf("expected StyleName=%q, got %q", "ノーマル", rec.StyleName)
	}
	if rec.Timestamp != fixedNow.UnixMilli() {
		t.Errorf("expected Timestamp=%d, got %d", fixedNow.UnixMilli(), rec.Timestamp)
	}
}

func TestHTTPStreamPlayer_History_WritesClipOnPlayText(t *testing.T) {
	historyDir := t.TempDir()
	fixedNow := time.Date(2026, 4, 28, 10, 0, 0, 0, time.Local)
	lookup := map[int]entity.SpeakerStyleInfo{
		3: {SpeakerName: "ずんだもん", StyleName: "ノーマル"},
	}
	p, err := NewHTTPStreamPlayer("127.0.0.1:0", newTestStreamAssets(),
		WithHistoryDir(historyDir),
		withNowFunc(func() time.Time { return fixedNow }),
		WithSpeakerLookup(lookup),
	)
	if err != nil {
		t.Fatalf("NewHTTPStreamPlayer: %v", err)
	}
	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = p.Shutdown(ctx)
	})
	p.silentInterval = 1 * time.Millisecond

	if err := p.PlayText(context.Background(), app.PlayMeta{
		Text: "サイレントセリフ", SpeakerID: 3,
	}); err != nil {
		t.Fatalf("PlayText: %v", err)
	}

	filePath := filepath.Join(historyDir, "2026-04-28.jsonl")
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("expected history file at %s: %v", filePath, err)
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	var rec historyRecord
	if err := json.Unmarshal([]byte(lines[0]), &rec); err != nil {
		t.Fatalf("failed to parse history line: %v", err)
	}
	if rec.Text != "サイレントセリフ" {
		t.Errorf("expected Text=%q, got %q", "サイレントセリフ", rec.Text)
	}
}

func TestHTTPStreamPlayer_History_RotatesOnDateChange(t *testing.T) {
	historyDir := t.TempDir()
	day1 := time.Date(2026, 4, 28, 23, 59, 0, 0, time.Local)
	day2 := time.Date(2026, 4, 29, 0, 1, 0, 0, time.Local)
	currentTime := day1

	p, err := NewHTTPStreamPlayer("127.0.0.1:0", newTestStreamAssets(),
		WithHistoryDir(historyDir),
		withNowFunc(func() time.Time { return currentTime }),
	)
	if err != nil {
		t.Fatalf("NewHTTPStreamPlayer: %v", err)
	}
	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = p.Shutdown(ctx)
	})

	if err := p.Play(context.Background(), []byte("RIFFwavdata"), app.PlayMeta{Text: "day1"}); err != nil {
		t.Fatalf("Play day1: %v", err)
	}
	currentTime = day2
	if err := p.Play(context.Background(), []byte("RIFFwavdata"), app.PlayMeta{Text: "day2"}); err != nil {
		t.Fatalf("Play day2: %v", err)
	}

	file1 := filepath.Join(historyDir, "2026-04-28.jsonl")
	data1, err := os.ReadFile(file1)
	if err != nil {
		t.Fatalf("expected history file at %s: %v", file1, err)
	}
	if lines := strings.Split(strings.TrimRight(string(data1), "\n"), "\n"); len(lines) != 1 {
		t.Errorf("expected 1 line in day1 file, got %d", len(lines))
	}

	file2 := filepath.Join(historyDir, "2026-04-29.jsonl")
	data2, err := os.ReadFile(file2)
	if err != nil {
		t.Fatalf("expected history file at %s: %v", file2, err)
	}
	if lines := strings.Split(strings.TrimRight(string(data2), "\n"), "\n"); len(lines) != 1 {
		t.Errorf("expected 1 line in day2 file, got %d", len(lines))
	}
}

func TestHTTPStreamPlayer_History_PrunesOldFiles(t *testing.T) {
	historyDir := t.TempDir()
	fixedNow := time.Date(2026, 4, 28, 10, 0, 0, 0, time.Local)

	oldDate := fixedNow.AddDate(0, 0, -31)
	oldFile := filepath.Join(historyDir, oldDate.Format("2006-01-02")+".jsonl")
	if err := os.WriteFile(oldFile, []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	recentDate := fixedNow.AddDate(0, 0, -29)
	recentFile := filepath.Join(historyDir, recentDate.Format("2006-01-02")+".jsonl")
	if err := os.WriteFile(recentFile, []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	p, err := NewHTTPStreamPlayer("127.0.0.1:0", newTestStreamAssets(),
		WithHistoryDir(historyDir),
		withNowFunc(func() time.Time { return fixedNow }),
	)
	if err != nil {
		t.Fatalf("NewHTTPStreamPlayer: %v", err)
	}
	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = p.Shutdown(ctx)
	})

	if _, err := os.Stat(oldFile); !os.IsNotExist(err) {
		t.Errorf("expected old file to be deleted: %s", oldFile)
	}
	if _, err := os.Stat(recentFile); err != nil {
		t.Errorf("expected recent file to still exist: %s", recentFile)
	}
}

func TestHTTPStreamPlayer_Start_NoHistoryFile(t *testing.T) {
	historyDir := t.TempDir()

	p, err := NewHTTPStreamPlayer("127.0.0.1:0", newTestStreamAssets(),
		WithHistoryDir(historyDir),
	)
	if err != nil {
		t.Fatalf("NewHTTPStreamPlayer: %v", err)
	}
	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("Start with no history file: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = p.Shutdown(ctx)
	})
}

func TestHTTPStreamPlayer_APIHistory_ReturnsTodaysEntries(t *testing.T) {
	historyDir := t.TempDir()
	fixedNow := time.Date(2026, 4, 28, 10, 0, 0, 0, time.Local)

	filePath := filepath.Join(historyDir, "2026-04-28.jsonl")
	var sb strings.Builder
	for i := 1; i <= 60; i++ {
		rec := historyRecord{
			Text:        fmt.Sprintf("text%d", i),
			SpeakerName: "ずんだもん",
			StyleName:   "ノーマル",
			Timestamp:   fixedNow.UnixMilli() + int64(i),
		}
		data, _ := json.Marshal(rec)
		sb.Write(data)
		sb.WriteByte('\n')
	}
	if err := os.WriteFile(filePath, []byte(sb.String()), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	p, err := NewHTTPStreamPlayer("127.0.0.1:0", newTestStreamAssets(),
		WithHistoryDir(historyDir),
		withNowFunc(func() time.Time { return fixedNow }),
	)
	if err != nil {
		t.Fatalf("NewHTTPStreamPlayer: %v", err)
	}
	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = p.Shutdown(ctx)
	})

	resp, err := http.Get("http://" + p.Addr() + "/api/history")
	if err != nil {
		t.Fatalf("GET /api/history: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result apiHistoryResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(result.Entries) != 50 {
		t.Errorf("expected 50 entries, got %d", len(result.Entries))
	}
	if len(result.Entries) > 0 && result.Entries[0].Text != "text11" {
		t.Errorf("expected first entry text=text11, got %q", result.Entries[0].Text)
	}
	if len(result.Entries) > 0 && result.Entries[len(result.Entries)-1].Text != "text60" {
		t.Errorf("expected last entry text=text60, got %q", result.Entries[len(result.Entries)-1].Text)
	}
}

func TestHTTPStreamPlayer_APIHistory_EmptyWhenNoFile(t *testing.T) {
	historyDir := t.TempDir()

	p, err := NewHTTPStreamPlayer("127.0.0.1:0", newTestStreamAssets(),
		WithHistoryDir(historyDir),
	)
	if err != nil {
		t.Fatalf("NewHTTPStreamPlayer: %v", err)
	}
	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = p.Shutdown(ctx)
	})

	resp, err := http.Get("http://" + p.Addr() + "/api/history")
	if err != nil {
		t.Fatalf("GET /api/history: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result apiHistoryResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(result.Entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(result.Entries))
	}
}

// TestHTTPStreamPlayer_APIHistory_ReturnsUpdatedHistory は Start() 後にファイルへ追記された
// 履歴エントリが /api/history リクエスト時に反映されることを検証する。
func TestHTTPStreamPlayer_APIHistory_ReturnsUpdatedHistory(t *testing.T) {
	historyDir := t.TempDir()
	fixedNow := time.Date(2026, 4, 28, 10, 0, 0, 0, time.Local)

	p, err := NewHTTPStreamPlayer("127.0.0.1:0", newTestStreamAssets(),
		WithHistoryDir(historyDir),
		withNowFunc(func() time.Time { return fixedNow }),
	)
	if err != nil {
		t.Fatalf("NewHTTPStreamPlayer: %v", err)
	}
	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = p.Shutdown(ctx)
	})

	// Start() 後に履歴ファイルへエントリを追記する。
	filePath := filepath.Join(historyDir, "2026-04-28.jsonl")
	rec := historyRecord{
		Text:        "Start後に追記されたテキスト",
		SpeakerName: "ずんだもん",
		StyleName:   "ノーマル",
		Timestamp:   fixedNow.UnixMilli(),
	}
	data, _ := json.Marshal(rec)
	data = append(data, '\n')
	if err := os.WriteFile(filePath, data, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	resp, err := http.Get("http://" + p.Addr() + "/api/history")
	if err != nil {
		t.Fatalf("GET /api/history: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result apiHistoryResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(result.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(result.Entries))
	}
	if result.Entries[0].Text != "Start後に追記されたテキスト" {
		t.Errorf("expected text %q, got %q", "Start後に追記されたテキスト", result.Entries[0].Text)
	}
}

// TestHTTPStreamPlayer_APIHistory_CrossDayReturnsNewFile は日付を跨いだ後に
// /api/history が新しい日付のファイルを読み込むことを検証する。
func TestHTTPStreamPlayer_APIHistory_CrossDayReturnsNewFile(t *testing.T) {
	historyDir := t.TempDir()
	day1 := time.Date(2026, 4, 28, 23, 59, 0, 0, time.Local)
	day2 := time.Date(2026, 4, 29, 0, 1, 0, 0, time.Local)
	nowPtr := &day1

	p, err := NewHTTPStreamPlayer("127.0.0.1:0", newTestStreamAssets(),
		WithHistoryDir(historyDir),
		withNowFunc(func() time.Time { return *nowPtr }),
	)
	if err != nil {
		t.Fatalf("NewHTTPStreamPlayer: %v", err)
	}
	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = p.Shutdown(ctx)
	})

	// Day1 のファイルを作成。
	file1 := filepath.Join(historyDir, "2026-04-28.jsonl")
	rec1 := historyRecord{Text: "day1エントリ", SpeakerName: "ずんだもん", StyleName: "ノーマル", Timestamp: day1.UnixMilli()}
	data1, _ := json.Marshal(rec1)
	data1 = append(data1, '\n')
	if err := os.WriteFile(file1, data1, 0o644); err != nil {
		t.Fatalf("WriteFile day1: %v", err)
	}

	// Day2 へ日付変更し、Day2 のファイルを作成。
	*nowPtr = day2
	file2 := filepath.Join(historyDir, "2026-04-29.jsonl")
	rec2 := historyRecord{Text: "day2エントリ", SpeakerName: "ずんだもん", StyleName: "ノーマル", Timestamp: day2.UnixMilli()}
	data2, _ := json.Marshal(rec2)
	data2 = append(data2, '\n')
	if err := os.WriteFile(file2, data2, 0o644); err != nil {
		t.Fatalf("WriteFile day2: %v", err)
	}

	resp, err := http.Get("http://" + p.Addr() + "/api/history")
	if err != nil {
		t.Fatalf("GET /api/history: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result apiHistoryResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(result.Entries) != 1 {
		t.Fatalf("expected 1 entry from day2, got %d", len(result.Entries))
	}
	if result.Entries[0].Text != "day2エントリ" {
		t.Errorf("expected day2 entry, got %q", result.Entries[0].Text)
	}
}

// --- /api/play tests (#418) ---

func TestHTTPStreamPlayer_APIPlay_MethodNotAllowed(t *testing.T) {
	t.Parallel()
	p := newStartedPlayer(t)
	resp, err := http.Get("http://" + p.Addr() + "/api/play")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", resp.StatusCode)
	}
}

func TestHTTPStreamPlayer_APIPlay_NoClient(t *testing.T) {
	t.Parallel()
	p := newStartedPlayer(t)
	body := bytes.NewBufferString(`{"clips":[{"text":"hello","speaker_id":2}]}`)
	resp, err := http.Post("http://"+p.Addr()+"/api/play", "application/json", body)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", resp.StatusCode)
	}
}

func TestHTTPStreamPlayer_APIPlay_InvalidJSON(t *testing.T) {
	t.Parallel()
	stub := &testVoicevoxClient{wav: []byte("RIFFx")}
	p := newStartedPlayerWithOpts(t, WithVoicevoxClient(stub))
	body := bytes.NewBufferString(`not-json`)
	resp, err := http.Post("http://"+p.Addr()+"/api/play", "application/json", body)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestHTTPStreamPlayer_APIPlay_Success(t *testing.T) {
	t.Parallel()
	stub := &testVoicevoxClient{wav: []byte("RIFFx")}
	p := newStartedPlayerWithOpts(t, WithVoicevoxClient(stub))

	events := subscribeSSE(t, "http://"+p.Addr())
	time.Sleep(100 * time.Millisecond)

	body := bytes.NewBufferString(`{"clips":[{"text":"こんにちは","speaker_id":2}]}`)
	resp, err := http.Post("http://"+p.Addr()+"/api/play", "application/json", body)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	var result ViewerPlayResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result.Silent {
		t.Errorf("expected silent=false, got true")
	}
	if result.PlaybackID == "" {
		t.Errorf("expected non-empty playback_id")
	}

	// ワーカーが処理するのを待ってから stub の状態を確認
	select {
	case <-events:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for worker to process clip")
	}

	if stub.capturedText != "こんにちは" {
		t.Errorf("expected capturedText=%q, got %q", "こんにちは", stub.capturedText)
	}
	if stub.capturedSpeaker != 2 {
		t.Errorf("expected capturedSpeaker=2, got %d", stub.capturedSpeaker)
	}
}

func TestHTTPStreamPlayer_APIPlay_SilentMode(t *testing.T) {
	t.Parallel()
	stub := &testVoicevoxClient{wav: []byte("RIFFx")}
	p := newStartedPlayerWithOpts(t, WithVoicevoxClient(stub))
	p.SetSilent("VOICEVOX接続失敗")
	body := bytes.NewBufferString(`{"clips":[{"text":"こんにちは","speaker_id":2}]}`)
	resp, err := http.Post("http://"+p.Addr()+"/api/play", "application/json", body)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	var result ViewerPlayResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !result.Silent {
		t.Errorf("expected silent=true")
	}
	if result.SilentReason == "" {
		t.Errorf("expected non-empty silent_reason")
	}
	if stub.createQueryCall != 0 {
		t.Errorf("expected no synthesis in silent mode, got createQueryCall=%d", stub.createQueryCall)
	}
}

func TestHTTPStreamPlayer_APIPlay_SynthesisFails(t *testing.T) {
	t.Parallel()
	stub := &testVoicevoxClient{createQueryErr: errors.New("synthesis error")}
	p := newStartedPlayerWithOpts(t, WithVoicevoxClient(stub))
	body := bytes.NewBufferString(`{"clips":[{"text":"hello","speaker_id":2}]}`)
	resp, err := http.Post("http://"+p.Addr()+"/api/play", "application/json", body)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	// 非同期 worker 化により、リクエストは即座に 200 で返る
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 (async), got %d", resp.StatusCode)
	}
}

// --- #487 /api/play 複数クリップ対応＋非同期 worker ---
//
// DONE: clips 形式で UUID v4 playback_id が即時返る               → TestHTTPStreamPlayer_APIPlay_ClipsFormatReturnsPlaybackID
// DONE: 旧形式 (text/speaker_id) で 400 Bad Request               → TestHTTPStreamPlayer_APIPlay_OldFormatRejected
// DONE: clips 空配列で 400 Bad Request                            → TestHTTPStreamPlayer_APIPlay_EmptyClipsRejected
// DONE: キュー満杯（64 pending）で 503 Service Unavailable        → TestHTTPStreamPlayer_APIPlay_QueueFull
// DONE: worker が clips を順次 SSE broadcast する                  → TestHTTPStreamPlayer_Worker_BroadcastsClipsInOrder
// DONE: 1 件目 synthesize 失敗で残りクリップが broadcast されない → TestHTTPStreamPlayer_Worker_SynthesizeFailureStopsProcessing
// DONE: silent モード時は SSE broadcast されず即 completed 状態   → TestHTTPStreamPlayer_APIPlay_SilentModeSkipsBroadcast

func TestHTTPStreamPlayer_APIPlay_ClipsFormatReturnsPlaybackID(t *testing.T) {
	t.Parallel()
	stub := &testVoicevoxClient{wav: []byte("RIFFx")}
	p := newStartedPlayerWithOpts(t, WithVoicevoxClient(stub))
	body := bytes.NewBufferString(`{"clips":[{"text":"こんにちは","speaker_id":2}]}`)
	resp, err := http.Post("http://"+p.Addr()+"/api/play", "application/json", body)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var result ViewerPlayResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result.PlaybackID == "" {
		t.Error("expected non-empty playback_id")
	}
	uuidPattern := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	if !uuidPattern.MatchString(result.PlaybackID) {
		t.Errorf("expected UUID v4 format, got: %s", result.PlaybackID)
	}
	if result.ClipCount != 1 {
		t.Errorf("expected clip_count=1, got %d", result.ClipCount)
	}
	if result.Silent {
		t.Error("expected silent=false")
	}
}

func TestHTTPStreamPlayer_APIPlay_OldFormatRejected(t *testing.T) {
	t.Parallel()
	stub := &testVoicevoxClient{wav: []byte("RIFFx")}
	p := newStartedPlayerWithOpts(t, WithVoicevoxClient(stub))
	body := bytes.NewBufferString(`{"text":"こんにちは","speaker_id":2}`)
	resp, err := http.Post("http://"+p.Addr()+"/api/play", "application/json", body)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for old format, got %d", resp.StatusCode)
	}
}

func TestHTTPStreamPlayer_APIPlay_EmptyClipsRejected(t *testing.T) {
	t.Parallel()
	stub := &testVoicevoxClient{wav: []byte("RIFFx")}
	p := newStartedPlayerWithOpts(t, WithVoicevoxClient(stub))
	body := bytes.NewBufferString(`{"clips":[]}`)
	resp, err := http.Post("http://"+p.Addr()+"/api/play", "application/json", body)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for empty clips, got %d", resp.StatusCode)
	}
}

func TestHTTPStreamPlayer_APIPlay_QueueFull(t *testing.T) {
	t.Parallel()
	block := make(chan struct{})
	t.Cleanup(func() { close(block) })
	stub := &testVoicevoxClient{createQueryBlock: block}
	p := newStartedPlayerWithOpts(t, WithVoicevoxClient(stub))
	url := "http://" + p.Addr() + "/api/play"

	// 1件目を送ってワーカーに取らせてからキューを埋める
	body1 := bytes.NewBufferString(`{"clips":[{"text":"x","speaker_id":2}]}`)
	resp1, err := http.Post(url, "application/json", body1)
	if err != nil {
		t.Fatalf("POST 1: %v", err)
	}
	_ = resp1.Body.Close()
	time.Sleep(50 * time.Millisecond) // ワーカーが取り出すのを待つ

	for i := 0; i < batchQueueCapacity; i++ {
		body := bytes.NewBufferString(`{"clips":[{"text":"x","speaker_id":2}]}`)
		resp, err := http.Post(url, "application/json", body)
		if err != nil {
			t.Fatalf("POST %d: %v", i+2, err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 filling queue (batch %d), got %d", i+2, resp.StatusCode)
		}
		_ = resp.Body.Close()
	}

	// キュー満杯で 503 になるはず
	bodyFull := bytes.NewBufferString(`{"clips":[{"text":"x","speaker_id":2}]}`)
	respFull, err := http.Post(url, "application/json", bodyFull)
	if err != nil {
		t.Fatalf("POST full: %v", err)
	}
	defer func() { _ = respFull.Body.Close() }()
	if respFull.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("expected 503 when queue is full, got %d", respFull.StatusCode)
	}
}

func TestHTTPStreamPlayer_Worker_BroadcastsClipsInOrder(t *testing.T) {
	t.Parallel()
	stub := &testVoicevoxClient{wav: []byte("RIFFx")}
	p := newStartedPlayerWithOpts(t, WithVoicevoxClient(stub))

	events := subscribeSSE(t, "http://"+p.Addr())
	time.Sleep(100 * time.Millisecond)

	body := bytes.NewBufferString(`{"clips":[{"text":"first","speaker_id":2},{"text":"second","speaker_id":3}]}`)
	resp, err := http.Post("http://"+p.Addr()+"/api/play", "application/json", body)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var receivedTexts []string
	timeout := time.After(3 * time.Second)
	for len(receivedTexts) < 2 {
		select {
		case data := <-events:
			var ev clipEvent
			if err := json.Unmarshal([]byte(data), &ev); err == nil {
				receivedTexts = append(receivedTexts, ev.Text)
			}
		case <-timeout:
			t.Fatalf("timeout waiting for clip events, received: %v", receivedTexts)
		}
	}

	if len(receivedTexts) < 2 || receivedTexts[0] != "first" || receivedTexts[1] != "second" {
		t.Errorf("expected clips in order [first, second], got %v", receivedTexts)
	}
}

func TestHTTPStreamPlayer_Worker_SynthesizeFailureStopsProcessing(t *testing.T) {
	t.Parallel()
	stub := &testVoicevoxClient{createQueryErr: errors.New("synth failed")}
	p := newStartedPlayerWithOpts(t, WithVoicevoxClient(stub))

	events := subscribeSSE(t, "http://"+p.Addr())
	time.Sleep(100 * time.Millisecond)

	body := bytes.NewBufferString(`{"clips":[{"text":"first","speaker_id":2},{"text":"second","speaker_id":3}]}`)
	resp, err := http.Post("http://"+p.Addr()+"/api/play", "application/json", body)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result ViewerPlayResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	// ワーカーが処理を終えるのを少し待つ
	time.Sleep(200 * time.Millisecond)

	// SSE clip イベントは受信されていないはず
	select {
	case data := <-events:
		t.Errorf("expected no clip events after synthesis failure, got: %s", data)
	default:
		// OK
	}

	// playback state は failed のはず
	status, ok := p.PlaybackStatus(result.PlaybackID)
	if !ok {
		t.Fatalf("playback state not found for id=%s", result.PlaybackID)
	}
	if status != "failed" {
		t.Errorf("expected playback status=failed, got %s", status)
	}
}

func TestHTTPStreamPlayer_APIPlay_SilentModeSkipsBroadcast(t *testing.T) {
	t.Parallel()
	stub := &testVoicevoxClient{wav: []byte("RIFFx")}
	p := newStartedPlayerWithOpts(t, WithVoicevoxClient(stub))
	p.SetSilent("VOICEVOX接続失敗")

	events := subscribeSSE(t, "http://"+p.Addr())
	time.Sleep(100 * time.Millisecond)

	body := bytes.NewBufferString(`{"clips":[{"text":"hello","speaker_id":2}]}`)
	resp, err := http.Post("http://"+p.Addr()+"/api/play", "application/json", body)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result ViewerPlayResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !result.Silent {
		t.Error("expected silent=true")
	}
	if result.SilentReason == "" {
		t.Error("expected non-empty silent_reason")
	}

	// ワーカーが何も処理しないことを確認
	time.Sleep(100 * time.Millisecond)
	select {
	case data := <-events:
		t.Errorf("expected no clip events in silent mode, got: %s", data)
	default:
		// OK
	}

	// playback state は completed のはず
	status, ok := p.PlaybackStatus(result.PlaybackID)
	if !ok {
		t.Fatalf("playback state not found for id=%s", result.PlaybackID)
	}
	if status != "completed" {
		t.Errorf("expected playback status=completed, got %s", status)
	}
}
