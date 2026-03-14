package infra

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FileMover はファイルをdone/サブディレクトリに移動する。
type FileMover struct{}

// NewFileMover はFileMoverを生成する。
func NewFileMover() *FileMover {
	return &FileMover{}
}

// MoveToDone はファイルを親ディレクトリのdone/サブディレクトリに移動する。
// done/ディレクトリが存在しない場合は自動作成する。
// 同名ファイルが既に存在する場合はタイムスタンプを付加して衝突を回避する。
func (m *FileMover) MoveToDone(path string) error {
	dir := filepath.Dir(path)
	doneDir := filepath.Join(dir, "done")

	if err := os.MkdirAll(doneDir, 0o755); err != nil {
		return err
	}

	base := filepath.Base(path)
	dest := filepath.Join(doneDir, base)

	// 既存ファイルがある場合はタイムスタンプを付加
	if _, err := os.Stat(dest); err == nil {
		ext := filepath.Ext(base)
		name := strings.TrimSuffix(base, ext)
		dest = filepath.Join(doneDir, fmt.Sprintf("%s_%d%s", name, time.Now().UnixNano(), ext))
	}

	return os.Rename(path, dest)
}

// Delete はファイルを削除する。
func (m *FileMover) Delete(path string) error {
	return os.Remove(path)
}
