package infra

import (
	"context"
	"errors"
	"io/fs"
	"log/slog"
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
	logger   *slog.Logger
}

var _ app.DirWatcher = (*PollingDirWatcher)(nil)

// PollingDirWatcherOption は PollingDirWatcher の生成時に指定するオプション。
type PollingDirWatcherOption func(*PollingDirWatcher)

// WithPollingDirWatcherLogger はロガーを設定するオプション。
// 指定しない場合はログを破棄する。
func WithPollingDirWatcherLogger(logger *slog.Logger) PollingDirWatcherOption {
	return func(w *PollingDirWatcher) {
		if logger != nil {
			w.logger = logger
		}
	}
}

// NewPollingDirWatcher はPollingDirWatcherを生成する。
func NewPollingDirWatcher(interval time.Duration, opts ...PollingDirWatcherOption) *PollingDirWatcher {
	w := &PollingDirWatcher{
		interval: interval,
		logger:   slog.New(slog.DiscardHandler),
	}
	for _, opt := range opts {
		opt(w)
	}
	return w
}

// Watch はディレクトリを監視し、新規ファイルのパスをチャネルで返す。
// 既知のファイル（done/含む）を除外し、辞書順で通知する。
// 監視中にディレクトリが削除された場合は自動再作成して監視を継続する。
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
				if handled := w.handleListError(ctx, dir, err, errCh); !handled {
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

// handleListError は listFiles のエラーを処理する。
// ENOENTの場合はディレクトリを自動再作成し、成功時はwarnログを出す。
// ENOENT以外、または再作成失敗時は errCh にエラーを送る。
// ctx.Done() でループを終了すべき場合は false を返す。
func (w *PollingDirWatcher) handleListError(ctx context.Context, dir string, err error, errCh chan<- error) bool {
	if errors.Is(err, fs.ErrNotExist) {
		if mkErr := os.MkdirAll(dir, 0o755); mkErr != nil {
			return sendErr(ctx, errCh, mkErr)
		}
		w.logger.Warn("watch directory was recreated", "path", dir)
		return true
	}
	return sendErr(ctx, errCh, err)
}

// sendErr は errCh にエラーを送信する。ctx.Done() が先に発生したら false を返す。
func sendErr(ctx context.Context, errCh chan<- error, err error) bool {
	select {
	case errCh <- err:
		return true
	case <-ctx.Done():
		return false
	}
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
