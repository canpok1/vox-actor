package infra

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gofrs/flock"
)

// ViewerAlreadyRunningError is returned when another viewer process is running.
type ViewerAlreadyRunningError struct {
	PID       int
	StartedAt time.Time
}

func (e *ViewerAlreadyRunningError) Error() string {
	return fmt.Sprintf("viewer は既に起動中です\n  PID: %d\n  起動時刻: %s",
		e.PID, e.StartedAt.Format(time.RFC3339))
}

// ViewerLock holds an acquired exclusive viewer lock.
type ViewerLock struct {
	lock *flock.Flock
}

// AcquireViewerLock creates the run directory, acquires an exclusive flock on
// lockPath, and writes the current process PID and start time to the file.
// Returns *ViewerAlreadyRunningError if the lock is held by another process.
func AcquireViewerLock(lockPath string) (*ViewerLock, error) {
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o755); err != nil {
		return nil, fmt.Errorf("failed to create lock directory: %w", err)
	}

	f := flock.New(lockPath)
	locked, err := f.TryLock()
	if err != nil {
		return nil, fmt.Errorf("failed to try lock: %w", err)
	}
	if !locked {
		pid, startedAt, readErr := readLockFile(lockPath)
		if readErr != nil {
			return nil, fmt.Errorf("viewer は既に起動中です (lock file unreadable: %w)", readErr)
		}
		return nil, &ViewerAlreadyRunningError{PID: pid, StartedAt: startedAt}
	}

	pid := os.Getpid()
	startedAt := time.Now()
	if writeErr := writeLockFile(lockPath, pid, startedAt); writeErr != nil {
		_ = f.Unlock()
		return nil, fmt.Errorf("failed to write lock file: %w", writeErr)
	}

	return &ViewerLock{lock: f}, nil
}

func (vl *ViewerLock) Release() error {
	return vl.lock.Unlock()
}

func readLockFile(path string) (int, time.Time, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, time.Time{}, err
	}

	var pid int
	var startedAt time.Time
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		switch k {
		case "pid":
			pid, err = strconv.Atoi(v)
			if err != nil {
				return 0, time.Time{}, fmt.Errorf("invalid pid: %w", err)
			}
		case "started_at":
			startedAt, err = time.Parse(time.RFC3339, v)
			if err != nil {
				return 0, time.Time{}, fmt.Errorf("invalid started_at: %w", err)
			}
		}
	}
	return pid, startedAt, nil
}

func writeLockFile(path string, pid int, startedAt time.Time) error {
	content := fmt.Sprintf("pid=%d\nstarted_at=%s\n", pid, startedAt.Format(time.RFC3339))
	return os.WriteFile(path, []byte(content), 0o644)
}
