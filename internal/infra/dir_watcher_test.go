package infra_test

import (
	"context"
	"os"
	"path/filepath"
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
