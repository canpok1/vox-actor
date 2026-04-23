package infra

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/canpok1/vox-actor/internal/app"
	"github.com/canpok1/vox-actor/internal/domain/entity"
)

// HTTPStreamPlayer はWAVをHTTP/SSEでブラウザに配信する AudioPlayer 実装。
type HTTPStreamPlayer struct {
	addr   string
	logger *slog.Logger

	server   *http.Server
	listener net.Listener

	// staticFS は配信画面用の静的ファイル FS（Vite の dist 出力を想定）。
	staticFS fs.FS
	// speakersJSON は Start 時に speakerLookup から一度だけマーシャルしたレスポンス。
	speakersJSON []byte

	// speakerLookup は SpeakerID → 話者名/スタイル名 の解決マップ。
	// nil または未ヒットの場合は `話者#<ID>` / 空文字 にフォールバックする。
	speakerLookup map[int]entity.SpeakerStyleInfo

	// nowFunc は clipEvent の timestamp 生成に使う時刻取得関数（テスト差し替え用）。
	nowFunc func() time.Time

	// voicevoxClient はテスト再生 (/test-clip) で合成に使うクライアント。
	// 未注入時は /test-clip が 503 を返す。
	voicevoxClient app.VoicevoxClient
	// testPhrase はテスト再生で合成するフレーズ。既定は defaultTestPhrase。
	testPhrase string

	// testClipCache は SpeakerID → WAV のキャッシュ。初回合成時に書き込まれ Shutdown まで保持される。
	testClipCacheMu sync.Mutex
	testClipCache   map[int][]byte

	mu          sync.Mutex
	started     bool
	shutdown    bool
	nextClipID  atomic.Uint64
	clips       *clipRingBuffer
	subscribers *subscriberRegistry
}

var _ app.StreamPlayer = (*HTTPStreamPlayer)(nil)

const (
	// streamHistorySize はサーバー側 WAV リングバッファの容量（固定値）。
	// バースト耐性を持たせた保守的な値。
	streamHistorySize   = 20
	clipPathPrefix      = "/clips/"
	clipPathSuffix      = ".wav"
	sseEventClip        = "clip"
	sseSubscriberBuffer = 16
	// defaultTestPhrase は /test-clip で合成するデフォルトフレーズ。
	defaultTestPhrase = "音量テストです"
)

// clipEvent は SSE で配信する clip イベントのペイロード。
type clipEvent struct {
	ID          uint64 `json:"id"`
	URL         string `json:"url"`
	Text        string `json:"text"`
	SpeakerName string `json:"speakerName"`
	StyleName   string `json:"styleName"`
	// Timestamp は配信時刻の Unix ms（UTC）。ブラウザ側で HH:MM:SS に整形する。
	Timestamp int64 `json:"timestamp"`
}

// HTTPStreamOption は HTTPStreamPlayer の生成時に指定するオプション。
type HTTPStreamOption func(*HTTPStreamPlayer)

// WithHTTPStreamLogger はロガーを設定するオプション。
func WithHTTPStreamLogger(logger *slog.Logger) HTTPStreamOption {
	return func(p *HTTPStreamPlayer) {
		if logger != nil {
			p.logger = logger
		}
	}
}

// WithSpeakerLookup は clipEvent の speakerName/styleName 解決に使うマップを設定するオプション。
func WithSpeakerLookup(lookup map[int]entity.SpeakerStyleInfo) HTTPStreamOption {
	return func(p *HTTPStreamPlayer) {
		if lookup != nil {
			p.speakerLookup = lookup
		}
	}
}

// WithVoicevoxClient は /test-clip エンドポイントで使う音声合成クライアントを設定するオプション。
// 未注入時は /test-clip が 503 を返す。
func WithVoicevoxClient(client app.VoicevoxClient) HTTPStreamOption {
	return func(p *HTTPStreamPlayer) {
		if client != nil {
			p.voicevoxClient = client
		}
	}
}

// WithTestPhrase は /test-clip でテスト合成に使うフレーズを設定するオプション。
// 既定は defaultTestPhrase。
func WithTestPhrase(phrase string) HTTPStreamOption {
	return func(p *HTTPStreamPlayer) {
		if phrase != "" {
			p.testPhrase = phrase
		}
	}
}

// withNowFunc は clipEvent の timestamp 取得に使う関数を設定する（テスト用）。
func withNowFunc(now func() time.Time) HTTPStreamOption {
	return func(p *HTTPStreamPlayer) {
		if now != nil {
			p.nowFunc = now
		}
	}
}

// NewHTTPStreamPlayer は新しい HTTPStreamPlayer を生成する。
// addr はバインドアドレス（例: "127.0.0.1:8080"）。":0" で動的ポート割当。
// staticFS には Vite でビルドした配信画面の dist ディレクトリ相当の FS を渡す
// （ルートに index.html、assets/ に <hash>.js / <hash>.css が配置されている想定）。
func NewHTTPStreamPlayer(addr string, staticFS fs.FS, opts ...HTTPStreamOption) (*HTTPStreamPlayer, error) {
	if addr == "" {
		return nil, fmt.Errorf("stream addr must not be empty")
	}
	if staticFS == nil {
		return nil, fmt.Errorf("stream staticFS must not be nil")
	}
	p := &HTTPStreamPlayer{
		addr:          addr,
		logger:        slog.New(slog.DiscardHandler),
		staticFS:      staticFS,
		subscribers:   newSubscriberRegistry(),
		nowFunc:       time.Now,
		testPhrase:    defaultTestPhrase,
		testClipCache: make(map[int][]byte),
	}
	for _, opt := range opts {
		opt(p)
	}
	p.clips = newClipRingBuffer(streamHistorySize)
	return p, nil
}

// Start はサーバーを起動する。ポートのバインドまで完了してから return する。
func (p *HTTPStreamPlayer) Start(_ context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.started {
		return fmt.Errorf("HTTPStreamPlayer already started")
	}

	if err := p.buildSpeakersJSON(); err != nil {
		return err
	}

	lis, err := net.Listen("tcp", p.addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", p.addr, err)
	}
	p.listener = lis

	fileServer := http.FileServer(http.FS(p.staticFS))

	mux := http.NewServeMux()
	mux.Handle("/", withStaticCacheControl(fileServer))
	mux.HandleFunc("/events", p.handleEvents)
	mux.HandleFunc("/speakers.json", p.handleSpeakers)
	mux.HandleFunc("/test-clip", p.handleTestClip)
	mux.HandleFunc(clipPathPrefix, p.handleClip)

	p.server = &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	p.started = true

	go func() {
		err := p.server.Serve(lis)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			p.logger.Error("stream server error", "error", err)
		}
	}()

	p.logger.Info("stream server started", "addr", lis.Addr().String())
	return nil
}

// withStaticCacheControl は /assets/ 配下のハッシュ付きアセットに強力なキャッシュを、
// それ以外（index.html 等）には no-cache を付与してから次のハンドラに渡す。
// これにより index.html は常に最新を取得し、そこから参照されるハッシュ付きアセットは
// ハッシュが変わらない限り長期キャッシュで配信される（キャッシュバスティング）。
func withStaticCacheControl(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/assets/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			w.Header().Set("Cache-Control", "no-cache")
		}
		next.ServeHTTP(w, r)
	})
}

// Shutdown はサーバーを停止する。
func (p *HTTPStreamPlayer) Shutdown(ctx context.Context) error {
	p.mu.Lock()
	if !p.started || p.shutdown {
		p.mu.Unlock()
		return nil
	}
	p.shutdown = true
	server := p.server
	p.mu.Unlock()

	p.subscribers.closeAll()
	if server == nil {
		return nil
	}
	return server.Shutdown(ctx)
}

// resolveSpeaker は SpeakerID から (話者名, スタイル名) を解決する。
// lookup に存在しない場合は SpeakerName を `話者#<ID>`、StyleName を空文字にフォールバックする。
func (p *HTTPStreamPlayer) resolveSpeaker(id int) (string, string) {
	if info, ok := p.speakerLookup[id]; ok {
		return info.SpeakerName, info.StyleName
	}
	return fmt.Sprintf("話者#%d", id), ""
}

// Addr はサーバーがリッスン中のアドレスを返す。Start 前は空文字を返す。
func (p *HTTPStreamPlayer) Addr() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.listener == nil {
		return ""
	}
	return p.listener.Addr().String()
}

// Play はWAVバイト列をキューに積み、SSE購読者にブロードキャストする。
// 配信後はWAV推定再生時間ぶん同期的にブロックして、合成パイプラインに対する
// 自然な backpressure として働く（ローカル再生モードと挙動を揃えるため）。
// meta.Text はブラウザのタイムライン表示に利用される。
func (p *HTTPStreamPlayer) Play(ctx context.Context, wavData []byte, meta app.PlayMeta) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if len(wavData) == 0 {
		return fmt.Errorf("WAV data is empty")
	}

	p.mu.Lock()
	started := p.started
	p.mu.Unlock()
	if !started {
		return fmt.Errorf("HTTPStreamPlayer is not started")
	}

	id := p.nextClipID.Add(1)
	// 呼び出し元のバッファ再利用に備えてコピーを保持する。
	buf := make([]byte, len(wavData))
	copy(buf, wavData)
	p.clips.put(id, buf)

	speakerName, styleName := p.resolveSpeaker(meta.SpeakerID)
	payload, err := json.Marshal(clipEvent{
		ID:          id,
		URL:         clipPathPrefix + strconv.FormatUint(id, 10) + clipPathSuffix,
		Text:        meta.Text,
		SpeakerName: speakerName,
		StyleName:   styleName,
		Timestamp:   p.nowFunc().UnixMilli(),
	})
	if err != nil {
		return fmt.Errorf("failed to marshal clip payload: %w", err)
	}

	n, dropped := p.subscribers.broadcast(sseEventClip, payload)
	if dropped > 0 {
		p.logger.Warn("stream clip dropped for slow subscribers", "clipId", id, "dropped", dropped)
	}
	p.logger.Info("stream clip delivered", "clipId", id, "subscribers", n)

	duration, err := estimateWAVDuration(wavData)
	if err != nil {
		p.logger.Warn("failed to estimate WAV duration, skipping playback backpressure", "clipId", id, "error", err)
		return nil
	}
	if duration <= 0 {
		return nil
	}
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *HTTPStreamPlayer) handleClip(w http.ResponseWriter, r *http.Request) {
	id, ok := parseClipID(r.URL.Path)
	if !ok {
		http.NotFound(w, r)
		return
	}
	data, ok := p.clips.get(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	writeWAV(w, data)
}

func parseClipID(path string) (uint64, bool) {
	if !strings.HasPrefix(path, clipPathPrefix) || !strings.HasSuffix(path, clipPathSuffix) {
		return 0, false
	}
	raw := strings.TrimSuffix(strings.TrimPrefix(path, clipPathPrefix), clipPathSuffix)
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0, false
	}
	return id, true
}

// speakerJSON は /speakers.json のレスポンス1要素分のペイロード。
type speakerJSON struct {
	ID          int    `json:"id"`
	SpeakerName string `json:"speakerName"`
	StyleName   string `json:"styleName"`
}

// buildSpeakersJSON は speakerLookup を id 昇順で配列化し Start 時に一度だけマーシャルする。
func (p *HTTPStreamPlayer) buildSpeakersJSON() error {
	ids := make([]int, 0, len(p.speakerLookup))
	for id := range p.speakerLookup {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	items := make([]speakerJSON, 0, len(ids))
	for _, id := range ids {
		info := p.speakerLookup[id]
		items = append(items, speakerJSON{
			ID:          id,
			SpeakerName: info.SpeakerName,
			StyleName:   info.StyleName,
		})
	}
	payload, err := json.Marshal(items)
	if err != nil {
		return fmt.Errorf("failed to marshal speakers.json: %w", err)
	}
	p.speakersJSON = payload
	return nil
}

func (p *HTTPStreamPlayer) handleSpeakers(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = w.Write(p.speakersJSON)
}

func (p *HTTPStreamPlayer) handleTestClip(w http.ResponseWriter, r *http.Request) {
	if p.voicevoxClient == nil {
		http.Error(w, "voicevox client is not configured", http.StatusServiceUnavailable)
		return
	}
	speakerStr := r.URL.Query().Get("speaker")
	if speakerStr == "" {
		http.Error(w, "speaker parameter is required", http.StatusBadRequest)
		return
	}
	speakerID, err := strconv.Atoi(speakerStr)
	if err != nil {
		http.Error(w, "speaker parameter must be an integer", http.StatusBadRequest)
		return
	}
	if _, ok := p.speakerLookup[speakerID]; !ok {
		http.Error(w, "speaker not found", http.StatusBadRequest)
		return
	}

	p.testClipCacheMu.Lock()
	cached, ok := p.testClipCache[speakerID]
	p.testClipCacheMu.Unlock()
	if ok {
		writeWAV(w, cached)
		return
	}

	ctx := r.Context()
	query, err := p.voicevoxClient.CreateQuery(ctx, p.testPhrase, speakerID)
	if err != nil {
		p.logger.Error("test-clip CreateQuery failed", "speakerID", speakerID, "error", err)
		http.Error(w, "failed to create audio query", http.StatusBadGateway)
		return
	}
	wav, err := p.voicevoxClient.Synthesize(ctx, query, speakerID)
	if err != nil {
		p.logger.Error("test-clip Synthesize failed", "speakerID", speakerID, "error", err)
		http.Error(w, "failed to synthesize", http.StatusBadGateway)
		return
	}

	p.testClipCacheMu.Lock()
	p.testClipCache[speakerID] = wav
	p.testClipCacheMu.Unlock()

	writeWAV(w, wav)
}

func writeWAV(w http.ResponseWriter, data []byte) {
	w.Header().Set("Content-Type", "audio/wav")
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	_, _ = w.Write(data)
}

func (p *HTTPStreamPlayer) handleEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	sub := p.subscribers.add()
	defer p.subscribers.remove(sub)

	for {
		select {
		case <-r.Context().Done():
			return
		case msg, ok := <-sub.ch:
			if !ok {
				return
			}
			if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", msg.event, msg.data); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

// --- ring buffer ---

type clipEntry struct {
	id   uint64
	data []byte
}

type clipRingBuffer struct {
	mu      sync.Mutex
	cap     int
	entries []clipEntry
}

func newClipRingBuffer(capacity int) *clipRingBuffer {
	return &clipRingBuffer{cap: capacity, entries: make([]clipEntry, 0, capacity)}
}

func (b *clipRingBuffer) put(id uint64, data []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.entries) >= b.cap {
		// 旧エントリのバイト列を GC 対象にするため参照をクリアしてから前詰めする。
		b.entries[0] = clipEntry{}
		b.entries = b.entries[1:]
	}
	b.entries = append(b.entries, clipEntry{id: id, data: data})
}

func (b *clipRingBuffer) get(id uint64) ([]byte, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, e := range b.entries {
		if e.id == id {
			return e.data, true
		}
	}
	return nil, false
}

// --- subscriber registry ---

type sseMessage struct {
	event string
	data  []byte
}

type subscriber struct {
	ch chan sseMessage
}

type subscriberRegistry struct {
	mu     sync.Mutex
	subs   map[*subscriber]struct{}
	closed bool
}

func newSubscriberRegistry() *subscriberRegistry {
	return &subscriberRegistry{subs: make(map[*subscriber]struct{})}
}

func (r *subscriberRegistry) add() *subscriber {
	r.mu.Lock()
	defer r.mu.Unlock()
	s := &subscriber{ch: make(chan sseMessage, sseSubscriberBuffer)}
	if r.closed {
		close(s.ch)
		return s
	}
	r.subs[s] = struct{}{}
	return s
}

func (r *subscriberRegistry) remove(s *subscriber) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.subs[s]; !ok {
		return
	}
	delete(r.subs, s)
	close(s.ch)
}

// broadcast は配信対象全員にメッセージを送る。
// 戻り値は (配信成功数, バッファ溢れで破棄したクライアント数)。
func (r *subscriberRegistry) broadcast(event string, data []byte) (int, int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	msg := sseMessage{event: event, data: data}
	delivered, dropped := 0, 0
	for s := range r.subs {
		select {
		case s.ch <- msg:
			delivered++
		default:
			dropped++
		}
	}
	return delivered, dropped
}

func (r *subscriberRegistry) closeAll() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return
	}
	r.closed = true
	for s := range r.subs {
		close(s.ch)
		delete(r.subs, s)
	}
}
