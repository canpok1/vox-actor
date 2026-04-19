package app

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"sync"

	"github.com/canpok1/vox-actor/internal/domain/entity"
)

// WatchParams はWatchUsecaseのパラメータ。
type WatchParams struct {
	Paths      []string
	SpeakerID  int
	Speed      *float64
	Pitch      *float64
	Intonation *float64
	// DryRun が true の場合、VOICEVOXエンジン/音声再生は一切呼ばずログ出力のみ行う。
	// ディレクトリ監視・done/移動・削除は通常通り実施する。
	DryRun bool
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

// WatchUsecase はディレクトリ監視モードのユースケース。
type WatchUsecase struct {
	reader     ScriptReader
	client     VoicevoxClient
	player     AudioPlayer
	mover      FileMover
	watcher    DirWatcher
	logger     *slog.Logger
	deleteMode bool
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
	}
	for _, opt := range opts {
		opt(u)
	}
	return u
}

// Run はディレクトリ監視モードを実行する。
// 疎通確認後、指定された全ディレクトリを並列監視してファイルを fan-in で順次処理する。
func (u *WatchUsecase) Run(ctx context.Context, params WatchParams) error {
	u.logger.Debug("watch mode starting", "paths", params.Paths, "speakerID", params.SpeakerID)

	if !params.DryRun {
		if err := u.client.HealthCheck(ctx); err != nil {
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
			} else {
				u.logger.Debug("file deleted", "path", path)
			}
		} else {
			if moveErr := u.mover.MoveToDone(path); moveErr != nil {
				u.logger.Error("move error", "path", path, "error", moveErr)
			} else {
				u.logger.Debug("file moved to done", "path", path)
			}
		}
	}()

	scripts, err := u.reader.Read(path)
	if err != nil {
		u.logger.Error("read error (skipping)", "path", path, "error", err)
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

	cfg := synthPipelineConfig{
		defaultSpeakerID:  params.SpeakerID,
		defaultSpeed:      params.Speed,
		defaultPitch:      params.Pitch,
		defaultIntonation: params.Intonation,
		synthesisLogLevel: slog.LevelDebug,
	}
	ch := startSynthPipeline(ctx, u.client, u.logger, scripts, cfg)

	for result := range ch {
		switch result.stage {
		case synthStageQueryFailed:
			u.logger.Error("create query error (skipping script)", "path", result.script.Path, "error", result.err)
			continue
		case synthStageSynthesizeFailed:
			u.logger.Error("synthesize error (skipping script)", "path", result.script.Path, "error", result.err)
			continue
		}

		u.logger.Debug("processing script", "path", result.script.Path)

		if err := u.player.Play(ctx, result.wav); err != nil {
			u.logger.Error("play error (skipping script)", "path", result.script.Path, "error", err)
			continue
		}
		u.logger.Info(fmt.Sprintf("[%d/%d] playback completed", result.index, total), "text", truncateAndEscapeText(result.script.Text))

		// 現在のセリフ処理完了後にctxキャンセルを検知したら残りをスキップする。
		if ctx.Err() != nil {
			return
		}
	}
}

// processScriptsDryRun はDryRunモードで1ファイル分のスクリプト群を処理する。
// VOICEVOXエンジン/音声再生は一切呼ばず、既存のログ互換性を保ったまま逐次出力する。
func (u *WatchUsecase) processScriptsDryRun(_ context.Context, scripts []entity.Script, total int, params WatchParams) {
	current := 0
	for _, script := range scripts {
		if script.IsEmpty {
			u.logger.Debug("skipping empty script", "path", script.Path)
			continue
		}
		current++

		speakerID := script.ResolveSpeakerID(params.SpeakerID)
		speed := script.ResolveSpeed(params.Speed)
		pitch := script.ResolvePitch(params.Pitch)
		intonation := script.ResolveIntonation(params.Intonation)

		attrs := dryRunPlaybackAttrs(script.Text, speakerID, speed, pitch, intonation)
		u.logger.Info(fmt.Sprintf("[%d/%d] playback completed", current, total), attrs...)
	}
}
