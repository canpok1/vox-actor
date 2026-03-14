package app_test

import (
	"context"
	"errors"
	"sort"
	"sync"
	"testing"

	"github.com/canpok1/vox-actor/internal/app"
	"github.com/canpok1/vox-actor/internal/domain/entity"
)

// --- WatchUsecase用モック ---

type mockFileMover struct {
	mu         sync.Mutex
	movedFiles []string
	err        error
}

func (m *mockFileMover) MoveToDone(path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.movedFiles = append(m.movedFiles, path)
	return m.err
}

type mockDirWatcher struct {
	files []string
	errs  []error
}

func (m *mockDirWatcher) Watch(ctx context.Context, _ string) (<-chan string, <-chan error) {
	fileCh := make(chan string, len(m.files))
	errCh := make(chan error, len(m.errs))

	for _, f := range m.files {
		fileCh <- f
	}
	for _, e := range m.errs {
		errCh <- e
	}
	close(fileCh)
	close(errCh)

	return fileCh, errCh
}

// --- テスト ---

func TestWatchUsecase_Run_ProcessesFilesFromWatcher(t *testing.T) {
	reader := &mockScriptReader{
		scripts: []entity.Script{
			{Path: "a.txt", Text: "おはよう", IsEmpty: false},
		},
	}
	client := &mockVoicevoxClient{
		query:   &entity.AudioQuery{},
		wavData: []byte("fake-wav"),
	}
	player := &mockAudioPlayer{}
	mover := &mockFileMover{}
	watcher := &mockDirWatcher{
		files: []string{"/tmp/watch/a.txt"},
	}

	uc := app.NewWatchUsecase(reader, client, player, mover, watcher)
	params := app.ActParams{
		Path:      "/tmp/watch",
		SpeakerID: 3,
	}

	err := uc.Run(context.Background(), params)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if client.createQueryCalls != 1 {
		t.Errorf("expected 1 CreateQuery call, got: %d", client.createQueryCalls)
	}
	if player.playCalls != 1 {
		t.Errorf("expected 1 Play call, got: %d", player.playCalls)
	}
}

func TestWatchUsecase_Run_MovesFileToDoneAfterProcessing(t *testing.T) {
	reader := &mockScriptReader{
		scripts: []entity.Script{
			{Path: "a.txt", Text: "おはよう", IsEmpty: false},
		},
	}
	client := &mockVoicevoxClient{
		query:   &entity.AudioQuery{},
		wavData: []byte("fake-wav"),
	}
	player := &mockAudioPlayer{}
	mover := &mockFileMover{}
	watcher := &mockDirWatcher{
		files: []string{"/tmp/watch/a.txt"},
	}

	uc := app.NewWatchUsecase(reader, client, player, mover, watcher)
	params := app.ActParams{
		Path:      "/tmp/watch",
		SpeakerID: 3,
	}

	if err := uc.Run(context.Background(), params); err != nil {
		t.Fatal(err)
	}

	if len(mover.movedFiles) != 1 {
		t.Fatalf("expected 1 file moved to done, got: %d", len(mover.movedFiles))
	}
	if mover.movedFiles[0] != "/tmp/watch/a.txt" {
		t.Errorf("expected moved file '/tmp/watch/a.txt', got: %s", mover.movedFiles[0])
	}
}

func TestWatchUsecase_Run_EmptyFileSkippedAndMovedToDone(t *testing.T) {
	reader := &mockScriptReader{
		scripts: []entity.Script{
			{Path: "empty.txt", Text: "", IsEmpty: true},
		},
	}
	client := &mockVoicevoxClient{
		query:   &entity.AudioQuery{},
		wavData: []byte("fake-wav"),
	}
	player := &mockAudioPlayer{}
	mover := &mockFileMover{}
	watcher := &mockDirWatcher{
		files: []string{"/tmp/watch/empty.txt"},
	}

	uc := app.NewWatchUsecase(reader, client, player, mover, watcher)
	params := app.ActParams{
		Path:      "/tmp/watch",
		SpeakerID: 3,
	}

	if err := uc.Run(context.Background(), params); err != nil {
		t.Fatal(err)
	}

	// 空ファイルは音声合成されない
	if client.createQueryCalls != 0 {
		t.Errorf("expected 0 CreateQuery calls for empty file, got: %d", client.createQueryCalls)
	}
	// ただしdone/には移動される
	if len(mover.movedFiles) != 1 {
		t.Fatalf("expected 1 file moved to done, got: %d", len(mover.movedFiles))
	}
}

func TestWatchUsecase_Run_ReadError_SkipsAndMovesToDone(t *testing.T) {
	reader := &mockScriptReader{
		err: errors.New("invalid file"),
	}
	client := &mockVoicevoxClient{
		query:   &entity.AudioQuery{},
		wavData: []byte("fake-wav"),
	}
	player := &mockAudioPlayer{}
	mover := &mockFileMover{}
	watcher := &mockDirWatcher{
		files: []string{"/tmp/watch/bad.txt"},
	}

	uc := app.NewWatchUsecase(reader, client, player, mover, watcher)
	params := app.ActParams{
		Path:      "/tmp/watch",
		SpeakerID: 3,
	}

	// 不正ファイルはスキップしてエラーにならない
	err := uc.Run(context.Background(), params)
	if err != nil {
		t.Fatalf("expected no error (file should be skipped), got: %v", err)
	}

	// done/には移動される
	if len(mover.movedFiles) != 1 {
		t.Fatalf("expected 1 file moved to done, got: %d", len(mover.movedFiles))
	}
}

func TestWatchUsecase_Run_MultipleFiles_ProcessedInOrder(t *testing.T) {
	// DirWatcherが返すファイルの順序で処理される
	reader := &mockScriptReader{
		scripts: []entity.Script{
			{Path: "file.txt", Text: "text", IsEmpty: false},
		},
	}
	client := &mockVoicevoxClient{
		query:   &entity.AudioQuery{},
		wavData: []byte("fake-wav"),
	}
	player := &mockAudioPlayer{}
	mover := &mockFileMover{}
	watcher := &mockDirWatcher{
		files: []string{"/tmp/watch/a.txt", "/tmp/watch/b.txt", "/tmp/watch/c.txt"},
	}

	uc := app.NewWatchUsecase(reader, client, player, mover, watcher)
	params := app.ActParams{
		Path:      "/tmp/watch",
		SpeakerID: 3,
	}

	if err := uc.Run(context.Background(), params); err != nil {
		t.Fatal(err)
	}

	// 3ファイル処理されている
	if client.createQueryCalls != 3 {
		t.Errorf("expected 3 CreateQuery calls, got: %d", client.createQueryCalls)
	}

	// 3ファイルがdone/に移動されている
	if len(mover.movedFiles) != 3 {
		t.Fatalf("expected 3 files moved to done, got: %d", len(mover.movedFiles))
	}

	// 辞書順かチェック
	sorted := make([]string, len(mover.movedFiles))
	copy(sorted, mover.movedFiles)
	sort.Strings(sorted)
	for i, f := range mover.movedFiles {
		if f != sorted[i] {
			t.Errorf("expected files to be processed in sorted order, got: %v", mover.movedFiles)
			break
		}
	}
}

func TestWatchUsecase_Run_HealthCheckError(t *testing.T) {
	reader := &mockScriptReader{}
	client := &mockVoicevoxClient{
		healthCheckErr: errors.New("connection refused"),
	}
	player := &mockAudioPlayer{}
	mover := &mockFileMover{}
	watcher := &mockDirWatcher{}

	uc := app.NewWatchUsecase(reader, client, player, mover, watcher)
	params := app.ActParams{Path: "/tmp/watch", SpeakerID: 3}

	err := uc.Run(context.Background(), params)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestWatchUsecase_Run_ContextCancelled_StopsProcessing(t *testing.T) {
	// チャネルが閉じていない場合、contextキャンセルで停止する
	fileCh := make(chan string)
	errCh := make(chan error)

	// カスタムwatcherを使用
	customWatcher := &blockingDirWatcher{fileCh: fileCh, errCh: errCh}

	reader := &mockScriptReader{
		scripts: []entity.Script{
			{Path: "file.txt", Text: "text", IsEmpty: false},
		},
	}
	client := &mockVoicevoxClient{
		query:   &entity.AudioQuery{},
		wavData: []byte("fake-wav"),
	}
	player := &mockAudioPlayer{}
	mover := &mockFileMover{}

	uc := app.NewWatchUsecase(reader, client, player, mover, customWatcher)
	params := app.ActParams{Path: "/tmp/watch", SpeakerID: 3}

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- uc.Run(ctx, params)
	}()

	// 1ファイル送信後にキャンセル
	fileCh <- "/tmp/watch/a.txt"
	cancel()

	err := <-done
	if err != nil {
		t.Fatalf("expected no error on context cancellation, got: %v", err)
	}
}

// blockingDirWatcher はテスト用のブロッキングウォッチャー。
type blockingDirWatcher struct {
	fileCh chan string
	errCh  chan error
}

func (w *blockingDirWatcher) Watch(_ context.Context, _ string) (<-chan string, <-chan error) {
	return w.fileCh, w.errCh
}
