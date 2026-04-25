package infra

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/canpok1/vox-actor/internal/app"
	"github.com/canpok1/vox-actor/internal/domain/entity"
)

// FileWriter は entity.Script をファイルへ書き出す。
// app.ScriptWriter インターフェースを実装する。
type FileWriter struct {
	logger *slog.Logger
}

var _ app.ScriptWriter = (*FileWriter)(nil)

// FileWriterOption は FileWriter の生成時に指定するオプション。
type FileWriterOption func(*FileWriter)

// WithFileWriterLogger は FileWriter のロガーを設定するオプション。
func WithFileWriterLogger(logger *slog.Logger) FileWriterOption {
	return func(w *FileWriter) {
		if logger != nil {
			w.logger = logger
		}
	}
}

// NewFileWriter は FileWriter を生成する。
func NewFileWriter(opts ...FileWriterOption) *FileWriter {
	w := &FileWriter{
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	for _, opt := range opts {
		opt(w)
	}
	return w
}

// Write は entity.Script を path に書き出す。
// 拡張子で書き出し形式を判定する: .json / .jsonl / その他は本文のみ。
// 既存ファイルがある場合は <name>_<UnixNano><ext> で連番を付与する。
// 親ディレクトリが存在しない場合はエラーを返す。
// 戻り値は実際の書き出し先パス。
func (w *FileWriter) Write(path string, script entity.Script) (string, error) {
	parent := filepath.Dir(path)
	info, err := os.Stat(parent)
	if err != nil {
		return "", fmt.Errorf("parent directory does not exist: %s: %w", parent, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("parent path is not a directory: %s", parent)
	}

	dest := w.resolveDest(path)

	ext := strings.ToLower(filepath.Ext(dest))
	switch ext {
	case ".json", ".jsonl":
		return dest, writeJSONLine(dest, script)
	default:
		w.warnUnsavedParams(dest, script)
		return dest, writeText(dest, script.Text)
	}
}

// resolveDest は出力先パスを返す。同名ファイルが既に存在する場合は <name>_<UnixNano><ext> を付与する。
func (w *FileWriter) resolveDest(path string) string {
	if _, err := os.Stat(path); err != nil {
		return path
	}
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	return filepath.Join(dir, fmt.Sprintf("%s_%d%s", name, time.Now().UnixNano(), ext))
}

// warnUnsavedParams は .txt / 未知拡張子の出力時に保存できないパラメータが指定されていれば WARN ログを出す。
func (w *FileWriter) warnUnsavedParams(path string, script entity.Script) {
	var dropped []string
	if script.SpeakerID != nil {
		dropped = append(dropped, "speaker")
	}
	if script.SpeedScale != nil {
		dropped = append(dropped, "speed")
	}
	if script.PitchScale != nil {
		dropped = append(dropped, "pitch")
	}
	if script.IntonationScale != nil {
		dropped = append(dropped, "intonation")
	}
	if len(dropped) == 0 {
		return
	}
	w.logger.Warn(
		"text-format output cannot store voice parameters",
		"path", path,
		"droppedParams", strings.Join(dropped, ","),
	)
}

// jsonScriptOut は .json / .jsonl 出力用の構造体。omitempty をポインタで実現する。
type jsonScriptOut struct {
	Text            string   `json:"text"`
	Speaker         *int     `json:"speaker,omitempty"`
	SpeedScale      *float64 `json:"speedScale,omitempty"`
	PitchScale      *float64 `json:"pitchScale,omitempty"`
	IntonationScale *float64 `json:"intonationScale,omitempty"`
}

func toJSONScript(script entity.Script) jsonScriptOut {
	return jsonScriptOut{
		Text:            script.Text,
		Speaker:         script.SpeakerID,
		SpeedScale:      script.SpeedScale,
		PitchScale:      script.PitchScale,
		IntonationScale: script.IntonationScale,
	}
}

// writeJSONLine は1行JSON（末尾改行付き）を書き出す。
// .json / .jsonl で同じ実装。.jsonl が将来複数行追記をサポートする場合は分岐させる。
func writeJSONLine(path string, script entity.Script) error {
	data, err := json.Marshal(toJSONScript(script))
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func writeText(path string, text string) error {
	return os.WriteFile(path, []byte(text), 0o644)
}
