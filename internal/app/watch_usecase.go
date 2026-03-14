package app

import (
	"context"
	"log"
)

// WatchUsecase はディレクトリ監視モードのユースケース。
type WatchUsecase struct {
	reader  ScriptReader
	client  VoicevoxClient
	player  AudioPlayer
	mover   FileMover
	watcher DirWatcher
}

// NewWatchUsecase は新しいWatchUsecaseを生成する。
func NewWatchUsecase(reader ScriptReader, client VoicevoxClient, player AudioPlayer, mover FileMover, watcher DirWatcher) *WatchUsecase {
	return &WatchUsecase{
		reader:  reader,
		client:  client,
		player:  player,
		mover:   mover,
		watcher: watcher,
	}
}

// Run はディレクトリ監視モードを実行する。
// 疎通確認後、ディレクトリを監視してファイルを順次処理する。
func (u *WatchUsecase) Run(ctx context.Context, params ActParams) error {
	if err := u.client.HealthCheck(ctx); err != nil {
		return err
	}

	fileCh, errCh := u.watcher.Watch(ctx, params.Path)

	for {
		select {
		case <-ctx.Done():
			return nil
		case err, ok := <-errCh:
			if !ok {
				errCh = nil
				continue
			}
			log.Printf("watcher error: %v", err)
		case path, ok := <-fileCh:
			if !ok {
				return nil
			}
			u.processFile(ctx, path, params)
		}
	}
}

// processFile は1ファイルを処理し、完了後にdone/に移動する。
// エラーが発生してもスキップしてdone/に移動する。
func (u *WatchUsecase) processFile(ctx context.Context, path string, params ActParams) {
	scripts, err := u.reader.Read(path)
	if err != nil {
		log.Printf("read error (skipping): %s: %v", path, err)
		if moveErr := u.mover.MoveToDone(path); moveErr != nil {
			log.Printf("move error: %s: %v", path, moveErr)
		}
		return
	}

	for _, script := range scripts {
		if script.IsEmpty {
			continue
		}

		query, err := u.client.CreateQuery(ctx, script.Text, params.SpeakerID)
		if err != nil {
			log.Printf("create query error (skipping): %s: %v", path, err)
			break
		}

		wavData, err := u.client.Synthesize(ctx, query, params.SpeakerID, params.Speed, params.Pitch, params.Intonation)
		if err != nil {
			log.Printf("synthesize error (skipping): %s: %v", path, err)
			break
		}

		if err := u.player.Play(ctx, wavData); err != nil {
			log.Printf("play error (skipping): %s: %v", path, err)
			break
		}
	}

	if moveErr := u.mover.MoveToDone(path); moveErr != nil {
		log.Printf("move error: %s: %v", path, moveErr)
	}
}
