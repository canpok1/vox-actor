package app_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/canpok1/vox-actor/internal/app"
	"github.com/canpok1/vox-actor/internal/domain/entity"
)

// --- WatchUsecase用モック ---

type mockFileMover struct {
	mu           sync.Mutex
	movedFiles   []string
	deletedFiles []string
	err          error
	deleteErr    error
}

func (m *mockFileMover) MoveToDone(path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.movedFiles = append(m.movedFiles, path)
	return m.err
}

func (m *mockFileMover) Delete(path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.deletedFiles = append(m.deletedFiles, path)
	return m.deleteErr
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

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("expected no error on context cancellation, got: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for Run to stop after context cancellation")
	}
}

func TestWatchUsecase_Run_ScriptParamsOverrideGlobal(t *testing.T) {
	scriptSpeaker := 7
	scriptSpeed := 0.5
	globalSpeed := 2.0

	reader := &mockScriptReader{
		scripts: []entity.Script{
			{
				Path:       "script.json",
				Text:       "ゆっくり",
				IsEmpty:    false,
				SpeakerID:  &scriptSpeaker,
				SpeedScale: &scriptSpeed,
			},
		},
	}
	client := &mockVoicevoxClient{
		query:   &entity.AudioQuery{},
		wavData: []byte("fake-wav"),
	}
	player := &mockAudioPlayer{}
	mover := &mockFileMover{}
	watcher := &mockDirWatcher{
		files: []string{"/tmp/watch/script.json"},
	}

	uc := app.NewWatchUsecase(reader, client, player, mover, watcher)
	params := app.ActParams{
		Path:      "/tmp/watch",
		SpeakerID: 3,
		Speed:     &globalSpeed,
	}

	err := uc.Run(context.Background(), params)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// CreateQueryにスクリプトのSpeakerID(7)が渡されること
	if len(client.createQueryCallArgs) != 1 {
		t.Fatalf("expected 1 CreateQuery call, got %d", len(client.createQueryCallArgs))
	}
	if client.createQueryCallArgs[0].speakerID != 7 {
		t.Errorf("expected CreateQuery speakerID 7, got %d", client.createQueryCallArgs[0].speakerID)
	}

	// Synthesizeにスクリプト単位のパラメータが渡されること
	if len(client.synthesizeArgs) != 1 {
		t.Fatalf("expected 1 Synthesize call, got %d", len(client.synthesizeArgs))
	}
	args := client.synthesizeArgs[0]
	if args.speakerID != 7 {
		t.Errorf("expected Synthesize speakerID 7, got %d", args.speakerID)
	}
	// スクリプト単位のSpeed(0.5)がグローバル(2.0)より優先される（WithOverridesでqueryに適用済み）
	if args.query.SpeedScale != 0.5 {
		t.Errorf("expected SpeedScale 0.5 (script override), got %f", args.query.SpeedScale)
	}
}

func TestWatchUsecase_Run_ScriptNoParams_UsesGlobal(t *testing.T) {
	globalSpeed := 2.0

	reader := &mockScriptReader{
		scripts: []entity.Script{
			{Path: "script.txt", Text: "普通", IsEmpty: false},
		},
	}
	client := &mockVoicevoxClient{
		query:   &entity.AudioQuery{},
		wavData: []byte("fake-wav"),
	}
	player := &mockAudioPlayer{}
	mover := &mockFileMover{}
	watcher := &mockDirWatcher{
		files: []string{"/tmp/watch/script.txt"},
	}

	uc := app.NewWatchUsecase(reader, client, player, mover, watcher)
	params := app.ActParams{
		Path:      "/tmp/watch",
		SpeakerID: 3,
		Speed:     &globalSpeed,
	}

	err := uc.Run(context.Background(), params)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(client.synthesizeArgs) != 1 {
		t.Fatalf("expected 1 Synthesize call, got %d", len(client.synthesizeArgs))
	}
	args := client.synthesizeArgs[0]
	// スクリプトにSpeedScaleがないのでグローバル(2.0)が使われる（WithOverridesでqueryに適用済み）
	if args.query.SpeedScale != 2.0 {
		t.Errorf("expected SpeedScale 2.0 (global), got %f", args.query.SpeedScale)
	}
	// SpeakerIDもデフォルト(3)が使われる
	if args.speakerID != 3 {
		t.Errorf("expected speakerID 3 (global), got %d", args.speakerID)
	}
}

func TestWatchUsecase_Run_DeleteMode_DeletesFileInsteadOfMove(t *testing.T) {
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

	uc := app.NewWatchUsecase(reader, client, player, mover, watcher, app.WithDeleteMode())
	params := app.ActParams{
		Path:      "/tmp/watch",
		SpeakerID: 3,
	}

	if err := uc.Run(context.Background(), params); err != nil {
		t.Fatal(err)
	}

	// MoveToDoneは呼ばれない
	if len(mover.movedFiles) != 0 {
		t.Errorf("expected 0 files moved to done, got: %d", len(mover.movedFiles))
	}
	// Deleteが呼ばれる
	if len(mover.deletedFiles) != 1 {
		t.Fatalf("expected 1 file deleted, got: %d", len(mover.deletedFiles))
	}
	if mover.deletedFiles[0] != "/tmp/watch/a.txt" {
		t.Errorf("expected deleted file '/tmp/watch/a.txt', got: %s", mover.deletedFiles[0])
	}
}

func TestWatchUsecase_Run_DefaultMode_MovesToDoneNotDelete(t *testing.T) {
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

	// WithDeleteModeなしで作成
	uc := app.NewWatchUsecase(reader, client, player, mover, watcher)
	params := app.ActParams{
		Path:      "/tmp/watch",
		SpeakerID: 3,
	}

	if err := uc.Run(context.Background(), params); err != nil {
		t.Fatal(err)
	}

	// MoveToDoneが呼ばれる
	if len(mover.movedFiles) != 1 {
		t.Errorf("expected 1 file moved to done, got: %d", len(mover.movedFiles))
	}
	// Deleteは呼ばれない
	if len(mover.deletedFiles) != 0 {
		t.Errorf("expected 0 files deleted, got: %d", len(mover.deletedFiles))
	}
}

func TestWatchUsecase_Run_DeleteMode_LogsFileDeleted(t *testing.T) {
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

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	uc := app.NewWatchUsecase(reader, client, player, mover, watcher, app.WithDeleteMode(), app.WithWatchLogger(logger))
	params := app.ActParams{Path: "/tmp/watch", SpeakerID: 3}

	err := uc.Run(context.Background(), params)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "file deleted") {
		t.Errorf("expected log to contain 'file deleted', got: %s", output)
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

func TestWatchUsecase_Run_FileMovedToDoneLoggedAtInfoLevel(t *testing.T) {
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

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	uc := app.NewWatchUsecase(reader, client, player, mover, watcher, app.WithWatchLogger(logger))
	params := app.ActParams{Path: "/tmp/watch", SpeakerID: 3}

	err := uc.Run(context.Background(), params)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "file moved to done") {
		t.Errorf("expected log to contain 'file moved to done' at Info level, got: %s", output)
	}
}

func TestWatchUsecase_Run_SynthesisCompletedLoggedAtInfoLevel(t *testing.T) {
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

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	uc := app.NewWatchUsecase(reader, client, player, mover, watcher, app.WithWatchLogger(logger))
	params := app.ActParams{Path: "/tmp/watch", SpeakerID: 3}

	err := uc.Run(context.Background(), params)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "synthesis completed") {
		t.Errorf("expected log to contain 'synthesis completed' at Info level, got: %s", output)
	}
}

func TestWatchUsecase_Run_LogsProcessingStatus(t *testing.T) {
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

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	uc := app.NewWatchUsecase(reader, client, player, mover, watcher, app.WithWatchLogger(logger))
	params := app.ActParams{
		Path:      "/tmp/watch",
		SpeakerID: 3,
	}

	err := uc.Run(context.Background(), params)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	output := buf.String()
	// ファイル検知のログが出力されること
	if !strings.Contains(output, "a.txt") {
		t.Errorf("expected log output to contain file path 'a.txt', got: %s", output)
	}
}

func TestWatchUsecase_Run_WatcherError_LoggedViaLogger(t *testing.T) {
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

	// エラーを先に送信し、その後ファイルを送信してからチャネルをクローズするカスタムウォッチャー
	customWatcher := &errorThenFileWatcher{
		errToSend:  errors.New("watcher test error"),
		fileToSend: "/tmp/watch/a.txt",
	}

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	uc := app.NewWatchUsecase(reader, client, player, mover, customWatcher, app.WithWatchLogger(logger))
	params := app.ActParams{
		Path:      "/tmp/watch",
		SpeakerID: 3,
	}

	err := uc.Run(context.Background(), params)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "watcher test error") {
		t.Errorf("expected log output to contain watcher error, got: %s", output)
	}
}

// errorThenFileWatcher はエラーを先に送信し、処理後にファイルを送信するウォッチャー。
type errorThenFileWatcher struct {
	errToSend  error
	fileToSend string
}

func (w *errorThenFileWatcher) Watch(_ context.Context, _ string) (<-chan string, <-chan error) {
	fileCh := make(chan string, 1)
	errCh := make(chan error, 1)

	// エラーを先に送信
	errCh <- w.errToSend
	close(errCh)

	// 少し遅延してファイルを送信（エラーが先に処理されることを保証）
	go func() {
		time.Sleep(10 * time.Millisecond)
		fileCh <- w.fileToSend
		close(fileCh)
	}()

	return fileCh, errCh
}
