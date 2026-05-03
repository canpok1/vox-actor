package infra_test

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/canpok1/vox-actor/internal/infra"
)

func TestPollingDirWatcher_DetectsExistingFiles(t *testing.T) {
	dir := t.TempDir()

	// 既存ファイルを作成
	for _, name := range []string{"c.txt", "a.txt", "b.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("hello"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	watcher := infra.NewPollingDirWatcher(50 * time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	fileCh, _ := watcher.Watch(ctx, dir)

	var detected []string
	for i := 0; i < 3; i++ {
		select {
		case f := <-fileCh:
			detected = append(detected, filepath.Base(f))
		case <-ctx.Done():
			t.Fatal("timeout waiting for files")
		}
	}

	// 辞書順であること
	expected := []string{"a.txt", "b.txt", "c.txt"}
	for i, name := range expected {
		if detected[i] != name {
			t.Errorf("expected %s at position %d, got %s", name, i, detected[i])
		}
	}
}

func TestPollingDirWatcher_DetectsNewFiles(t *testing.T) {
	dir := t.TempDir()

	watcher := infra.NewPollingDirWatcher(50 * time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	fileCh, _ := watcher.Watch(ctx, dir)

	// 少し待ってから新規ファイルを作成
	time.Sleep(100 * time.Millisecond)
	if err := os.WriteFile(filepath.Join(dir, "new.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	select {
	case f := <-fileCh:
		if filepath.Base(f) != "new.txt" {
			t.Errorf("expected new.txt, got %s", filepath.Base(f))
		}
	case <-ctx.Done():
		t.Fatal("timeout waiting for new file")
	}
}

func TestPollingDirWatcher_IgnoresSubdirectories(t *testing.T) {
	dir := t.TempDir()

	// サブディレクトリ(done/)を作成
	if err := os.Mkdir(filepath.Join(dir, "done"), 0o755); err != nil {
		t.Fatal(err)
	}
	// ファイルを作成
	if err := os.WriteFile(filepath.Join(dir, "test.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	watcher := infra.NewPollingDirWatcher(50 * time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	fileCh, _ := watcher.Watch(ctx, dir)

	select {
	case f := <-fileCh:
		if filepath.Base(f) != "test.txt" {
			t.Errorf("expected test.txt, got %s (subdirectory should be ignored)", filepath.Base(f))
		}
	case <-ctx.Done():
		t.Fatal("timeout waiting for file")
	}

	// 他のファイルは来ないはず(done/は除外)
	select {
	case f := <-fileCh:
		t.Errorf("unexpected file detected: %s", f)
	case <-time.After(200 * time.Millisecond):
		// 期待通り
	}
}

func TestPollingDirWatcher_DoesNotDuplicateFiles(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "test.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	watcher := infra.NewPollingDirWatcher(50 * time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	fileCh, _ := watcher.Watch(ctx, dir)

	// 最初の検出
	select {
	case <-fileCh:
		// OK
	case <-ctx.Done():
		t.Fatal("timeout")
	}

	// 2回目は来ないはず（同じファイルを重複通知しない）
	select {
	case f := <-fileCh:
		t.Errorf("unexpected duplicate detection: %s", f)
	case <-time.After(200 * time.Millisecond):
		// 期待通り
	}
}

func TestPollingDirWatcher_RedetectsFileAfterRemovalAndReplacement(t *testing.T) {
	dir := t.TempDir()

	watcher := infra.NewPollingDirWatcher(50 * time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// ファイルを作成
	filePath := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(filePath, []byte("first"), 0o644); err != nil {
		t.Fatal(err)
	}

	fileCh, _ := watcher.Watch(ctx, dir)

	// 1回目の検出
	select {
	case f := <-fileCh:
		if filepath.Base(f) != "test.txt" {
			t.Fatalf("expected test.txt, got %s", filepath.Base(f))
		}
	case <-ctx.Done():
		t.Fatal("timeout waiting for first detection")
	}

	// ファイルを削除（done/への移動をシミュレート）
	if err := os.Remove(filePath); err != nil {
		t.Fatal(err)
	}

	// ポーリングが削除を検知するのを待つ
	time.Sleep(150 * time.Millisecond)

	// 同名ファイルを再配置
	if err := os.WriteFile(filePath, []byte("second"), 0o644); err != nil {
		t.Fatal(err)
	}

	// 2回目の検出（再配置されたファイルが検知されるべき）
	select {
	case f := <-fileCh:
		if filepath.Base(f) != "test.txt" {
			t.Fatalf("expected test.txt on re-detection, got %s", filepath.Base(f))
		}
	case <-ctx.Done():
		t.Fatal("timeout waiting for re-detection of replaced file")
	}
}

func TestPollingDirWatcher_RecreatesMissingDirectory(t *testing.T) {
	parent := t.TempDir()
	dir := filepath.Join(parent, "watched")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn}))

	watcher := infra.NewPollingDirWatcher(
		50*time.Millisecond,
		infra.WithPollingDirWatcherLogger(logger),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	fileCh, errCh := watcher.Watch(ctx, dir)
	t.Cleanup(func() {
		cancel()
		for range fileCh {
		}
		for range errCh {
		}
	})

	// 監視ディレクトリを削除
	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}

	// 自動再作成されるのを待つ
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		info, err := os.Stat(dir)
		if err == nil && info.IsDir() {
			break
		}
		select {
		case e := <-errCh:
			t.Fatalf("unexpected error from errCh: %v", e)
		case <-time.After(50 * time.Millisecond):
		}
	}
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("directory was not recreated: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("recreated path is not a directory")
	}

	// 再作成されたディレクトリに新規ファイルを置くと検知される
	newFile := filepath.Join(dir, "after.txt")
	if err := os.WriteFile(newFile, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	select {
	case f := <-fileCh:
		if filepath.Base(f) != "after.txt" {
			t.Errorf("expected after.txt, got %s", filepath.Base(f))
		}
	case <-ctx.Done():
		t.Fatal("timeout waiting for re-detection after recreation")
	}

	// warn ログが1回だけ出ていることを確認
	warnLines := collectLogLines(buf.String(), "watch directory was recreated")
	if len(warnLines) != 1 {
		t.Errorf("expected 1 warn log for recreation, got %d: %s", len(warnLines), buf.String())
	} else if !strings.Contains(warnLines[0], "level=WARN") {
		t.Errorf("expected WARN level log, got: %s", warnLines[0])
	}
}

func TestPollingDirWatcher_RecreateFailureGoesToErrCh(t *testing.T) {
	parent := t.TempDir()

	// 親パスをファイルにすることで、その配下のディレクトリ再作成を失敗させる
	fileAsParent := filepath.Join(parent, "file")
	if err := os.WriteFile(fileAsParent, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	// "file/watched" は親がファイルなのでMkdirAllが失敗する
	dir := filepath.Join(fileAsParent, "watched")

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn}))

	watcher := infra.NewPollingDirWatcher(
		50*time.Millisecond,
		infra.WithPollingDirWatcherLogger(logger),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, errCh := watcher.Watch(ctx, dir)

	// 再作成失敗のエラーがerrChに流れることを確認
	select {
	case err := <-errCh:
		if err == nil {
			t.Fatal("expected non-nil error")
		}
	case <-ctx.Done():
		t.Fatal("timeout waiting for error on recreate failure")
	}
}

func TestPollingDirWatcher_NonENOENTErrorGoesToErrCh(t *testing.T) {
	parent := t.TempDir()

	// 監視対象をファイル（ディレクトリではない）にする
	// os.ReadDir は ENOTDIR を返し、ENOENTではないため従来通りerrChに流れる
	notADir := filepath.Join(parent, "not-a-dir.txt")
	if err := os.WriteFile(notADir, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	watcher := infra.NewPollingDirWatcher(50 * time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, errCh := watcher.Watch(ctx, notADir)

	select {
	case err := <-errCh:
		if err == nil {
			t.Fatal("expected non-nil error")
		}
		if errors.Is(err, fs.ErrNotExist) {
			t.Errorf("expected non-ENOENT error, got ENOENT-equivalent: %v", err)
		}
	case <-ctx.Done():
		t.Fatal("timeout waiting for error")
	}
}

func TestPollingDirWatcher_MultipleDirsIndependentOnOneDeleted(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn}))

	watcher := infra.NewPollingDirWatcher(
		50*time.Millisecond,
		infra.WithPollingDirWatcherLogger(logger),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	fileCh1, errCh1 := watcher.Watch(ctx, dir1)
	fileCh2, errCh2 := watcher.Watch(ctx, dir2)
	t.Cleanup(func() {
		cancel()
		for range fileCh1 {
		}
		for range errCh1 {
		}
		for range fileCh2 {
		}
		for range errCh2 {
		}
	})

	// dir1 のみ削除
	if err := os.RemoveAll(dir1); err != nil {
		t.Fatal(err)
	}

	// dir2 に新規ファイルを置くと検知される（dir1削除の影響を受けない）
	if err := os.WriteFile(filepath.Join(dir2, "x.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	select {
	case f := <-fileCh2:
		if filepath.Base(f) != "x.txt" {
			t.Errorf("expected x.txt, got %s", filepath.Base(f))
		}
	case e := <-errCh2:
		t.Fatalf("unexpected error on dir2 errCh: %v", e)
	case <-ctx.Done():
		t.Fatal("timeout waiting for dir2 detection")
	}

	// dir1 側は errCh にエラーが流れていないこと（再作成成功している）
	select {
	case e := <-errCh1:
		t.Errorf("unexpected error on dir1 errCh: %v", e)
	default:
	}
}

func TestPollingDirWatcher_StopsOnContextCancel(t *testing.T) {
	dir := t.TempDir()

	watcher := infra.NewPollingDirWatcher(50 * time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())

	fileCh, _ := watcher.Watch(ctx, dir)

	cancel()

	// チャネルがクローズされることを確認
	select {
	case _, ok := <-fileCh:
		if ok {
			// ファイルが来た場合は、次のreceiveでクローズを確認
			select {
			case _, ok := <-fileCh:
				if ok {
					t.Error("expected channel to be closed")
				}
			case <-time.After(1 * time.Second):
				t.Fatal("timeout: channel should be closed after context cancel")
			}
		}
		// ok == false: チャネルがクローズされた
	case <-time.After(1 * time.Second):
		t.Fatal("timeout: channel should be closed after context cancel")
	}
}

func collectLogLines(logs, needle string) []string {
	var matched []string
	for _, line := range strings.Split(strings.TrimRight(logs, "\n"), "\n") {
		if strings.Contains(line, needle) {
			matched = append(matched, line)
		}
	}
	return matched
}
