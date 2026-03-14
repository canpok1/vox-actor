package infra

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/canpok1/vox-actor/internal/app"
	"github.com/canpok1/vox-actor/internal/domain/entity"
)

// utf8BOM は UTF-8 のバイトオーダーマーク。
var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// supportedExts はディレクトリ読み込み時に対象とする拡張子。
var supportedExts = map[string]bool{
	".txt":  true,
	".json": true,
}

// jsonScript は .json ファイルの台本データを表す。
type jsonScript struct {
	Text            *string  `json:"text"`
	Speaker         *int     `json:"speaker"`
	SpeedScale      *float64 `json:"speedScale"`
	PitchScale      *float64 `json:"pitchScale"`
	IntonationScale *float64 `json:"intonationScale"`
}

// FileReader はファイルシステムから台本を読み込む。
// app.ScriptReader インターフェースを実装する。
type FileReader struct{}

var _ app.ScriptReader = (*FileReader)(nil)

// NewFileReader は FileReader を生成する。
func NewFileReader() *FileReader {
	return &FileReader{}
}

// Read は指定パスから台本を読み込む。
// パスがファイルの場合はそのファイルのみ、ディレクトリの場合は対象拡張子のファイルを辞書順で返す。
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
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".json" {
		return r.readJSONFile(path)
	}
	return r.readTextFile(path)
}

// readFileBytes はファイルを読み込み、BOM除去とUTF-8検証を行う。
func (r *FileReader) readFileBytes(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	data = bytes.TrimPrefix(data, utf8BOM)
	if !utf8.Valid(data) {
		return nil, fmt.Errorf("file is not valid UTF-8: %s", path)
	}
	return data, nil
}

func (r *FileReader) readTextFile(path string) ([]entity.Script, error) {
	data, err := r.readFileBytes(path)
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

func (r *FileReader) readJSONFile(path string) ([]entity.Script, error) {
	data, err := r.readFileBytes(path)
	if err != nil {
		return nil, err
	}

	var js jsonScript
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&js); err != nil {
		return nil, fmt.Errorf("invalid JSON in %s: %w", path, err)
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return nil, fmt.Errorf("invalid JSON in %s: trailing data", path)
	}

	if js.Text == nil {
		return nil, fmt.Errorf("missing required field 'text' in %s", path)
	}

	text := *js.Text
	return []entity.Script{
		{
			Path:            path,
			Text:            text,
			IsEmpty:         len(text) == 0,
			SpeakerID:       js.Speaker,
			SpeedScale:      js.SpeedScale,
			PitchScale:      js.PitchScale,
			IntonationScale: js.IntonationScale,
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
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if supportedExts[ext] {
			names = append(names, entry.Name())
		}
	}

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
