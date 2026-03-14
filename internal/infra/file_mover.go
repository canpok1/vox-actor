package infra

import (
	"os"
	"path/filepath"
)

// FileMover はファイルをdone/サブディレクトリに移動する。
type FileMover struct{}

// NewFileMover はFileMoverを生成する。
func NewFileMover() *FileMover {
	return &FileMover{}
}

// MoveToDone はファイルを親ディレクトリのdone/サブディレクトリに移動する。
// done/ディレクトリが存在しない場合は自動作成する。
func (m *FileMover) MoveToDone(path string) error {
	dir := filepath.Dir(path)
	doneDir := filepath.Join(dir, "done")

	if err := os.MkdirAll(doneDir, 0o755); err != nil {
		return err
	}

	dest := filepath.Join(doneDir, filepath.Base(path))
	return os.Rename(path, dest)
}
