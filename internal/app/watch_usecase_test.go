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

// --- #284 無音モード ---

// silentModePlayer は WithSilent 経路で使われる PlayText 対応の AudioPlayer。
type silentModePlayer struct {
	mu            sync.Mutex
	playCalls     int
	playTextCalls int
	playTextMetas []app.PlayMeta
}

func (p *silentModePlayer) Play(_ context.Context, _ []byte, _ app.PlayMeta) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.playCalls++
	return nil
}

func (p *silentModePlayer) PlayText(_ context.Context, meta app.PlayMeta) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.playTextCalls++
	p.playTextMetas = append(p.playTextMetas, meta)
	return nil
}

func TestWatchUsecase_Run_SilentMode_SkipsSynthAndCallsPlayText(t *testing.T) {
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
	player := &silentModePlayer{}
	mover := &mockFileMover{}
	watcher := &mockDirWatcher{
		files: []string{"/tmp/watch/a.txt"},
	}

	uc := app.NewWatchUsecase(reader, client, player, mover, watcher, app.WithSilent())
	params := app.WatchParams{
		Paths:     []string{"/tmp/watch"},
		SpeakerID: 3,
	}

	if err := uc.Run(context.Background(), params); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// 無音モードでは CreateQuery / Synthesize は呼ばれない
	if client.createQueryCalls != 0 {
		t.Errorf("expected 0 CreateQuery calls in silent mode, got: %d", client.createQueryCalls)
	}
	if client.synthesizeCalls != 0 {
		t.Errorf("expected 0 Synthesize calls in silent mode, got: %d", client.synthesizeCalls)
	}
	// Play も呼ばれない
	if player.playCalls != 0 {
		t.Errorf("expected 0 Play calls in silent mode, got: %d", player.playCalls)
	}
	// 各スクリプトが PlayText で1回ずつ配信される
	if player.playTextCalls != 2 {
		t.Fatalf("expected 2 PlayText calls, got: %d", player.playTextCalls)
	}
	if player.playTextMetas[0].SpeakerID != 3 || player.playTextMetas[0].Text != "default" {
		t.Errorf("unexpected meta[0]: %+v", player.playTextMetas[0])
	}
	if player.playTextMetas[1].SpeakerID != 11 || player.playTextMetas[1].Text != "override" {
		t.Errorf("unexpected meta[1]: %+v", player.playTextMetas[1])
	}
	// done/ 移動も従来通り行われる
	if len(mover.movedFiles) != 1 {
		t.Errorf("expected 1 file moved to done, got: %d", len(mover.movedFiles))
	}
}

func TestWatchUsecase_Run_SilentMode_SkipsHealthCheck(t *testing.T) {
	reader := &mockScriptReader{
		scripts: []entity.Script{{Path: "a.txt", Text: "x", IsEmpty: false}},
	}
	// 無音モードでは HealthCheck すら呼ばないので、HealthCheck でエラーを返しても処理が進むこと
	client := &mockVoicevoxClient{
		healthCheckErr: errors.New("engine down"),
	}
	player := &silentModePlayer{}
	mover := &mockFileMover{}
	watcher := &mockDirWatcher{files: []string{"/tmp/watch/a.txt"}}

	uc := app.NewWatchUsecase(reader, client, player, mover, watcher, app.WithSilent())
	params := app.WatchParams{Paths: []string{"/tmp/watch"}, SpeakerID: 3}

	if err := uc.Run(context.Background(), params); err != nil {
		t.Fatalf("expected no error in silent mode even with HealthCheck error, got: %v", err)
	}
	if player.playTextCalls != 1 {
		t.Errorf("expected 1 PlayText call, got: %d", player.playTextCalls)
	}
}

func TestWatchUsecase_Run_SilentMode_EmptyScriptsSkipped(t *testing.T) {
	reader := &mockScriptReader{
		scripts: []entity.Script{
			{Path: "empty.txt", Text: "", IsEmpty: true},
			{Path: "full.txt", Text: "こんにちは", IsEmpty: false},
		},
	}
	client := &mockVoicevoxClient{}
	player := &silentModePlayer{}
	mover := &mockFileMover{}
	watcher := &mockDirWatcher{files: []string{"/tmp/watch/a.txt"}}

	uc := app.NewWatchUsecase(reader, client, player, mover, watcher, app.WithSilent())
	params := app.WatchParams{Paths: []string{"/tmp/watch"}, SpeakerID: 3}

	if err := uc.Run(context.Background(), params); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if player.playTextCalls != 1 {
		t.Errorf("expected 1 PlayText call (empty script skipped), got: %d", player.playTextCalls)
	}
}

// --- #285 サーバー側エラー配信 ---

// recordingErrorPlayer は AudioPlayer + textPlayer + ErrorBroadcaster を実装し、
// watch_usecase から broadcast されたエラーを記録する。
type recordingErrorPlayer struct {
	mu              sync.Mutex
	playErr         error
	playTextErr     error
	playCalls       int
	playTextCalls   int
	broadcastErrors []app.StreamError
}

func (p *recordingErrorPlayer) Play(_ context.Context, _ []byte, _ app.PlayMeta) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.playCalls++
	return p.playErr
}

func (p *recordingErrorPlayer) PlayText(_ context.Context, _ app.PlayMeta) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.playTextCalls++
	return p.playTextErr
}

func (p *recordingErrorPlayer) BroadcastError(e app.StreamError) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.broadcastErrors = append(p.broadcastErrors, e)
}

func (p *recordingErrorPlayer) errors() []app.StreamError {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]app.StreamError, len(p.broadcastErrors))
	copy(out, p.broadcastErrors)
	return out
}

func findError(errs []app.StreamError, category app.StreamErrorCategory) (app.StreamError, bool) {
	for _, e := range errs {
		if e.Category == category {
			return e, true
		}
	}
	return app.StreamError{}, false
}

func TestWatchUsecase_Run_BroadcastsFileError_OnReadFailure(t *testing.T) {
	reader := &mockScriptReader{err: errors.New("read denied")}
	client := &mockVoicevoxClient{query: &entity.AudioQuery{}, wavData: []byte("w")}
	player := &recordingErrorPlayer{}
	mover := &mockFileMover{}
	watcher := &mockDirWatcher{files: []string{"/tmp/watch/a.txt"}}

	uc := app.NewWatchUsecase(reader, client, player, mover, watcher)
	params := app.WatchParams{Paths: []string{"/tmp/watch"}, SpeakerID: 3}

	if err := uc.Run(context.Background(), params); err != nil {
		t.Fatalf("Run: %v", err)
	}

	errs := player.errors()
	e, ok := findError(errs, app.StreamErrorCategoryFile)
	if !ok {
		t.Fatalf("expected file category error, got: %+v", errs)
	}
	if e.Path != "/tmp/watch/a.txt" {
		t.Errorf("expected path=/tmp/watch/a.txt, got %q", e.Path)
	}
	if !strings.Contains(e.Message, "read denied") {
		t.Errorf("expected message to contain underlying error, got %q", e.Message)
	}
}

func TestWatchUsecase_Run_BroadcastsFileError_OnMoveFailure(t *testing.T) {
	reader := &mockScriptReader{
		scripts: []entity.Script{{Path: "a.txt", Text: "hi", IsEmpty: false}},
	}
	client := &mockVoicevoxClient{query: &entity.AudioQuery{}, wavData: []byte("w")}
	player := &recordingErrorPlayer{}
	mover := &mockFileMover{err: errors.New("permission denied")}
	watcher := &mockDirWatcher{files: []string{"/tmp/watch/a.txt"}}

	uc := app.NewWatchUsecase(reader, client, player, mover, watcher)
	params := app.WatchParams{Paths: []string{"/tmp/watch"}, SpeakerID: 3}
	if err := uc.Run(context.Background(), params); err != nil {
		t.Fatalf("Run: %v", err)
	}

	errs := player.errors()
	e, ok := findError(errs, app.StreamErrorCategoryFile)
	if !ok {
		t.Fatalf("expected file category error, got: %+v", errs)
	}
	if e.Path != "/tmp/watch/a.txt" {
		t.Errorf("expected path=/tmp/watch/a.txt, got %q", e.Path)
	}
	if !strings.Contains(e.Message, "permission denied") {
		t.Errorf("expected message to contain underlying error, got %q", e.Message)
	}
}

func TestWatchUsecase_Run_BroadcastsFileError_OnDeleteFailure(t *testing.T) {
	reader := &mockScriptReader{
		scripts: []entity.Script{{Path: "a.txt", Text: "hi", IsEmpty: false}},
	}
	client := &mockVoicevoxClient{query: &entity.AudioQuery{}, wavData: []byte("w")}
	player := &recordingErrorPlayer{}
	mover := &mockFileMover{deleteErr: errors.New("unlink failed")}
	watcher := &mockDirWatcher{files: []string{"/tmp/watch/a.txt"}}

	uc := app.NewWatchUsecase(reader, client, player, mover, watcher, app.WithDeleteMode())
	params := app.WatchParams{Paths: []string{"/tmp/watch"}, SpeakerID: 3}
	if err := uc.Run(context.Background(), params); err != nil {
		t.Fatalf("Run: %v", err)
	}

	errs := player.errors()
	e, ok := findError(errs, app.StreamErrorCategoryFile)
	if !ok {
		t.Fatalf("expected file category error, got: %+v", errs)
	}
	if !strings.Contains(e.Message, "unlink failed") {
		t.Errorf("expected message to contain underlying error, got %q", e.Message)
	}
}

func TestWatchUsecase_Run_BroadcastsSynthesisError_OnCreateQueryFailure(t *testing.T) {
	reader := &mockScriptReader{
		scripts: []entity.Script{{Path: "a.txt", Text: "hi", IsEmpty: false}},
	}
	client := &mockVoicevoxClient{createQueryErr: errors.New("bad query")}
	player := &recordingErrorPlayer{}
	mover := &mockFileMover{}
	watcher := &mockDirWatcher{files: []string{"/tmp/watch/a.txt"}}

	uc := app.NewWatchUsecase(reader, client, player, mover, watcher)
	params := app.WatchParams{Paths: []string{"/tmp/watch"}, SpeakerID: 3}
	if err := uc.Run(context.Background(), params); err != nil {
		t.Fatalf("Run: %v", err)
	}

	errs := player.errors()
	e, ok := findError(errs, app.StreamErrorCategorySynthesis)
	if !ok {
		t.Fatalf("expected synthesis category error, got: %+v", errs)
	}
	if e.Path != "a.txt" {
		t.Errorf("expected path=a.txt, got %q", e.Path)
	}
	if e.Text != "hi" {
		t.Errorf("expected text=hi, got %q", e.Text)
	}
	if e.SpeakerID != 3 {
		t.Errorf("expected speakerID=3, got %d", e.SpeakerID)
	}
	if !strings.Contains(e.Message, "bad query") {
		t.Errorf("expected message to contain underlying error, got %q", e.Message)
	}
}

func TestWatchUsecase_Run_BroadcastsSynthesisError_OnSynthesizeFailure(t *testing.T) {
	reader := &mockScriptReader{
		scripts: []entity.Script{{Path: "a.txt", Text: "hi", IsEmpty: false}},
	}
	client := &mockVoicevoxClient{query: &entity.AudioQuery{}, synthesizeErr: errors.New("synth fail")}
	player := &recordingErrorPlayer{}
	mover := &mockFileMover{}
	watcher := &mockDirWatcher{files: []string{"/tmp/watch/a.txt"}}

	uc := app.NewWatchUsecase(reader, client, player, mover, watcher)
	params := app.WatchParams{Paths: []string{"/tmp/watch"}, SpeakerID: 3}
	if err := uc.Run(context.Background(), params); err != nil {
		t.Fatalf("Run: %v", err)
	}

	errs := player.errors()
	e, ok := findError(errs, app.StreamErrorCategorySynthesis)
	if !ok {
		t.Fatalf("expected synthesis category error, got: %+v", errs)
	}
	if !strings.Contains(e.Message, "synth fail") {
		t.Errorf("expected message to contain underlying error, got %q", e.Message)
	}
}

func TestWatchUsecase_Run_BroadcastsSynthesisError_OnPlayFailure(t *testing.T) {
	reader := &mockScriptReader{
		scripts: []entity.Script{{Path: "a.txt", Text: "hi", IsEmpty: false}},
	}
	client := &mockVoicevoxClient{query: &entity.AudioQuery{}, wavData: []byte("w")}
	player := &recordingErrorPlayer{playErr: errors.New("play fail")}
	mover := &mockFileMover{}
	watcher := &mockDirWatcher{files: []string{"/tmp/watch/a.txt"}}

	uc := app.NewWatchUsecase(reader, client, player, mover, watcher)
	params := app.WatchParams{Paths: []string{"/tmp/watch"}, SpeakerID: 3}
	if err := uc.Run(context.Background(), params); err != nil {
		t.Fatalf("Run: %v", err)
	}

	errs := player.errors()
	e, ok := findError(errs, app.StreamErrorCategorySynthesis)
	if !ok {
		t.Fatalf("expected synthesis category error, got: %+v", errs)
	}
	if !strings.Contains(e.Message, "play fail") {
		t.Errorf("expected message to contain underlying error, got %q", e.Message)
	}
}

func TestWatchUsecase_Run_BroadcastsSynthesisError_OnSilentPlayTextFailure(t *testing.T) {
	reader := &mockScriptReader{
		scripts: []entity.Script{{Path: "a.txt", Text: "hi", IsEmpty: false}},
	}
	client := &mockVoicevoxClient{}
	player := &recordingErrorPlayer{playTextErr: errors.New("silent play fail")}
	mover := &mockFileMover{}
	watcher := &mockDirWatcher{files: []string{"/tmp/watch/a.txt"}}

	uc := app.NewWatchUsecase(reader, client, player, mover, watcher, app.WithSilent())
	params := app.WatchParams{Paths: []string{"/tmp/watch"}, SpeakerID: 3}
	if err := uc.Run(context.Background(), params); err != nil {
		t.Fatalf("Run: %v", err)
	}

	errs := player.errors()
	e, ok := findError(errs, app.StreamErrorCategorySynthesis)
	if !ok {
		t.Fatalf("expected synthesis category error, got: %+v", errs)
	}
	if !strings.Contains(e.Message, "silent play fail") {
		t.Errorf("expected message to contain underlying error, got %q", e.Message)
	}
}

func TestWatchUsecase_Run_BroadcastsConnectionError_OnHealthCheckFailure(t *testing.T) {
	reader := &mockScriptReader{}
	client := &mockVoicevoxClient{healthCheckErr: errors.New("engine unreachable")}
	player := &recordingErrorPlayer{}
	mover := &mockFileMover{}
	watcher := &mockDirWatcher{}

	uc := app.NewWatchUsecase(reader, client, player, mover, watcher)
	params := app.WatchParams{Paths: []string{"/tmp/watch"}, SpeakerID: 3}

	if err := uc.Run(context.Background(), params); err == nil {
		t.Fatal("expected error from HealthCheck, got nil")
	}

	errs := player.errors()
	e, ok := findError(errs, app.StreamErrorCategoryConnection)
	if !ok {
		t.Fatalf("expected connection category error, got: %+v", errs)
	}
	if !strings.Contains(e.Message, "engine unreachable") {
		t.Errorf("expected message to contain underlying error, got %q", e.Message)
	}
}

// --- GenerateWavFilename テスト ---
// DONE: 基本形: <UnixMs>_<text>.wav が生成される               → TestGenerateWavFilename_Basic
// DONE: テキスト20文字超は先頭20文字に切り捨て               → TestGenerateWavFilename_Truncates20Runes
// DONE: / を _ に置換する                                       → TestGenerateWavFilename_SanitizesForwardSlash
// DONE: \ を _ に置換する                                       → TestGenerateWavFilename_SanitizesBackslash
// DONE: 制御文字を _ に置換する                                 → TestGenerateWavFilename_SanitizesControlChars

func TestGenerateWavFilename_Basic(t *testing.T) {
	got := app.GenerateWavFilename("こんにちは", 1000)
	want := "1000_こんにちは.wav"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestGenerateWavFilename_Truncates20Runes(t *testing.T) {
	text := "あいうえおかきくけこさしすせそたちつてと超過分"
	got := app.GenerateWavFilename(text, 2000)
	want := "2000_あいうえおかきくけこさしすせそたちつてと.wav"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestGenerateWavFilename_SanitizesForwardSlash(t *testing.T) {
	got := app.GenerateWavFilename("a/b", 3000)
	want := "3000_a_b.wav"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestGenerateWavFilename_SanitizesBackslash(t *testing.T) {
	got := app.GenerateWavFilename("a\\b", 4000)
	want := "4000_a_b.wav"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestGenerateWavFilename_SanitizesControlChars(t *testing.T) {
	got := app.GenerateWavFilename("a\nb", 5000)
	want := "5000_a_b.wav"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// --- WatchUsecase WavSaver 統合テスト ---
// DONE: SaveWavDir 指定時にローカル合成で wavSaver.Save が呼ばれる     → TestWatchUsecase_Run_SavesWav_WhenSaveWavDirSet
// DONE: SaveWavDir が空の場合は wavSaver.Save は呼ばれない             → TestWatchUsecase_Run_DoesNotSaveWav_WhenSaveWavDirEmpty
// DONE: DryRun 時は wavSaver.Save は呼ばれない                         → TestWatchUsecase_Run_DoesNotSaveWav_WhenDryRun
// DONE: WithSilent（viewer 経路）時は wavSaver.Save は呼ばれない       → TestWatchUsecase_Run_DoesNotSaveWav_WhenSilent

type mockWavSaverForWatch struct {
	mu         sync.Mutex
	savedPaths []string
	savedData  [][]byte
	err        error
}

func (m *mockWavSaverForWatch) Save(path string, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.savedPaths = append(m.savedPaths, path)
	m.savedData = append(m.savedData, data)
	return m.err
}

func TestWatchUsecase_Run_SavesWav_WhenSaveWavDirSet(t *testing.T) {
	reader := &mockScriptReader{
		scripts: []entity.Script{
			{Path: "a.txt", Text: "こんにちは", IsEmpty: false},
		},
	}
	wavBytes := []byte("fake-wav")
	client := &mockVoicevoxClient{
		query:   &entity.AudioQuery{},
		wavData: wavBytes,
	}
	player := &mockAudioPlayer{}
	mover := &mockFileMover{}
	watcher := &mockDirWatcher{files: []string{"/tmp/watch/a.txt"}}
	saver := &mockWavSaverForWatch{}
	fixedNowMs := int64(9999)

	uc := app.NewWatchUsecase(reader, client, player, mover, watcher,
		app.WithWatchWavSaver(saver),
		app.WithWatchNowFunc(func() time.Time { return time.UnixMilli(fixedNowMs) }),
	)
	params := app.WatchParams{
		Paths:      []string{"/tmp/watch"},
		SpeakerID:  3,
		SaveWavDir: "/out",
	}
	if err := uc.Run(context.Background(), params); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(saver.savedPaths) != 1 {
		t.Fatalf("expected 1 save call, got %d", len(saver.savedPaths))
	}
	wantPath := "/out/" + app.GenerateWavFilename("こんにちは", fixedNowMs)
	if saver.savedPaths[0] != wantPath {
		t.Errorf("saved path = %q, want %q", saver.savedPaths[0], wantPath)
	}
	if !bytes.Equal(saver.savedData[0], wavBytes) {
		t.Errorf("saved data does not match wav bytes")
	}
}

func TestWatchUsecase_Run_DoesNotSaveWav_WhenSaveWavDirEmpty(t *testing.T) {
	reader := &mockScriptReader{
		scripts: []entity.Script{
			{Path: "a.txt", Text: "テスト", IsEmpty: false},
		},
	}
	client := &mockVoicevoxClient{query: &entity.AudioQuery{}, wavData: []byte("w")}
	player := &mockAudioPlayer{}
	mover := &mockFileMover{}
	watcher := &mockDirWatcher{files: []string{"/tmp/watch/a.txt"}}
	saver := &mockWavSaverForWatch{}

	uc := app.NewWatchUsecase(reader, client, player, mover, watcher,
		app.WithWatchWavSaver(saver),
	)
	params := app.WatchParams{
		Paths:      []string{"/tmp/watch"},
		SpeakerID:  3,
		SaveWavDir: "",
	}
	if err := uc.Run(context.Background(), params); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(saver.savedPaths) != 0 {
		t.Errorf("expected no save calls, got %d: %v", len(saver.savedPaths), saver.savedPaths)
	}
}

func TestWatchUsecase_Run_DoesNotSaveWav_WhenDryRun(t *testing.T) {
	reader := &mockScriptReader{
		scripts: []entity.Script{
			{Path: "a.txt", Text: "テスト", IsEmpty: false},
		},
	}
	client := &mockVoicevoxClient{query: &entity.AudioQuery{}, wavData: []byte("w")}
	player := &mockAudioPlayer{}
	mover := &mockFileMover{}
	watcher := &mockDirWatcher{files: []string{"/tmp/watch/a.txt"}}
	saver := &mockWavSaverForWatch{}

	uc := app.NewWatchUsecase(reader, client, player, mover, watcher,
		app.WithWatchWavSaver(saver),
	)
	params := app.WatchParams{
		Paths:      []string{"/tmp/watch"},
		SpeakerID:  3,
		SaveWavDir: "/out",
		DryRun:     true,
	}
	if err := uc.Run(context.Background(), params); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(saver.savedPaths) != 0 {
		t.Errorf("expected no save calls during dry-run, got %d", len(saver.savedPaths))
	}
}

func TestWatchUsecase_Run_DoesNotSaveWav_WhenSilent(t *testing.T) {
	reader := &mockScriptReader{
		scripts: []entity.Script{
			{Path: "a.txt", Text: "テスト", IsEmpty: false},
		},
	}
	client := &mockVoicevoxClient{query: &entity.AudioQuery{}, wavData: []byte("w")}
	player := &recordingErrorPlayer{}
	mover := &mockFileMover{}
	watcher := &mockDirWatcher{files: []string{"/tmp/watch/a.txt"}}
	saver := &mockWavSaverForWatch{}

	uc := app.NewWatchUsecase(reader, client, player, mover, watcher,
		app.WithWatchWavSaver(saver),
		app.WithSilent(),
	)
	params := app.WatchParams{
		Paths:      []string{"/tmp/watch"},
		SpeakerID:  3,
		SaveWavDir: "/out",
	}
	if err := uc.Run(context.Background(), params); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(saver.savedPaths) != 0 {
		t.Errorf("expected no save calls in silent mode, got %d", len(saver.savedPaths))
	}
}
