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
	"os"
	"path/filepath"
	"regexp"
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
	// apiStatusJSON は Start 時に speakerLookup と silent 状態から一度だけマーシャルしたレスポンス。
	apiStatusJSON []byte

	// workspacePath はキャラクター設定ファイルの読み込み先ワークスペースルート。
	// 空文字の場合はキャラクター機能は無効。WithWorkspacePath で設定される。
	workspacePath string
	// assetsDirs はキャラクター資産ディレクトリのリスト（優先順: index 0 が最高優先）。
	// 設定されている場合、workspacePath の代わりにこちらが使われる。
	assetsDirs []string
	// apiCharactersJSON は Start 時に characters 設定から一度だけマーシャルしたレスポンス。
	apiCharactersJSON []byte

	// speakerLookup は SpeakerID → 話者名/スタイル名 の解決マップ。
	// nil または未ヒットの場合は `話者#<ID>` / 空文字 にフォールバックする。
	speakerLookup map[int]entity.SpeakerStyleInfo

	// orderedSpeakerIDs は /api/status で返す speakers 配列の順序。
	// 設定されていない場合は speakerLookup のキーを ID 昇順でソートして使用する（後方互換）。
	orderedSpeakerIDs []int

	// silent が true の場合、Play() / PlayText() は WAV を配信せずテキストのみを配信する。
	// silentReason は /api/status でフロントに伝える文面（改行を含んでよい）。
	silent       bool
	silentReason string

	// nowFunc は clipEvent の timestamp 生成に使う時刻取得関数（テスト差し替え用）。
	nowFunc func() time.Time

	// historyDir は再生履歴 JSONL ファイルの保存ディレクトリ。空文字の場合は履歴機能無効。
	historyDir string

	// silentInterval は PlayText で使う固定待機時間（backpressure の暫定値）。
	silentInterval time.Duration

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
	nextErrorID atomic.Uint64
	clips       *clipRingBuffer
	subscribers *subscriberRegistry
}

var (
	_ app.StreamPlayer     = (*HTTPStreamPlayer)(nil)
	_ app.ErrorBroadcaster = (*HTTPStreamPlayer)(nil)
)

const (
	// streamHistorySize はサーバー側 WAV リングバッファの容量（固定値）。
	// バースト耐性を持たせた保守的な値。
	streamHistorySize   = 20
	clipPathPrefix      = "/clips/"
	clipPathSuffix      = ".wav"
	sseEventClip        = "clip"
	sseEventError       = "error"
	sseSubscriberBuffer = 16
	// defaultTestPhrase は /test-clip で合成するデフォルトフレーズ。
	defaultTestPhrase = "音量テストです"
	// defaultSilentInterval は無音モードで PlayText が固定的に待機する時間。
	// タイムラインが一瞬で流れ去らないようにするための暫定値。
	defaultSilentInterval = 500 * time.Millisecond
	// workspaceAssetsDir は workspacePath 配下のアセットディレクトリ名。
	workspaceAssetsDir = "assets"
	// historyLoadSize は起動時に読み込む履歴の最大件数。
	historyLoadSize = 50
	// historyRetentionDays は履歴ファイルの保持日数。
	historyRetentionDays = 30
)

var (
	// pathSegmentPattern はファイルパスのセグメント検証に使う正規表現。
	pathSegmentPattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
)

// historyRecord は履歴ファイル（YYYY-MM-DD.jsonl）の 1 行スキーマ。
// WAV URL は viewer 再起動で失効するため含めない。
type historyRecord struct {
	Text        string `json:"text"`
	SpeakerName string `json:"speakerName"`
	StyleName   string `json:"styleName"`
	Timestamp   int64  `json:"timestamp"`
}

// apiHistoryResponse は GET /api/history のレスポンスペイロード。
type apiHistoryResponse struct {
	Entries []historyRecord `json:"entries"`
}

// clipEvent は SSE で配信する clip イベントのペイロード。
type clipEvent struct {
	URL         string `json:"url"`
	Text        string `json:"text"`
	SpeakerName string `json:"speakerName"`
	StyleName   string `json:"styleName"`
	// Timestamp は配信時刻の Unix ms（UTC）。ブラウザ側で HH:MM:SS に整形する。
	// セッションをまたいでも単調増加するため再起動後の id 衝突が発生しない。
	Timestamp int64 `json:"timestamp"`
}

// errorEvent は SSE で配信する error イベントのペイロード。
// Category に応じて一部フィールドは省略される（omitempty）。
type errorEvent struct {
	// ID はエラーイベントの連番（1 始まり、clipEvent とは独立）。
	ID       uint64 `json:"id"`
	Category string `json:"category"`
	Message  string `json:"message"`
	// Timestamp は配信時刻の Unix ms（UTC）。
	Timestamp int64 `json:"timestamp"`
	// Path は synthesis / file カテゴリで対象ファイル情報を伝える。
	Path string `json:"path,omitempty"`
	// Text は synthesis カテゴリでの読み上げテキスト。
	Text string `json:"text,omitempty"`
	// SpeakerName / StyleName は synthesis カテゴリで speakerLookup から解決される。
	SpeakerName string `json:"speakerName,omitempty"`
	StyleName   string `json:"styleName,omitempty"`
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

// WithOrderedSpeakerIDs は /api/status で返す speakers 配列の順序を設定するオプション。
// 設定されない場合は speakerLookup のキーを ID 昇順でソートして使用する。
func WithOrderedSpeakerIDs(ids []int) HTTPStreamOption {
	return func(p *HTTPStreamPlayer) {
		if ids != nil {
			p.orderedSpeakerIDs = ids
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

// WithWorkspacePath はキャラクター設定ファイルの読み込み元ワークスペースルートを設定するオプション。
func WithWorkspacePath(path string) HTTPStreamOption {
	return func(p *HTTPStreamPlayer) {
		if path != "" {
			p.workspacePath = path
			p.assetsDirs = []string{filepath.Join(path, workspaceAssetsDir)}
		}
	}
}

// WithAssetsDirs はキャラクター資産ディレクトリのリストを設定するオプション（index 0 が最高優先）。
func WithAssetsDirs(dirs []string) HTTPStreamOption {
	return func(p *HTTPStreamPlayer) {
		if len(dirs) > 0 {
			p.assetsDirs = dirs
		}
	}
}

// WithHistoryDir は再生履歴 JSONL ファイルの保存ディレクトリを設定するオプション。
// 設定されない場合は履歴機能が無効になる。
func WithHistoryDir(dir string) HTTPStreamOption {
	return func(p *HTTPStreamPlayer) {
		if dir != "" {
			p.historyDir = dir
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
		addr:           addr,
		logger:         slog.New(slog.DiscardHandler),
		staticFS:       staticFS,
		subscribers:    newSubscriberRegistry(),
		nowFunc:        time.Now,
		silentInterval: defaultSilentInterval,
		testPhrase:     defaultTestPhrase,
		testClipCache:  make(map[int][]byte),
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

	if err := p.buildAPIStatusJSON(); err != nil {
		return err
	}
	if err := p.buildAPICharactersJSON(); err != nil {
		return err
	}
	p.pruneOldHistory()

	lis, err := net.Listen("tcp", p.addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", p.addr, err)
	}
	p.listener = lis

	fileServer := http.FileServer(http.FS(p.staticFS))

	mux := http.NewServeMux()
	mux.Handle("/", withStaticCacheControl(fileServer))
	mux.HandleFunc("/events", p.handleEvents)
	mux.HandleFunc("/api/status", p.handleAPIStatus)
	mux.HandleFunc("/api/history", p.handleAPIHistory)
	mux.HandleFunc("/api/characters", p.handleAPICharacters)
	mux.HandleFunc("/assets/images/", p.handleCharacterImage)
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

// SetSilent は無音モードへの切り替えと silentReason を設定する。
// Start 前に呼び出すこと（Start で /api/status をキャッシュするため）。
// reason が空文字なら通常モードとして扱う。
func (p *HTTPStreamPlayer) SetSilent(reason string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if reason == "" {
		p.silent = false
		p.silentReason = ""
		return
	}
	p.silent = true
	p.silentReason = reason
}

// PlayText は WAV 合成を伴わないテキストのみの SSE 配信を行う。
// clipEvent.url は空文字で配信され、ブラウザ側は再生をスキップする。
// 配信後は固定 silentInterval だけブロックして backpressure として働く。
func (p *HTTPStreamPlayer) PlayText(ctx context.Context, meta app.PlayMeta) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	p.mu.Lock()
	started := p.started
	p.mu.Unlock()
	if !started {
		return fmt.Errorf("HTTPStreamPlayer is not started")
	}

	ts := p.nowFunc().UnixMilli()
	speakerName, styleName := p.resolveSpeaker(meta.SpeakerID)
	ev := clipEvent{
		URL:         "",
		Text:        meta.Text,
		SpeakerName: speakerName,
		StyleName:   styleName,
		Timestamp:   ts,
	}
	payload, err := json.Marshal(ev)
	if err != nil {
		return fmt.Errorf("failed to marshal clip payload: %w", err)
	}

	n, dropped := p.subscribers.broadcast(sseEventClip, payload)
	if dropped > 0 {
		p.logger.Warn("stream clip dropped for slow subscribers", "clipTimestamp", ts, "dropped", dropped)
	}
	p.logger.Info("stream clip delivered (silent)", "clipTimestamp", ts, "subscribers", n)
	p.appendHistory(ev)

	if p.silentInterval <= 0 {
		return nil
	}
	timer := time.NewTimer(p.silentInterval)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
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

	ts := p.nowFunc().UnixMilli()
	// 呼び出し元のバッファ再利用に備えてコピーを保持する。
	buf := make([]byte, len(wavData))
	copy(buf, wavData)
	p.clips.put(ts, buf)

	speakerName, styleName := p.resolveSpeaker(meta.SpeakerID)
	ev := clipEvent{
		URL:         clipPathPrefix + strconv.FormatInt(ts, 10) + clipPathSuffix,
		Text:        meta.Text,
		SpeakerName: speakerName,
		StyleName:   styleName,
		Timestamp:   ts,
	}
	payload, err := json.Marshal(ev)
	if err != nil {
		return fmt.Errorf("failed to marshal clip payload: %w", err)
	}

	n, dropped := p.subscribers.broadcast(sseEventClip, payload)
	if dropped > 0 {
		p.logger.Warn("stream clip dropped for slow subscribers", "clipTimestamp", ts, "dropped", dropped)
	}
	p.logger.Info("stream clip delivered", "clipTimestamp", ts, "subscribers", n)
	p.appendHistory(ev)

	duration, err := estimateWAVDuration(wavData)
	if err != nil {
		p.logger.Warn("failed to estimate WAV duration, skipping playback backpressure", "clipTimestamp", ts, "error", err)
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

// BroadcastError はサーバー側エラーを SSE "error" イベントとして購読者に配信する。
// Category が StreamErrorCategorySynthesis のときのみ speakerLookup を参照して
// speakerName / styleName を解決する。それ以外のカテゴリでは該当フィールドを空にする。
// Start 前の呼び出しは no-op となる（subscribers は空、clip と同様）。
func (p *HTTPStreamPlayer) BroadcastError(e app.StreamError) {
	id := p.nextErrorID.Add(1)
	ev := errorEvent{
		ID:        id,
		Category:  string(e.Category),
		Message:   e.Message,
		Timestamp: p.nowFunc().UnixMilli(),
	}
	switch e.Category {
	case app.StreamErrorCategorySynthesis:
		ev.Path = e.Path
		ev.Text = e.Text
		ev.SpeakerName, ev.StyleName = p.resolveSpeaker(e.SpeakerID)
	case app.StreamErrorCategoryFile:
		ev.Path = e.Path
	case app.StreamErrorCategoryConnection:
	}

	payload, err := json.Marshal(ev)
	if err != nil {
		p.logger.Error("failed to marshal error payload", "error", err, "category", ev.Category)
		return
	}

	n, dropped := p.subscribers.broadcast(sseEventError, payload)
	if dropped > 0 {
		p.logger.Warn("stream error event dropped for slow subscribers", "errorId", id, "dropped", dropped)
	}
	p.logger.Debug("stream error event delivered", "errorId", id, "category", ev.Category, "subscribers", n)
}

func (p *HTTPStreamPlayer) handleClip(w http.ResponseWriter, r *http.Request) {
	ts, ok := parseClipTimestamp(r.URL.Path)
	if !ok {
		http.NotFound(w, r)
		return
	}
	data, ok := p.clips.get(ts)
	if !ok {
		http.NotFound(w, r)
		return
	}
	writeWAV(w, data)
}

func parseClipTimestamp(path string) (int64, bool) {
	if !strings.HasPrefix(path, clipPathPrefix) || !strings.HasSuffix(path, clipPathSuffix) {
		return 0, false
	}
	raw := strings.TrimSuffix(strings.TrimPrefix(path, clipPathPrefix), clipPathSuffix)
	ts, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || ts <= 0 {
		return 0, false
	}
	return ts, true
}

// speakerJSON は /api/status の speakers 配列要素のペイロード。
type speakerJSON struct {
	ID          int    `json:"id"`
	SpeakerName string `json:"speakerName"`
	StyleName   string `json:"styleName"`
}

// apiStatusJSON は GET /api/status のレスポンスペイロード。
type apiStatusJSON struct {
	Silent       bool          `json:"silent"`
	SilentReason string        `json:"silentReason"`
	Speakers     []speakerJSON `json:"speakers"`
}

// characterEntry は speaker.json から変換された character エントリ。
type characterEntry struct {
	SpeakerName string `json:"speakerName"`
	StyleName   string `json:"styleName"`
	MouthClosed string `json:"mouthClosed"`
	MouthOpen   string `json:"mouthOpen"`
}

// apiCharactersJSON は GET /api/characters のレスポンスペイロード。
type apiCharactersJSON struct {
	Enabled    bool             `json:"enabled"`
	Characters []characterEntry `json:"characters"`
}

// buildAPIStatusJSON は speakerLookup と silent 状態から /api/status のレスポンスを
// Start 時に一度だけマーシャルしてキャッシュする。
// 無音モードでは speakers は空配列として固定する。
// orderedSpeakerIDs が設定されている場合はその順序を使用し、
// そうでない場合は speakerLookup のキーを ID 昇順でソートする（後方互換）。
func (p *HTTPStreamPlayer) buildAPIStatusJSON() error {
	var items []speakerJSON
	if p.silent {
		items = []speakerJSON{}
	} else {
		var ids []int
		if len(p.orderedSpeakerIDs) > 0 {
			ids = p.orderedSpeakerIDs
		} else {
			ids = make([]int, 0, len(p.speakerLookup))
			for id := range p.speakerLookup {
				ids = append(ids, id)
			}
			sort.Ints(ids)
		}
		items = make([]speakerJSON, 0, len(ids))
		for _, id := range ids {
			info := p.speakerLookup[id]
			items = append(items, speakerJSON{
				ID:          id,
				SpeakerName: info.SpeakerName,
				StyleName:   info.StyleName,
			})
		}
	}
	payload, err := json.Marshal(apiStatusJSON{
		Silent:       p.silent,
		SilentReason: p.silentReason,
		Speakers:     items,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal api status: %w", err)
	}
	p.apiStatusJSON = payload
	return nil
}

// buildAPICharactersJSON はキャラクター設定から /api/characters のレスポンスを
// Start 時に一度だけマーシャルしてキャッシュする。
// assetsDirs が空の場合は enabled=false で固定する。
// speaker.json から読み込み、有効なエントリが見つからない場合は enabled=false。
func (p *HTTPStreamPlayer) buildAPICharactersJSON() error {
	entries := []characterEntry{}
	enabled := false

	if len(p.assetsDirs) > 0 {
		mergedEntries := p.loadMergedCharacterSettings()
		if len(mergedEntries) > 0 {
			entries = mergedEntries
			enabled = true
			p.logger.Info("character settings loaded", "count", len(entries))
		}
	}

	payload, err := json.Marshal(apiCharactersJSON{
		Enabled:    enabled,
		Characters: entries,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal api characters: %w", err)
	}
	p.apiCharactersJSON = payload
	return nil
}

// loadMergedCharacterSettings は assetsDirs から優先順にキャラクター設定を読み込み、
// 同一 charID の重複は高優先側（index 0）で上書きしたエントリリストを返す。
func (p *HTTPStreamPlayer) loadMergedCharacterSettings() []characterEntry {
	seen := make(map[string]bool)
	var result []characterEntry

	for _, assetsDir := range p.assetsDirs {
		if _, err := os.Stat(assetsDir); errors.Is(err, fs.ErrNotExist) {
			p.logger.Info("assets directory not found, skipping load",
				"assetsDir", assetsDir)
			continue
		}
		fsys := os.DirFS(assetsDir)
		loadedEntries, loadErr := loadCharacterSettingsFromSpeakerJSON(fsys, assetsDir, ".", p.logger)
		if loadErr != nil {
			p.logger.Warn("failed to load character settings",
				"assetsDir", assetsDir,
				"error", loadErr)
			continue
		}
		// Group entries by charID (first path segment)
		charGroups := make(map[string][]characterEntry)
		for _, entry := range loadedEntries {
			charID := strings.SplitN(entry.MouthClosed, "/", 2)[0]
			charGroups[charID] = append(charGroups[charID], entry)
		}
		for charID, group := range charGroups {
			if !seen[charID] {
				seen[charID] = true
				result = append(result, group...)
			}
		}
	}
	return result
}

// loadCharacterSettingsFromSpeakerJSON はfsys内のassetsDir/*/speaker.jsonを走査し、
// キャラクター設定を読み込む。JSON パース失敗、画像不在、パス検証エラーなどで
// 無効なエントリは自動的に除外される。有効なエントリが 0 件の場合はエラーを返す。
func loadCharacterSettingsFromSpeakerJSON(fsys fs.FS, basePath string, assetsDir string, logger *slog.Logger) ([]characterEntry, error) {
	entries := []characterEntry{}

	assetsDirFS, err := fs.Sub(fsys, assetsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to access assets directory: %w", err)
	}

	dirs, err := fs.ReadDir(assetsDirFS, ".")
	if err != nil {
		return nil, fmt.Errorf("failed to read assets directory: %w", err)
	}

	for _, dir := range dirs {
		if !dir.IsDir() {
			continue
		}

		speakerJSONPath := filepath.Join(dir.Name(), "speaker.json")
		data, err := fs.ReadFile(assetsDirFS, speakerJSONPath)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			logger.Warn("failed to read speaker.json", "path", speakerJSONPath, "error", err)
			continue
		}

		speakerJSON, err := entity.ParseSpeakerJSON(data)
		if err != nil {
			logger.Warn("failed to parse speaker.json", "path", speakerJSONPath, "error", err)
			continue
		}

		if err := speakerJSON.Validate(); err != nil {
			logger.Warn("invalid speaker.json", "path", speakerJSONPath, "error", err)
			continue
		}

		for _, style := range speakerJSON.Styles {
			entry := characterEntry{
				SpeakerName: speakerJSON.GetSpeakerName(),
				StyleName:   style.StyleName,
				MouthClosed: filepath.Join(dir.Name(), style.MouthClosed),
				MouthOpen:   filepath.Join(dir.Name(), style.MouthOpened),
			}

			baseDir := filepath.Join(basePath, assetsDir)
			if err := validateCharacterEntryInFS(entry, assetsDirFS, baseDir); err != nil {
				logger.Warn("skipping invalid character entry", "speakerName", entry.SpeakerName, "styleName", entry.StyleName, "error", err)
				continue
			}

			entries = append(entries, entry)
		}
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("no valid character entries found")
	}

	return entries, nil
}

// validateCharacterEntryInFS は単一のキャラクター設定を fs.FS に対して検証する。
// パス検証: .. や先頭 / を含む、セグメントが [A-Za-z0-9._-]+ 以外を含む場合はエラー。
// 画像ファイル存在確認。
func validateCharacterEntryInFS(entry characterEntry, fsys fs.FS, baseDir string) error {
	for _, imgPath := range []string{entry.MouthClosed, entry.MouthOpen} {
		if imgPath == "" {
			return fmt.Errorf("image path is empty")
		}
		if err := validateImagePathInFS(imgPath, fsys, baseDir); err != nil {
			return err
		}
	}
	return nil
}

// validateImagePathInFS は相対パスの検証と fs.FS に対する存在確認を行う。
// パストラバーサル (..を含む、/で始まる) を拒否し、
// 各セグメントが [A-Za-z0-9._-]+ にマッチするか確認する。
func validateImagePathInFS(relPath string, fsys fs.FS, baseDir string) error {
	if strings.HasPrefix(relPath, "/") || strings.HasPrefix(relPath, "../") || strings.Contains(relPath, "/../") || strings.Contains(relPath, "..") {
		return fmt.Errorf("path traversal detected")
	}

	pathSegments := strings.Split(relPath, "/")
	for _, seg := range pathSegments {
		if !pathSegmentPattern.MatchString(seg) {
			return fmt.Errorf("invalid path segment: %s", seg)
		}
	}

	if _, err := fs.Stat(fsys, relPath); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			fullPath := filepath.Join(baseDir, relPath)
			return fmt.Errorf("image file not found: %s", fullPath)
		}
		return err
	}

	return nil
}

// validateImagePath は相対パスの検証と存在確認を行う。
// パストラバーサル (..を含む、/で始まる) を拒否し、
// 各セグメントが [A-Za-z0-9._-]+ にマッチするか確認する。
func validateImagePath(relPath string, baseDir string) error {
	if strings.HasPrefix(relPath, "/") || strings.HasPrefix(relPath, "../") || strings.Contains(relPath, "/../") || strings.Contains(relPath, "..") {
		return fmt.Errorf("path traversal detected")
	}

	pathSegments := strings.Split(relPath, "/")
	for _, seg := range pathSegments {
		if !pathSegmentPattern.MatchString(seg) {
			return fmt.Errorf("invalid path segment: %s", seg)
		}
	}

	fullPath := filepath.Join(baseDir, relPath)
	if _, err := os.Stat(fullPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("image file not found: %s", fullPath)
		}
		return err
	}

	return nil
}

func (p *HTTPStreamPlayer) handleAPIStatus(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = w.Write(p.apiStatusJSON)
}

func (p *HTTPStreamPlayer) handleAPICharacters(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = w.Write(p.apiCharactersJSON)
}

func (p *HTTPStreamPlayer) handleAPIHistory(w http.ResponseWriter, _ *http.Request) {
	entries := p.loadHistory(historyLoadSize)
	if entries == nil {
		entries = []historyRecord{}
	}
	payload, err := json.Marshal(apiHistoryResponse{Entries: entries})
	if err != nil {
		p.logger.Warn("failed to marshal api history", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = w.Write(payload)
}

// loadHistory は今日の履歴ファイルから末尾 n 件を読み込む。
func (p *HTTPStreamPlayer) loadHistory(n int) []historyRecord {
	if p.historyDir == "" {
		return nil
	}
	filePath := p.historyFilePath(p.nowFunc())
	data, err := os.ReadFile(filePath)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			p.logger.Warn("failed to read history file", "error", err)
		}
		return nil
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	start := 0
	if len(lines) > n {
		start = len(lines) - n
	}
	records := make([]historyRecord, 0, len(lines)-start)
	for _, line := range lines[start:] {
		if line == "" {
			continue
		}
		var r historyRecord
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			p.logger.Warn("failed to parse history line", "error", err)
			continue
		}
		records = append(records, r)
	}
	return records
}

// pruneOldHistory は historyDir 配下の 30日より古い *.jsonl を削除する（ベストエフォート）。
func (p *HTTPStreamPlayer) pruneOldHistory() {
	if p.historyDir == "" {
		return
	}
	now := p.nowFunc()
	cutoff := time.Date(now.Year(), now.Month(), now.Day()-historyRetentionDays, 0, 0, 0, 0, time.Local)

	entries, err := os.ReadDir(p.historyDir)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			p.logger.Warn("failed to read history dir for pruning", "error", err)
		}
		return
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") {
			continue
		}
		nameWithoutExt := strings.TrimSuffix(entry.Name(), ".jsonl")
		t, err := time.ParseInLocation("2006-01-02", nameWithoutExt, time.Local)
		if err != nil {
			continue
		}
		if t.Before(cutoff) {
			path := filepath.Join(p.historyDir, entry.Name())
			if err := os.Remove(path); err != nil {
				p.logger.Warn("failed to delete old history file", "path", path, "error", err)
			} else {
				p.logger.Info("deleted old history file", "path", path)
			}
		}
	}
}

// appendHistory は clip イベントを今日の履歴ファイルに追記する（ベストエフォート）。
func (p *HTTPStreamPlayer) appendHistory(ev clipEvent) {
	if p.historyDir == "" {
		return
	}
	record := historyRecord{
		Text:        ev.Text,
		SpeakerName: ev.SpeakerName,
		StyleName:   ev.StyleName,
		Timestamp:   ev.Timestamp,
	}
	data, err := json.Marshal(record)
	if err != nil {
		p.logger.Warn("failed to marshal history record", "error", err)
		return
	}
	filePath := p.historyFilePath(p.nowFunc())
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		p.logger.Warn("failed to create history directory", "error", err)
		return
	}
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		p.logger.Warn("failed to open history file", "error", err)
		return
	}
	defer func() { _ = f.Close() }()
	_, _ = fmt.Fprintf(f, "%s\n", data)
}

// historyFilePath は日付から履歴ファイルのパスを返す。
func (p *HTTPStreamPlayer) historyFilePath(t time.Time) string {
	return filepath.Join(p.historyDir, t.Format("2006-01-02")+".jsonl")
}

func (p *HTTPStreamPlayer) handleCharacterImage(w http.ResponseWriter, r *http.Request) {
	relPath := strings.TrimPrefix(r.URL.Path, "/assets/images/")
	if relPath == r.URL.Path {
		http.NotFound(w, r)
		return
	}

	if len(p.assetsDirs) > 0 {
		// assetsDirs を優先順に検索し、charID が存在する最初の dir から配信する
		segments := strings.SplitN(relPath, "/", 2)
		if len(segments) < 2 {
			http.NotFound(w, r)
			return
		}
		charID := segments[0]
		for _, assetsDir := range p.assetsDirs {
			if _, err := os.Stat(filepath.Join(assetsDir, charID)); err != nil {
				continue
			}
			if err := validateImagePath(relPath, assetsDir); err != nil {
				http.Error(w, "invalid path", http.StatusBadRequest)
				return
			}
			http.ServeFile(w, r, filepath.Join(assetsDir, relPath))
			return
		}
		http.NotFound(w, r)
		return
	}

	http.NotFound(w, r)
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
	timestamp int64
	data      []byte
}

type clipRingBuffer struct {
	mu      sync.Mutex
	cap     int
	entries []clipEntry
}

func newClipRingBuffer(capacity int) *clipRingBuffer {
	return &clipRingBuffer{cap: capacity, entries: make([]clipEntry, 0, capacity)}
}

func (b *clipRingBuffer) put(timestamp int64, data []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.entries) >= b.cap {
		// 旧エントリのバイト列を GC 対象にするため参照をクリアしてから前詰めする。
		b.entries[0] = clipEntry{}
		b.entries = b.entries[1:]
	}
	b.entries = append(b.entries, clipEntry{timestamp: timestamp, data: data})
}

func (b *clipRingBuffer) get(timestamp int64) ([]byte, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, e := range b.entries {
		if e.timestamp == timestamp {
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
