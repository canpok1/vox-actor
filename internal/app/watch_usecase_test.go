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
	"github.com/canpok1/vox-actor/internal/infra/logging"
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
	params := app.WatchParams{
		Paths:     []string{"/tmp/watch"},
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
	if len(player.playMetas) != 1 || player.playMetas[0].SpeakerID != 3 {
		t.Errorf("expected PlayMeta.SpeakerID=3 (default), got: %+v", player.playMetas)
	}
}

func TestWatchUsecase_Run_PlayReceivesScriptResolvedSpeakerID(t *testing.T) {
	overrideID := 11
	reader := &mockScriptReader{
		scripts: []entity.Script{
			{Path: "a.txt", Text: "default", IsEmpty: false},
			{Path: "b.txt", Text: "override", IsEmpty: false, SpeakerID: &overrideID},
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
	params := app.WatchParams{Paths: []string{"/tmp/watch"}, SpeakerID: 3}

	if err := uc.Run(context.Background(), params); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(player.playMetas) != 2 {
		t.Fatalf("expected 2 Play calls, got: %d", len(player.playMetas))
	}
	if player.playMetas[0].SpeakerID != 3 {
		t.Errorf("expected first PlayMeta.SpeakerID=3 (default), got: %d", player.playMetas[0].SpeakerID)
	}
	if player.playMetas[1].SpeakerID != 11 {
		t.Errorf("expected second PlayMeta.SpeakerID=11 (script override), got: %d", player.playMetas[1].SpeakerID)
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
	params := app.WatchParams{
		Paths:     []string{"/tmp/watch"},
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
	params := app.WatchParams{
		Paths:     []string{"/tmp/watch"},
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
	params := app.WatchParams{
		Paths:     []string{"/tmp/watch"},
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
	params := app.WatchParams{
		Paths:     []string{"/tmp/watch"},
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
	params := app.WatchParams{Paths: []string{"/tmp/watch"}, SpeakerID: 3}

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
	params := app.WatchParams{Paths: []string{"/tmp/watch"}, SpeakerID: 3}

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
	params := app.WatchParams{
		Paths:     []string{"/tmp/watch"},
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
	params := app.WatchParams{
		Paths:     []string{"/tmp/watch"},
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
	params := app.WatchParams{
		Paths:     []string{"/tmp/watch"},
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
	params := app.WatchParams{
		Paths:     []string{"/tmp/watch"},
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

func TestWatchUsecase_Run_DeleteMode_FileDeletedSuppressedAtInfoLevel(t *testing.T) {
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
	logger := slog.New(logging.NewHumanHandler(&buf, &logging.HumanHandlerOptions{Level: slog.LevelInfo, NoColor: true}))

	uc := app.NewWatchUsecase(reader, client, player, mover, watcher, app.WithDeleteMode(), app.WithWatchLogger(logger))
	params := app.WatchParams{Paths: []string{"/tmp/watch"}, SpeakerID: 3}

	err := uc.Run(context.Background(), params)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	output := buf.String()
	if strings.Contains(output, "file deleted") {
		t.Errorf("expected 'file deleted' NOT in INFO log (demoted to DEBUG), got: %s", output)
	}
}

func TestWatchUsecase_Run_DeleteMode_FileDeletedAppearsAtDebugLevel(t *testing.T) {
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
	logger := slog.New(logging.NewHumanHandler(&buf, &logging.HumanHandlerOptions{Level: slog.LevelDebug, NoColor: true}))

	uc := app.NewWatchUsecase(reader, client, player, mover, watcher, app.WithDeleteMode(), app.WithWatchLogger(logger))
	params := app.WatchParams{Paths: []string{"/tmp/watch"}, SpeakerID: 3}

	err := uc.Run(context.Background(), params)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "file deleted") {
		t.Errorf("expected log to contain 'file deleted' at Debug level, got: %s", output)
	}
}

func TestWatchUsecase_Run_DryRun_SkipsClientAndPlayerAndMovesToDone(t *testing.T) {
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
	logger := slog.New(logging.NewHumanHandler(&buf, &logging.HumanHandlerOptions{
		Level:   slog.LevelInfo,
		NoColor: true,
		DryRun:  true,
	}))

	uc := app.NewWatchUsecase(reader, client, player, mover, watcher, app.WithWatchLogger(logger))
	params := app.WatchParams{
		Paths:     []string{"/tmp/watch"},
		SpeakerID: 3,
		DryRun:    true,
	}

	if err := uc.Run(context.Background(), params); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// dry-run: 合成・再生は呼ばれない
	if client.createQueryCalls != 0 {
		t.Errorf("expected 0 CreateQuery calls in dry-run, got: %d", client.createQueryCalls)
	}
	if player.playCalls != 0 {
		t.Errorf("expected 0 Play calls in dry-run, got: %d", player.playCalls)
	}
	// ただしdone/への移動は通常通り実施
	if len(mover.movedFiles) != 1 {
		t.Fatalf("expected 1 file moved to done in dry-run, got: %d", len(mover.movedFiles))
	}
	if mover.movedFiles[0] != "/tmp/watch/a.txt" {
		t.Errorf("expected moved file '/tmp/watch/a.txt', got: %s", mover.movedFiles[0])
	}

	output := buf.String()
	if !strings.Contains(output, "[dry run] [1/1] playback completed") {
		t.Errorf("expected '[dry run] [1/1] playback completed' in log, got: %s", output)
	}
	for _, want := range []string{"text=おはよう", "speaker=3", "speed=default", "pitch=default", "intonation=default"} {
		if !strings.Contains(output, want) {
			t.Errorf("expected %q in log, got: %s", want, output)
		}
	}
	// watch の playback completed には path を含めない
	playbackLine := findLineContaining(output, "playback completed")
	if playbackLine == "" {
		t.Fatalf("expected a 'playback completed' log line, got: %s", output)
	}
	if strings.Contains(playbackLine, "path=") {
		t.Errorf("expected no 'path=...' on playback completed, got: %s", playbackLine)
	}
}

// findLineContaining は s の中から substr を含む最初の行を返す（見つからなければ空文字）。
func findLineContaining(s, substr string) string {
	for line := range strings.SplitSeq(s, "\n") {
		if strings.Contains(line, substr) {
			return line
		}
	}
	return ""
}

func TestWatchUsecase_Run_DryRun_NoHealthCheck(t *testing.T) {
	reader := &mockScriptReader{
		scripts: []entity.Script{
			{Path: "a.txt", Text: "おはよう", IsEmpty: false},
		},
	}
	client := &mockVoicevoxClient{
		healthCheckErr: errors.New("connection refused"),
	}
	player := &mockAudioPlayer{}
	mover := &mockFileMover{}
	watcher := &mockDirWatcher{
		files: []string{"/tmp/watch/a.txt"},
	}

	uc := app.NewWatchUsecase(reader, client, player, mover, watcher)
	params := app.WatchParams{
		Paths:     []string{"/tmp/watch"},
		SpeakerID: 3,
		DryRun:    true,
	}

	err := uc.Run(context.Background(), params)
	if err != nil {
		t.Fatalf("expected no error in dry-run even with HealthCheck failing, got: %v", err)
	}
}

func TestWatchUsecase_Run_DryRun_DeleteModeStillDeletes(t *testing.T) {
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
	params := app.WatchParams{
		Paths:     []string{"/tmp/watch"},
		SpeakerID: 3,
		DryRun:    true,
	}

	if err := uc.Run(context.Background(), params); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// dry-runでもdelete modeは維持される
	if len(mover.deletedFiles) != 1 {
		t.Fatalf("expected 1 file deleted in dry-run delete mode, got: %d", len(mover.deletedFiles))
	}
	if len(mover.movedFiles) != 0 {
		t.Errorf("expected 0 files moved in dry-run delete mode, got: %d", len(mover.movedFiles))
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

func TestWatchUsecase_Run_FileMovedToDoneSuppressedAtInfoLevel(t *testing.T) {
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
	logger := slog.New(logging.NewHumanHandler(&buf, &logging.HumanHandlerOptions{Level: slog.LevelInfo, NoColor: true}))

	uc := app.NewWatchUsecase(reader, client, player, mover, watcher, app.WithWatchLogger(logger))
	params := app.WatchParams{Paths: []string{"/tmp/watch"}, SpeakerID: 3}

	err := uc.Run(context.Background(), params)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	output := buf.String()
	if strings.Contains(output, "file moved to done") {
		t.Errorf("expected 'file moved to done' NOT in INFO log (demoted to DEBUG), got: %s", output)
	}
}

func TestWatchUsecase_Run_SynthesisCompletedSuppressedAtInfoLevel(t *testing.T) {
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
	logger := slog.New(logging.NewHumanHandler(&buf, &logging.HumanHandlerOptions{Level: slog.LevelInfo, NoColor: true}))

	uc := app.NewWatchUsecase(reader, client, player, mover, watcher, app.WithWatchLogger(logger))
	params := app.WatchParams{Paths: []string{"/tmp/watch"}, SpeakerID: 3}

	err := uc.Run(context.Background(), params)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	output := buf.String()
	for _, msg := range []string{"processing script", "synthesis completed", "query created"} {
		if strings.Contains(output, msg) {
			t.Errorf("expected %q NOT in INFO log (demoted to DEBUG), got: %s", msg, output)
		}
	}
}

func TestWatchUsecase_Run_PlayReceivesScriptText(t *testing.T) {
	reader := &mockScriptReader{
		scripts: []entity.Script{
			{Path: "a.txt", Text: "こんにちはなのだ", IsEmpty: false},
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
	params := app.WatchParams{
		Paths:     []string{"/tmp/watch"},
		SpeakerID: 3,
	}

	if err := uc.Run(context.Background(), params); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(player.playMetas) != 1 {
		t.Fatalf("expected 1 Play call, got: %d", len(player.playMetas))
	}
	if player.playMetas[0].Text != "こんにちはなのだ" {
		t.Errorf("expected PlayMeta.Text=%q, got: %q", "こんにちはなのだ", player.playMetas[0].Text)
	}
}

func TestWatchUsecase_Run_PlaybackCompletedAtInfoIncludesProgressAndText(t *testing.T) {
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
	logger := slog.New(logging.NewHumanHandler(&buf, &logging.HumanHandlerOptions{Level: slog.LevelInfo, NoColor: true}))

	uc := app.NewWatchUsecase(reader, client, player, mover, watcher, app.WithWatchLogger(logger))
	params := app.WatchParams{Paths: []string{"/tmp/watch"}, SpeakerID: 3}

	if err := uc.Run(context.Background(), params); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "[1/1] playback completed") {
		t.Errorf("expected log to contain '[1/1] playback completed', got: %s", output)
	}
	if !strings.Contains(output, "text=おはよう") {
		t.Errorf("expected log to contain 'text=おはよう', got: %s", output)
	}
}

func TestWatchUsecase_Run_PlaybackCompletedTruncatesLongTextAndEscapesNewlines(t *testing.T) {
	longText := "あいうえおかきくけこさし\nすせそたちつ"
	reader := &mockScriptReader{
		scripts: []entity.Script{
			{Path: "a.txt", Text: longText, IsEmpty: false},
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
	logger := slog.New(logging.NewHumanHandler(&buf, &logging.HumanHandlerOptions{Level: slog.LevelInfo, NoColor: true}))

	uc := app.NewWatchUsecase(reader, client, player, mover, watcher, app.WithWatchLogger(logger))
	params := app.WatchParams{Paths: []string{"/tmp/watch"}, SpeakerID: 3}

	if err := uc.Run(context.Background(), params); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	output := buf.String()
	want := `text=あいうえおかきくけこさし\nすせ...`
	if !strings.Contains(output, want) {
		t.Errorf("expected log to contain %q, got: %s", want, output)
	}
}

func TestWatchUsecase_Run_PlaybackCompletedSkipsEmptyAndNumeratesContiguously(t *testing.T) {
	reader := &mockScriptReader{
		scripts: []entity.Script{
			{Path: "1.txt", Text: "first", IsEmpty: false},
			{Path: "2.txt", Text: "", IsEmpty: true},
			{Path: "3.txt", Text: "third", IsEmpty: false},
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
	logger := slog.New(logging.NewHumanHandler(&buf, &logging.HumanHandlerOptions{Level: slog.LevelInfo, NoColor: true}))

	uc := app.NewWatchUsecase(reader, client, player, mover, watcher, app.WithWatchLogger(logger))
	params := app.WatchParams{Paths: []string{"/tmp/watch"}, SpeakerID: 3}

	if err := uc.Run(context.Background(), params); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	output := buf.String()
	for _, expected := range []string{"[1/2] playback completed", "[2/2] playback completed"} {
		if !strings.Contains(output, expected) {
			t.Errorf("expected log to contain %q, got: %s", expected, output)
		}
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
	logger := slog.New(logging.NewHumanHandler(&buf, &logging.HumanHandlerOptions{Level: slog.LevelInfo, NoColor: true}))

	uc := app.NewWatchUsecase(reader, client, player, mover, watcher, app.WithWatchLogger(logger))
	params := app.WatchParams{
		Paths:     []string{"/tmp/watch"},
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
	logger := slog.New(logging.NewHumanHandler(&buf, &logging.HumanHandlerOptions{Level: slog.LevelInfo, NoColor: true}))

	uc := app.NewWatchUsecase(reader, client, player, mover, customWatcher, app.WithWatchLogger(logger))
	params := app.WatchParams{
		Paths:     []string{"/tmp/watch"},
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

// perPathDirWatcher は呼ばれた dir に応じて異なるファイルを返すウォッチャー。
type perPathDirWatcher struct {
	filesByPath map[string][]string
	errsByPath  map[string][]error
}

func (w *perPathDirWatcher) Watch(_ context.Context, dir string) (<-chan string, <-chan error) {
	files := w.filesByPath[dir]
	errs := w.errsByPath[dir]
	fileCh := make(chan string, len(files))
	errCh := make(chan error, len(errs))
	for _, f := range files {
		fileCh <- f
	}
	for _, e := range errs {
		errCh <- e
	}
	close(fileCh)
	close(errCh)
	return fileCh, errCh
}

func TestWatchUsecase_Run_MultiplePaths_FanIn(t *testing.T) {
	reader := &mockScriptReader{
		scripts: []entity.Script{
			{Path: "any.txt", Text: "こんにちは", IsEmpty: false},
		},
	}
	client := &mockVoicevoxClient{
		query:   &entity.AudioQuery{},
		wavData: []byte("fake-wav"),
	}
	player := &mockAudioPlayer{}
	mover := &mockFileMover{}
	watcher := &perPathDirWatcher{
		filesByPath: map[string][]string{
			"/tmp/a": {"/tmp/a/1.txt", "/tmp/a/2.txt"},
			"/tmp/b": {"/tmp/b/1.txt"},
		},
	}

	uc := app.NewWatchUsecase(reader, client, player, mover, watcher)
	params := app.WatchParams{
		Paths:     []string{"/tmp/a", "/tmp/b"},
		SpeakerID: 3,
	}

	if err := uc.Run(context.Background(), params); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if player.playCalls != 3 {
		t.Errorf("expected 3 Play calls across both dirs, got: %d", player.playCalls)
	}
	if len(mover.movedFiles) != 3 {
		t.Errorf("expected 3 files moved to done, got: %d", len(mover.movedFiles))
	}

	moved := make(map[string]bool)
	for _, p := range mover.movedFiles {
		moved[p] = true
	}
	for _, want := range []string{"/tmp/a/1.txt", "/tmp/a/2.txt", "/tmp/b/1.txt"} {
		if !moved[want] {
			t.Errorf("expected %s to be processed, missing: %v", want, mover.movedFiles)
		}
	}
}

func TestWatchUsecase_Run_DuplicatePaths_LogsWarningAndMerges(t *testing.T) {
	reader := &mockScriptReader{
		scripts: []entity.Script{
			{Path: "any.txt", Text: "こんにちは", IsEmpty: false},
		},
	}
	client := &mockVoicevoxClient{
		query:   &entity.AudioQuery{},
		wavData: []byte("fake-wav"),
	}
	player := &mockAudioPlayer{}
	mover := &mockFileMover{}
	watcher := &perPathDirWatcher{
		filesByPath: map[string][]string{
			"/tmp/a": {"/tmp/a/1.txt"},
		},
	}

	var buf bytes.Buffer
	logger := slog.New(logging.NewHumanHandler(&buf, &logging.HumanHandlerOptions{Level: slog.LevelInfo, NoColor: true}))

	uc := app.NewWatchUsecase(reader, client, player, mover, watcher, app.WithWatchLogger(logger))
	params := app.WatchParams{
		Paths:     []string{"/tmp/a", "/tmp/a"},
		SpeakerID: 3,
	}

	if err := uc.Run(context.Background(), params); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// 重複指定時は1回だけ監視される
	if player.playCalls != 1 {
		t.Errorf("expected 1 Play call (duplicate paths merged), got: %d", player.playCalls)
	}

	output := buf.String()
	if !strings.Contains(output, "duplicate watch path") {
		t.Errorf("expected warning log for duplicate path, got: %s", output)
	}
}

func TestWatchUsecase_Run_OneWatcherError_OthersContinue(t *testing.T) {
	reader := &mockScriptReader{
		scripts: []entity.Script{
			{Path: "any.txt", Text: "こんにちは", IsEmpty: false},
		},
	}
	client := &mockVoicevoxClient{
		query:   &entity.AudioQuery{},
		wavData: []byte("fake-wav"),
	}
	player := &mockAudioPlayer{}
	mover := &mockFileMover{}
	watcher := &perPathDirWatcher{
		filesByPath: map[string][]string{
			"/tmp/a": {"/tmp/a/1.txt"},
			"/tmp/b": nil,
		},
		errsByPath: map[string][]error{
			"/tmp/b": {errors.New("bad watcher for b")},
		},
	}

	var buf bytes.Buffer
	logger := slog.New(logging.NewHumanHandler(&buf, &logging.HumanHandlerOptions{Level: slog.LevelInfo, NoColor: true}))

	uc := app.NewWatchUsecase(reader, client, player, mover, watcher, app.WithWatchLogger(logger))
	params := app.WatchParams{
		Paths:     []string{"/tmp/a", "/tmp/b"},
		SpeakerID: 3,
	}

	if err := uc.Run(context.Background(), params); err != nil {
		t.Fatalf("expected no error (watcher errors should be logged), got: %v", err)
	}

	// /tmp/a のファイルは処理される
	if player.playCalls != 1 {
		t.Errorf("expected 1 Play call, got: %d", player.playCalls)
	}

	// エラーログが出力される
	output := buf.String()
	if !strings.Contains(output, "bad watcher for b") {
		t.Errorf("expected watcher error log, got: %s", output)
	}
}
