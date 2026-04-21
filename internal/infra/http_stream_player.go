package infra

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/canpok1/vox-actor/internal/app"
)

//go:embed stream_assets
var streamAssets embed.FS

// HTTPStreamPlayer はWAVをHTTP/SSEでブラウザに配信する AudioPlayer 実装。
type HTTPStreamPlayer struct {
	addr   string
	logger *slog.Logger

	server   *http.Server
	listener net.Listener

	// アセットは Start 時に embed.FS から一度だけ読み込む。
	indexHTML []byte
	appJS     []byte
	appCSS    []byte

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
)

// clipEvent は SSE で配信する clip イベントのペイロード。
type clipEvent struct {
	ID   uint64 `json:"id"`
	URL  string `json:"url"`
	Text string `json:"text"`
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

// NewHTTPStreamPlayer は新しい HTTPStreamPlayer を生成する。
// addr はバインドアドレス（例: "127.0.0.1:8080"）。":0" で動的ポート割当。
func NewHTTPStreamPlayer(addr string, opts ...HTTPStreamOption) (*HTTPStreamPlayer, error) {
	if addr == "" {
		return nil, fmt.Errorf("stream addr must not be empty")
	}
	p := &HTTPStreamPlayer{
		addr:        addr,
		logger:      slog.New(slog.DiscardHandler),
		subscribers: newSubscriberRegistry(),
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

	if err := p.loadAssets(); err != nil {
		return err
	}

	lis, err := net.Listen("tcp", p.addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", p.addr, err)
	}
	p.listener = lis

	mux := http.NewServeMux()
	mux.HandleFunc("/", p.handleRoot)
	mux.HandleFunc("/app.js", p.handleStaticAsset(p.appJS, "application/javascript; charset=utf-8"))
	mux.HandleFunc("/app.css", p.handleStaticAsset(p.appCSS, "text/css; charset=utf-8"))
	mux.HandleFunc("/events", p.handleEvents)
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

func (p *HTTPStreamPlayer) loadAssets() error {
	for _, entry := range []struct {
		name string
		dst  *[]byte
	}{
		{"stream_assets/index.html", &p.indexHTML},
		{"stream_assets/app.js", &p.appJS},
		{"stream_assets/app.css", &p.appCSS},
	} {
		data, err := fs.ReadFile(streamAssets, entry.name)
		if err != nil {
			return fmt.Errorf("failed to read embedded asset %s: %w", entry.name, err)
		}
		*entry.dst = data
	}
	return nil
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

	payload, err := json.Marshal(clipEvent{
		ID:   id,
		URL:  clipPathPrefix + strconv.FormatUint(id, 10) + clipPathSuffix,
		Text: meta.Text,
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

func (p *HTTPStreamPlayer) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(p.indexHTML)
}

func (p *HTTPStreamPlayer) handleStaticAsset(data []byte, contentType string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", contentType)
		_, _ = w.Write(data)
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
	w.Header().Set("Content-Type", "audio/wav")
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	_, _ = w.Write(data)
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
