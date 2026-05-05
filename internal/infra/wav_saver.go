package infra

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/canpok1/vox-actor/internal/app"
)

// WavSaver は WAV データをファイルに保存する。
// app.WavSaver インターフェースを実装する。
type WavSaver struct{}

var _ app.WavSaver = (*WavSaver)(nil)

// NewWavSaver は WavSaver を生成する。
func NewWavSaver() *WavSaver {
	return &WavSaver{}
}

// Save は wavData を path に保存する。
// 親ディレクトリが存在しない場合は自動作成する。
// 既存ファイルがある場合は上書きする。
func (s *WavSaver) Save(path string, wavData []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}
	if err := os.WriteFile(path, wavData, 0o644); err != nil {
		return fmt.Errorf("failed to write wav to %s: %w", path, err)
	}
	return nil
}
