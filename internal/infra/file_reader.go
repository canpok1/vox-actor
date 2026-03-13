package infra

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/canpok1/vox-actor/internal/domain/entity"
)

// utf8BOM は UTF-8 のバイトオーダーマーク。
var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// FileReader はファイルシステムから台本を読み込む。
type FileReader struct{}

// NewFileReader は FileReader を生成する。
func NewFileReader() *FileReader {
	return &FileReader{}
}

// Read は指定パスから台本を読み込む。
// パスがファイルの場合はそのファイルのみ、ディレクトリの場合は.txtファイルを辞書順で返す。
func (r *FileReader) Read(path string) ([]entity.Script, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if !info.IsDir() {
		return r.readFile(path)
	}

	return r.readDirectory(path)
}

func (r *FileReader) readFile(path string) ([]entity.Script, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	data = bytes.TrimPrefix(data, utf8BOM)
	text := string(data)

	return []entity.Script{
		{
			Path:    path,
			Text:    text,
			IsEmpty: len(text) == 0,
		},
	}, nil
}

func (r *FileReader) readDirectory(dir string) ([]entity.Script, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.ToLower(filepath.Ext(entry.Name())) == ".txt" {
			names = append(names, entry.Name())
		}
	}

	sort.Strings(names)

	var scripts []entity.Script
	for _, name := range names {
		result, err := r.readFile(filepath.Join(dir, name))
		if err != nil {
			return nil, err
		}
		scripts = append(scripts, result...)
	}

	return scripts, nil
}
