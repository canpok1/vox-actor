package infra

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/canpok1/vox-actor/internal/app"
)

// PollInterval はディレクトリのポーリング間隔。
const PollInterval = 1 * time.Second

// PollingDirWatcher はポーリングベースのディレクトリウォッチャー。
type PollingDirWatcher struct {
	interval time.Duration
}

var _ app.DirWatcher = (*PollingDirWatcher)(nil)

// NewPollingDirWatcher はPollingDirWatcherを生成する。
func NewPollingDirWatcher(interval time.Duration) *PollingDirWatcher {
	return &PollingDirWatcher{interval: interval}
}

// Watch はディレクトリを監視し、新規ファイルのパスをチャネルで返す。
// 既知のファイル（done/含む）を除外し、辞書順で通知する。
// コンテキストがキャンセルされると監視を停止してチャネルをクローズする。
func (w *PollingDirWatcher) Watch(ctx context.Context, dir string) (<-chan string, <-chan error) {
	fileCh := make(chan string)
	errCh := make(chan error, 1)

	go func() {
		defer close(fileCh)
		defer close(errCh)

		seen := make(map[string]bool)

		for {
			files, err := w.listFiles(dir)
			if err != nil {
				select {
				case errCh <- err:
				case <-ctx.Done():
					return
				}
				continue
			}

			// seenマップからディレクトリに存在しなくなったファイルを削除
			currentFiles := make(map[string]bool, len(files))
			for _, f := range files {
				currentFiles[f] = true
			}
			for f := range seen {
				if !currentFiles[f] {
					delete(seen, f)
				}
			}

			for _, f := range files {
				if seen[f] {
					continue
				}
				seen[f] = true
				select {
				case fileCh <- f:
				case <-ctx.Done():
					return
				}
			}

			select {
			case <-ctx.Done():
				return
			case <-time.After(w.interval):
			}
		}
	}()

	return fileCh, errCh
}

// listFiles はディレクトリ内のファイルを辞書順でリストする。
// done/サブディレクトリ内のファイルは除外する。
func (w *PollingDirWatcher) listFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		files = append(files, filepath.Join(dir, entry.Name()))
	}
	sort.Strings(files)
	return files, nil
}
