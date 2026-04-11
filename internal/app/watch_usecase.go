package app

import (
	"context"
	"log/slog"
)

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
// 疎通確認後、ディレクトリを監視してファイルを順次処理する。
func (u *WatchUsecase) Run(ctx context.Context, params ActParams) error {
	u.logger.Debug("watch mode starting", "path", params.Path, "speakerID", params.SpeakerID)

	if err := u.client.HealthCheck(ctx); err != nil {
		return err
	}
	u.logger.Info("engine health check passed")

	fileCh, errCh := u.watcher.Watch(ctx, params.Path)
	u.logger.Info("watching directory", "path", params.Path)

	for {
		select {
		case <-ctx.Done():
			u.logger.Info("watch mode stopping")
			return nil
		case err, ok := <-errCh:
			if !ok {
				errCh = nil
				continue
			}
			u.logger.Error("watcher error", "error", err)
		case path, ok := <-fileCh:
			if !ok {
				return nil
			}
			u.logger.Info("file detected", "path", path)
			u.processFile(ctx, path, params)
		}
	}
}

// processFile は1ファイルを処理し、完了後にdone/に移動する（deleteModeの場合は削除する）。
// エラーが発生してもスキップして後処理を実行する。
func (u *WatchUsecase) processFile(ctx context.Context, path string, params ActParams) {
	defer func() {
		if u.deleteMode {
			if delErr := u.mover.Delete(path); delErr != nil {
				u.logger.Error("delete error", "path", path, "error", delErr)
			} else {
				u.logger.Info("file deleted", "path", path)
			}
		} else {
			if moveErr := u.mover.MoveToDone(path); moveErr != nil {
				u.logger.Error("move error", "path", path, "error", moveErr)
			} else {
				u.logger.Info("file moved to done", "path", path)
			}
		}
	}()

	scripts, err := u.reader.Read(path)
	if err != nil {
		u.logger.Error("read error (skipping)", "path", path, "error", err)
		return
	}

	for _, script := range scripts {
		if script.IsEmpty {
			u.logger.Debug("skipping empty script", "path", script.Path)
			continue
		}

		u.logger.Info("processing script", "path", script.Path)

		// セリフ単位パラメータがあればグローバルパラメータより優先する
		speakerID := script.ResolveSpeakerID(params.SpeakerID)
		speed := script.ResolveSpeed(params.Speed)
		pitch := script.ResolvePitch(params.Pitch)
		intonation := script.ResolveIntonation(params.Intonation)

		query, err := u.client.CreateQuery(ctx, script.Text, speakerID)
		if err != nil {
			u.logger.Error("create query error (skipping script)", "path", script.Path, "error", err)
			continue
		}
		u.logger.Debug("query created", "path", script.Path)

		q := query.WithOverrides(speed, pitch, intonation)
		wavData, err := u.client.Synthesize(ctx, &q, speakerID)
		if err != nil {
			u.logger.Error("synthesize error (skipping script)", "path", script.Path, "error", err)
			continue
		}
		u.logger.Info("synthesis completed", "path", script.Path, "wavSize", len(wavData))

		if err := u.player.Play(ctx, wavData); err != nil {
			u.logger.Error("play error (skipping script)", "path", script.Path, "error", err)
			continue
		}
		u.logger.Info("playback completed", "path", script.Path)
	}
}
