package infra_test

// テストリスト: ViewerLock
//
// 受け入れ条件 vs テスト関数のマッピング:
// - ロック取得成功時にPIDと起動時刻をファイルに書き込む      → TestAcquireViewerLock_WritesLockFile
// - 2つ目のロック取得は ViewerAlreadyRunningError を返す     → TestAcquireViewerLock_AlreadyLocked
// - ViewerAlreadyRunningError に先行PIDと起動時刻が含まれる  → TestAcquireViewerLock_AlreadyLocked_HasPIDAndTime
// - ロック解放後に再取得できる（stale lock なし）             → TestAcquireViewerLock_ReacquireAfterRelease
// - run/ ディレクトリが存在しなければ自動作成する             → TestAcquireViewerLock_CreatesRunDir

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/canpok1/vox-actor/internal/infra"
)

// parseLockFile はロックファイルの内容を pid と started_at にパースするテストヘルパー。
func parseLockFile(t *testing.T, path string) (pid int, startedAt time.Time) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read lock file: %v", err)
	}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		switch k {
		case "pid":
			pid, err = strconv.Atoi(v)
			if err != nil {
				t.Fatalf("invalid pid in lock file: %v", err)
			}
		case "started_at":
			startedAt, err = time.Parse(time.RFC3339, v)
			if err != nil {
				t.Fatalf("invalid started_at in lock file: %v", err)
			}
		}
	}
	return pid, startedAt
}

func TestAcquireViewerLock_WritesLockFile(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, "run", "viewer.lock")

	vl, err := infra.AcquireViewerLock(lockPath)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	defer vl.Release() //nolint:errcheck

	data, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("expected lock file to exist at %s: %v", lockPath, err)
	}
	if len(data) == 0 {
		t.Error("expected lock file to have content")
	}

	pid, startedAt := parseLockFile(t, lockPath)
	if pid != os.Getpid() {
		t.Errorf("expected pid=%d, got %d", os.Getpid(), pid)
	}
	if time.Since(startedAt) > 5*time.Second {
		t.Errorf("expected started_at to be recent, got %v", startedAt)
	}
}

func TestAcquireViewerLock_AlreadyLocked(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, "run", "viewer.lock")

	first, err := infra.AcquireViewerLock(lockPath)
	if err != nil {
		t.Fatalf("first lock: expected no error, got: %v", err)
	}
	defer first.Release() //nolint:errcheck

	_, err = infra.AcquireViewerLock(lockPath)
	if err == nil {
		t.Fatal("second lock: expected error, got nil")
	}

	var alreadyRunning *infra.ViewerAlreadyRunningError
	if !errors.As(err, &alreadyRunning) {
		t.Errorf("expected ViewerAlreadyRunningError, got: %T: %v", err, err)
	}
}

func TestAcquireViewerLock_AlreadyLocked_HasPIDAndTime(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, "run", "viewer.lock")

	before := time.Now().Truncate(time.Second)

	first, err := infra.AcquireViewerLock(lockPath)
	if err != nil {
		t.Fatalf("first lock: expected no error, got: %v", err)
	}
	defer first.Release() //nolint:errcheck

	_, err = infra.AcquireViewerLock(lockPath)
	if err == nil {
		t.Fatal("second lock: expected error, got nil")
	}

	var alreadyRunning *infra.ViewerAlreadyRunningError
	if !errors.As(err, &alreadyRunning) {
		t.Fatalf("expected ViewerAlreadyRunningError, got: %T: %v", err, err)
	}

	if alreadyRunning.PID != os.Getpid() {
		t.Errorf("expected PID=%d, got %d", os.Getpid(), alreadyRunning.PID)
	}
	if alreadyRunning.StartedAt.Before(before) {
		t.Errorf("expected StartedAt >= %v, got %v", before, alreadyRunning.StartedAt)
	}
}

func TestAcquireViewerLock_ReacquireAfterRelease(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, "run", "viewer.lock")

	first, err := infra.AcquireViewerLock(lockPath)
	if err != nil {
		t.Fatalf("first lock: expected no error, got: %v", err)
	}
	if err := first.Release(); err != nil {
		t.Fatalf("release: expected no error, got: %v", err)
	}

	second, err := infra.AcquireViewerLock(lockPath)
	if err != nil {
		t.Fatalf("second lock after release: expected no error, got: %v", err)
	}
	defer second.Release() //nolint:errcheck
}

func TestAcquireViewerLock_CreatesRunDir(t *testing.T) {
	dir := t.TempDir()
	runDir := filepath.Join(dir, "run")
	lockPath := filepath.Join(runDir, "viewer.lock")

	if _, err := os.Stat(runDir); !os.IsNotExist(err) {
		t.Fatalf("expected run dir to not exist before test")
	}

	vl, err := infra.AcquireViewerLock(lockPath)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	defer vl.Release() //nolint:errcheck

	info, err := os.Stat(runDir)
	if err != nil {
		t.Fatalf("expected run dir to be created at %s: %v", runDir, err)
	}
	if !info.IsDir() {
		t.Errorf("expected %s to be a directory", runDir)
	}
}
