package infra

import (
	"os"

	"github.com/canpok1/vox-actor/internal/domain/entity"
)

// FileReader はファイルシステムから台本を読み込む。
type FileReader struct{}

// NewFileReader は FileReader を生成する。
func NewFileReader() *FileReader {
	return &FileReader{}
}

// Read は指定パスから台本を読み込む。
func (r *FileReader) Read(path string) ([]entity.Script, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	text := string(data)

	return []entity.Script{
		{
			Path:    path,
			Text:    text,
			IsEmpty: len(text) == 0,
		},
	}, nil
}
