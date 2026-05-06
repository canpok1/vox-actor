package app

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/canpok1/vox-actor/internal/domain/entity"
)

// GenerateWavFilename はタイムスタンプとテキストから WAV ファイル名を生成する。
// テキストはパス区切り文字と制御文字を _ に置換し、先頭 20 ルーンに切り捨てる。
// 書式: <nowMs>_<sanitized20>.wav
func GenerateWavFilename(text string, nowMs int64) string {
	runes := []rune(text)
	if len(runes) > 20 {
		runes = runes[:20]
	}
	var b strings.Builder
	for _, r := range runes {
		if r == '/' || r == '\\' || unicode.IsControl(r) {
			b.WriteRune('_')
		} else {
			b.WriteRune(r)
		}
	}
	return fmt.Sprintf("%d_%s.wav", nowMs, b.String())
}

// WatchParams はWatchUsecaseのパラメータ。
type WatchParams struct {
	Paths      []string
	SpeakerID  entity.SpeakerID
	Speed      *float64
	Pitch      *float64
	Intonation *float64
	// DryRun が true の場合、VOICEVOXエンジン/音声再生は一切呼ばずログ出力のみ行う。
	// ディレクトリ監視・done/移動・削除は通常通り実施する。
	DryRun bool
	// SaveWavDir が空でない場合、ローカル合成した WAV をこのディレクトリに保存する。
	// DryRun 時・viewer 送信経路では保存しない。
	SaveWavDir string
}

// WatchOption はWatchUsecaseの生成時に指定するオプション。
type WatchOption func(*WatchUsecase)

// WithWatchLogger はロガーを設定するオプション。
func WithWatchLogger(logger *slog.Logger) WatchOption {
	return func(u *WatchUsecase) {
		u.logger = logger
	}
}

// WithDeleteMode は処理済みファイルを削除するモードを有効にするオプション。
func WithDeleteMode() WatchOption {
	return func(u *WatchUsecase) {
		u.deleteMode = true
	}
}

// WithSilent は無音モード（合成パイプラインを使わずテキストのみ配信するモード）を有効にする。
// player は AudioPlayer に加えて textPlayer（PlayText(ctx, meta)）を実装している必要がある。
// 無音モードでは HealthCheck / CreateQuery / Synthesize を一切呼ばない。
func WithSilent() WatchOption {
	return func(u *WatchUsecase) {
		u.silent = true
	}
}

// WithWatchWavSaver は WAV 保存機能を有効にするオプション。
func WithWatchWavSaver(saver WavSaver) WatchOption {
	return func(u *WatchUsecase) {
		u.wavSaver = saver
	}
}

// WithWatchNowFunc は時刻取得関数を差し替えるオプション（テスト用）。
func WithWatchNowFunc(now func() time.Time) WatchOption {
	return func(u *WatchUsecase) {
		u.nowFunc = now
	}
}

// textPlayer は無音モードで WAV を伴わないセリフ配信を受け取るインターフェース。
// WithSilent を有効にした場合、player はこのインターフェースを実装している必要がある。
type textPlayer interface {
	PlayText(ctx context.Context, meta PlayMeta) error
}

// broadcastError は player が ErrorBroadcaster を実装している場合のみ、
// 配信画面向けのエラー通知をブロードキャストする。それ以外は no-op。
func (u *WatchUsecase) broadcastError(e StreamError) {
	if b, ok := u.player.(ErrorBroadcaster); ok {
		b.BroadcastError(e)
	}
}

// broadcastSynthesisError は synthesis カテゴリのエラーを対象スクリプト情報付きで配信する。
func (u *WatchUsecase) broadcastSynthesisError(script entity.Script, speakerID entity.SpeakerID, err error) {
	u.broadcastError(StreamError{
		Category:  StreamErrorCategorySynthesis,
		Message:   err.Error(),
		Path:      script.Path,
		Text:      script.Text,
		SpeakerID: speakerID,
	})
}

// broadcastFileError は file カテゴリのエラーをファイルパス付きで配信する。
func (u *WatchUsecase) broadcastFileError(path string, err error) {
	u.broadcastError(StreamError{
		Category: StreamErrorCategoryFile,
		Message:  err.Error(),
		Path:     path,
	})
}

// WatchUsecase はディレクトリ監視モードのユースケース。
type WatchUsecase struct {
	reader     ScriptReader
	client     VoicevoxClient
	player     AudioPlayer
	mover      FileMover
	watcher    DirWatcher
	logger     *slog.Logger
	deleteMode bool
	// silent が true の場合、HealthCheck / 合成パイプラインを一切呼ばず
	// player.PlayText 経由でテキストのみ配信する。
	silent   bool
	wavSaver WavSaver
	nowFunc  func() time.Time
}

// NewWatchUsecase は新しいWatchUsecaseを生成する。
func NewWatchUsecase(reader ScriptReader, client VoicevoxClient, player AudioPlayer, mover FileMover, watcher DirWatcher, opts ...WatchOption) *WatchUsecase {
	u := &WatchUsecase{
		reader:  reader,
		client:  client,
		player:  player,
		mover:   mover,
		watcher: watcher,
		logger:  discardLogger(),
		nowFunc: time.Now,
	}
	for _, opt := range opts {
		opt(u)
	}
	return u
}

// Run はディレクトリ監視モードを実行する。
// 疎通確認後、指定された全ディレクトリを並列監視してファイルを fan-in で順次処理する。
// 無音モード（WithSilent）では HealthCheck を一切呼ばず、ファイル検知時も合成パイプラインを使わず
// player.PlayText 経由でテキストのみ配信する。
func (u *WatchUsecase) Run(ctx context.Context, params WatchParams) error {
	u.logger.Debug("watch mode starting", "paths", params.Paths, "speakerID", params.SpeakerID, "silent", u.silent)

	if !params.DryRun && !u.silent {
		if err := u.client.HealthCheck(ctx); err != nil {
			u.broadcastError(StreamError{
				Category: StreamErrorCategoryConnection,
				Message:  err.Error(),
			})
			return err
		}
		u.logger.Info("engine health check passed")
	}

	paths := u.dedupePaths(params.Paths)
	fileCh := u.fanInWatchers(ctx, paths)

	for _, p := range paths {
		u.logger.Info("watching directory", "path", p)
	}

	for {
		select {
		case <-ctx.Done():
			u.logger.Info("watch mode stopping")
			return nil
		case path, ok := <-fileCh:
			if !ok {
				return nil
			}
			u.logger.Info("file detected", "path", path)
			u.processFile(ctx, path, params)
		}
	}
}

// dedupePaths は重複するパスを除去し、警告ログを出力する。
// 絶対パスに正規化して比較する。
func (u *WatchUsecase) dedupePaths(paths []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(paths))
	for _, p := range paths {
		key := p
		if abs, err := filepath.Abs(p); err == nil {
			key = abs
		}
		if seen[key] {
			u.logger.Warn("duplicate watch path, merging", "path", p)
			continue
		}
		seen[key] = true
		result = append(result, p)
	}
	return result
}

// fanInWatchers は指定された各パスで DirWatcher.Watch を起動し、
// 検知ファイルを単一チャネルに fan-in する。全 watcher 停止時にチャネルをクローズする。
func (u *WatchUsecase) fanInWatchers(ctx context.Context, paths []string) <-chan string {
	fileCh := make(chan string)
	var wg sync.WaitGroup

	for _, path := range paths {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()
			u.forwardWatcher(ctx, p, fileCh)
		}(path)
	}

	go func() {
		wg.Wait()
		close(fileCh)
	}()

	return fileCh
}

// forwardWatcher は1ディレクトリの監視結果を fan-in 先チャネルへ転送する。
// エラーはログ出力のみ行い監視を継続する。
func (u *WatchUsecase) forwardWatcher(ctx context.Context, path string, fileCh chan<- string) {
	sub, errSub := u.watcher.Watch(ctx, path)
	for sub != nil || errSub != nil {
		select {
		case <-ctx.Done():
			return
		case f, ok := <-sub:
			if !ok {
				sub = nil
				continue
			}
			select {
			case fileCh <- f:
			case <-ctx.Done():
				return
			}
		case e, ok := <-errSub:
			if !ok {
				errSub = nil
				continue
			}
			u.logger.Error("watcher error", "path", path, "error", e)
		}
	}
}

// processFile は1ファイルを処理し、完了後にdone/に移動する（deleteModeの場合は削除する）。
// エラーが発生してもスキップして後処理を実行する。
func (u *WatchUsecase) processFile(ctx context.Context, path string, params WatchParams) {
	defer func() {
		if u.deleteMode {
			if delErr := u.mover.Delete(path); delErr != nil {
				u.logger.Error("delete error", "path", path, "error", delErr)
				u.broadcastFileError(path, delErr)
			} else {
				u.logger.Debug("file deleted", "path", path)
			}
		} else {
			if moveErr := u.mover.MoveToDone(path); moveErr != nil {
				u.logger.Error("move error", "path", path, "error", moveErr)
				u.broadcastFileError(path, moveErr)
			} else {
				u.logger.Debug("file moved to done", "path", path)
			}
		}
	}()

	scripts, err := u.reader.Read(path)
	if err != nil {
		u.logger.Error("read error (skipping)", "path", path, "error", err)
		u.broadcastFileError(path, err)
		return
	}

	total := 0
	for _, s := range scripts {
		if !s.IsEmpty {
			total++
		}
	}

	if params.DryRun {
		u.processScriptsDryRun(ctx, scripts, total, params)
		return
	}

	if u.silent {
		u.processScriptsSilent(ctx, scripts, total, params)
		return
	}

	cfg := synthPipelineConfig{
		defaultSpeakerID: params.SpeakerID,
		defaultOverrides: entity.SynthOverrides{
			Speed:      params.Speed,
			Pitch:      params.Pitch,
			Intonation: params.Intonation,
		},
		synthesisLogLevel: slog.LevelDebug,
	}
	ch := startSynthPipeline(ctx, u.client, u.logger, scripts, cfg)

	for result := range ch {
		speakerID := result.script.ResolveSpeakerID(params.SpeakerID)
		switch result.stage {
		case synthStageQueryFailed:
			u.logger.Error("create query error (skipping script)", "path", result.script.Path, "error", result.err)
			u.broadcastSynthesisError(result.script, speakerID, result.err)
			continue
		case synthStageSynthesizeFailed:
			u.logger.Error("synthesize error (skipping script)", "path", result.script.Path, "error", result.err)
			u.broadcastSynthesisError(result.script, speakerID, result.err)
			continue
		}

		u.logger.Debug("processing script", "path", result.script.Path)

		if params.SaveWavDir != "" && u.wavSaver != nil {
			filename := GenerateWavFilename(result.script.Text, u.nowFunc().UnixMilli())
			savePath := filepath.Join(params.SaveWavDir, filename)
			if err := u.wavSaver.Save(savePath, result.wav); err != nil {
				u.logger.Error("save wav error (skipping save)", "path", savePath, "error", err)
			} else {
				u.logger.Info("wav saved", "path", savePath)
			}
		}

		if err := u.player.Play(ctx, result.wav, PlayMeta{Text: result.script.Text, SpeakerID: speakerID}); err != nil {
			u.logger.Error("play error (skipping script)", "path", result.script.Path, "error", err)
			u.broadcastSynthesisError(result.script, speakerID, err)
			continue
		}
		u.logger.Info(fmt.Sprintf("[%d/%d] playback completed", result.index, total), "text", truncateAndEscapeText(result.script.Text))

		// 現在のセリフ処理完了後にctxキャンセルを検知したら残りをスキップする。
		if ctx.Err() != nil {
			return
		}
	}
}

// processScriptsSilent は無音モードで1ファイル分のスクリプト群を処理する。
// VOICEVOX の合成を一切呼ばず、各スクリプトを player.PlayText 経由で SSE 配信する。
// player が textPlayer を実装していない場合はエラーログを出してスキップする。
func (u *WatchUsecase) processScriptsSilent(ctx context.Context, scripts []entity.Script, total int, params WatchParams) {
	tp, ok := u.player.(textPlayer)
	if !ok {
		u.logger.Error("silent mode requires a player that implements PlayText; skipping")
		return
	}
	current := 0
	for _, script := range scripts {
		if ctx.Err() != nil {
			return
		}
		if script.IsEmpty {
			u.logger.Debug("skipping empty script", "path", script.Path)
			continue
		}
		current++
		speakerID := script.ResolveSpeakerID(params.SpeakerID)
		if err := tp.PlayText(ctx, PlayMeta{Text: script.Text, SpeakerID: speakerID}); err != nil {
			u.logger.Error("silent play error (skipping script)", "path", script.Path, "error", err)
			u.broadcastSynthesisError(script, speakerID, err)
			continue
		}
		u.logger.Info(fmt.Sprintf("[%d/%d] playback completed (silent)", current, total), "text", truncateAndEscapeText(script.Text))
	}
}

// processScriptsDryRun はDryRunモードで1ファイル分のスクリプト群を処理する。
// VOICEVOXエンジン/音声再生は一切呼ばず、既存のログ互換性を保ったまま逐次出力する。
func (u *WatchUsecase) processScriptsDryRun(ctx context.Context, scripts []entity.Script, total int, params WatchParams) {
	defaultOverrides := entity.SynthOverrides{Speed: params.Speed, Pitch: params.Pitch, Intonation: params.Intonation}
	current := 0
	for _, script := range scripts {
		if ctx.Err() != nil {
			return
		}
		if script.IsEmpty {
			u.logger.Debug("skipping empty script", "path", script.Path)
			continue
		}
		current++

		speakerID := script.ResolveSpeakerID(params.SpeakerID)
		overrides := script.ResolveOverrides(defaultOverrides)

		attrs := dryRunPlaybackAttrs(script.Text, speakerID, overrides)
		u.logger.Info(fmt.Sprintf("[%d/%d] playback completed", current, total), attrs...)
	}
}
