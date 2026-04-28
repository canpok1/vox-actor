package infra_test

// テストリスト: FileWriter
//
// 受け入れ条件 vs テスト関数のマッピング:
// - 拡張子 .json で text のみ保存          → TestFileWriter_Write_JSON_TextOnly
// - 拡張子 .json で全パラメータ保存          → TestFileWriter_Write_JSON_AllParams
// - 拡張子 .json で nil フィールドは省略     → TestFileWriter_Write_JSON_OmitNilFields
// - 拡張子 .jsonl は1行JSONで書き出される    → TestFileWriter_Write_JSONL_SingleLine
// - 拡張子 .txt は本文のみ書き出される        → TestFileWriter_Write_Txt_TextOnly
// - 未知拡張子は本文のみ書き出される           → TestFileWriter_Write_UnknownExt_TextOnly
// - .txt + speaker指定 で WARN ログ          → TestFileWriter_Write_Txt_WithSpeaker_LogsWarn
// - .txt で全パラメータnilなら WARN なし      → TestFileWriter_Write_Txt_NoParams_NoWarn
// - 既存ファイルがある場合は末尾に追記される    → TestFileWriter_Write_ExistingFile_Appends
// - 既存ファイルがある場合に別ファイルを作らない  → TestFileWriter_Write_ExistingFile_NoSuffix
// - 親ディレクトリ未存在時はエラー             → TestFileWriter_Write_MissingParentDir_ReturnsError

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/canpok1/vox-actor/internal/domain/entity"
	"github.com/canpok1/vox-actor/internal/infra"
)

func ptrInt(v int) *int             { return &v }
func ptrFloat64(v float64) *float64 { return &v }

func TestFileWriter_Write_JSON_TextOnly(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dest := filepath.Join(dir, "out.json")

	w := infra.NewFileWriter()
	written, err := w.Write(dest, entity.Script{Text: "こんにちは"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if written != dest {
		t.Errorf("expected written=%q, got %q", dest, written)
	}

	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("invalid JSON written: %v\n%s", err, string(data))
	}
	if got["text"] != "こんにちは" {
		t.Errorf("expected text=こんにちは, got %v", got["text"])
	}
	if _, exists := got["speaker"]; exists {
		t.Errorf("expected speaker omitted, got %v", got["speaker"])
	}
	if _, exists := got["speedScale"]; exists {
		t.Errorf("expected speedScale omitted, got %v", got["speedScale"])
	}
	if _, exists := got["pitchScale"]; exists {
		t.Errorf("expected pitchScale omitted, got %v", got["pitchScale"])
	}
	if _, exists := got["intonationScale"]; exists {
		t.Errorf("expected intonationScale omitted, got %v", got["intonationScale"])
	}
}

func TestFileWriter_Write_JSON_AllParams(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dest := filepath.Join(dir, "out.json")

	w := infra.NewFileWriter()
	script := entity.Script{
		Text:            "感情込めて",
		SpeakerID:       ptrInt(5),
		SpeedScale:      ptrFloat64(1.5),
		PitchScale:      ptrFloat64(0.1),
		IntonationScale: ptrFloat64(1.8),
	}
	if _, err := w.Write(dest, script); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if got["text"] != "感情込めて" {
		t.Errorf("text mismatch: %v", got["text"])
	}
	if got["speaker"].(float64) != 5 {
		t.Errorf("speaker mismatch: %v", got["speaker"])
	}
	if got["speedScale"].(float64) != 1.5 {
		t.Errorf("speedScale mismatch: %v", got["speedScale"])
	}
	if got["pitchScale"].(float64) != 0.1 {
		t.Errorf("pitchScale mismatch: %v", got["pitchScale"])
	}
	if got["intonationScale"].(float64) != 1.8 {
		t.Errorf("intonationScale mismatch: %v", got["intonationScale"])
	}
}

func TestFileWriter_Write_JSON_OmitNilFields(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dest := filepath.Join(dir, "partial.json")

	w := infra.NewFileWriter()
	script := entity.Script{
		Text:      "部分指定",
		SpeakerID: ptrInt(2),
	}
	if _, err := w.Write(dest, script); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}

	contents := string(data)
	if strings.Contains(contents, "speedScale") {
		t.Errorf("expected speedScale omitted, got: %s", contents)
	}
	if strings.Contains(contents, "pitchScale") {
		t.Errorf("expected pitchScale omitted, got: %s", contents)
	}
	if strings.Contains(contents, "intonationScale") {
		t.Errorf("expected intonationScale omitted, got: %s", contents)
	}
	if !strings.Contains(contents, `"speaker":2`) && !strings.Contains(contents, `"speaker": 2`) {
		t.Errorf("expected speaker:2 included, got: %s", contents)
	}
}

func TestFileWriter_Write_JSONL_SingleLine(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dest := filepath.Join(dir, "out.jsonl")

	w := infra.NewFileWriter()
	script := entity.Script{
		Text:      "JSONL出力",
		SpeakerID: ptrInt(3),
	}
	if _, err := w.Write(dest, script); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}

	// 末尾改行を許容しつつ、ファイル全体は1行JSONであること
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected single line, got %d lines: %q", len(lines), string(data))
	}

	var got map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &got); err != nil {
		t.Fatalf("invalid JSON line: %v\n%s", err, lines[0])
	}
	if got["text"] != "JSONL出力" {
		t.Errorf("text mismatch: %v", got["text"])
	}
	if got["speaker"].(float64) != 3 {
		t.Errorf("speaker mismatch: %v", got["speaker"])
	}
}

func TestFileWriter_Write_Txt_TextOnly(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dest := filepath.Join(dir, "out.txt")

	w := infra.NewFileWriter()
	if _, err := w.Write(dest, entity.Script{Text: "プレーンテキスト"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if string(data) != "プレーンテキスト" {
		t.Errorf("expected exact text, got: %q", string(data))
	}
}

func TestFileWriter_Write_UnknownExt_ReturnsError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dest := filepath.Join(dir, "out.foo")

	w := infra.NewFileWriter()
	_, err := w.Write(dest, entity.Script{Text: "未知拡張子"})
	if err == nil {
		t.Fatal("expected error for unknown extension, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported output extension") {
		t.Errorf("expected 'unsupported output extension' in error, got: %v", err)
	}
}

func TestFileWriter_Write_Txt_WithSpeaker_LogsWarn(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	dir := t.TempDir()
	dest := filepath.Join(dir, "out.txt")

	w := infra.NewFileWriter(infra.WithFileWriterLogger(logger))
	script := entity.Script{
		Text:      "本文",
		SpeakerID: ptrInt(5),
	}
	if _, err := w.Write(dest, script); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// WARN レベルのログが1行以上出ていること（JSONハンドラなので level=WARN を含む行を検出）
	warnLines := filterLines(buf.String(), `"level":"WARN"`)
	if len(warnLines) == 0 {
		t.Fatalf("expected at least one WARN log, got: %s", buf.String())
	}
	if !strings.Contains(warnLines[0], "speaker") {
		t.Errorf("expected WARN log to mention 'speaker', got: %s", warnLines[0])
	}
}

func TestFileWriter_Write_Txt_NoParams_NoWarn(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	dir := t.TempDir()
	dest := filepath.Join(dir, "out.txt")

	w := infra.NewFileWriter(infra.WithFileWriterLogger(logger))
	if _, err := w.Write(dest, entity.Script{Text: "本文のみ"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if warnLines := filterLines(buf.String(), `"level":"WARN"`); len(warnLines) > 0 {
		t.Errorf("expected no WARN log, got: %v", warnLines)
	}
}

// filterLines は output 内で needle を含む行のみを返す。
func filterLines(output, needle string) []string {
	var matched []string
	for line := range strings.SplitSeq(strings.TrimRight(output, "\n"), "\n") {
		if line == "" {
			continue
		}
		if strings.Contains(line, needle) {
			matched = append(matched, line)
		}
	}
	return matched
}

func TestFileWriter_Write_ExistingFile_Appends(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dest := filepath.Join(dir, "out.jsonl")

	w := infra.NewFileWriter()

	// 1回目
	written1, err := w.Write(dest, entity.Script{Text: "A"})
	if err != nil {
		t.Fatalf("first Write failed: %v", err)
	}
	if written1 != dest {
		t.Errorf("expected written=%q, got %q", dest, written1)
	}

	// 2回目（既存ファイルへの追記）
	written2, err := w.Write(dest, entity.Script{Text: "B"})
	if err != nil {
		t.Fatalf("second Write failed: %v", err)
	}
	if written2 != dest {
		t.Errorf("expected written=%q, got %q", dest, written2)
	}

	// ファイルが1つだけ存在すること
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}
	if len(entries) != 1 {
		names := make([]string, len(entries))
		for i, e := range entries {
			names[i] = e.Name()
		}
		t.Errorf("expected 1 file, got %d: %v", len(entries), names)
	}

	// 2行記録されていること
	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %q", len(lines), string(data))
	}

	var got1 map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &got1); err != nil {
		t.Fatalf("invalid JSON line 0: %v", err)
	}
	if got1["text"] != "A" {
		t.Errorf("line 0 text mismatch: %v", got1["text"])
	}

	var got2 map[string]any
	if err := json.Unmarshal([]byte(lines[1]), &got2); err != nil {
		t.Fatalf("invalid JSON line 1: %v", err)
	}
	if got2["text"] != "B" {
		t.Errorf("line 1 text mismatch: %v", got2["text"])
	}
}

func TestFileWriter_Write_ExistingFile_NoSuffix(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dest := filepath.Join(dir, "out.jsonl")
	if err := os.WriteFile(dest, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	w := infra.NewFileWriter()
	written, err := w.Write(dest, entity.Script{Text: "新規"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 別ファイルが作られず、同一パスが返ること
	if written != dest {
		t.Errorf("expected same path %q, got %q", dest, written)
	}

	// ディレクトリに1ファイルだけ存在すること
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}
	if len(entries) != 1 {
		names := make([]string, len(entries))
		for i, e := range entries {
			names[i] = e.Name()
		}
		t.Errorf("expected 1 file, got %d: %v", len(entries), names)
	}
}

func TestFileWriter_Write_MissingParentDir_ReturnsError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dest := filepath.Join(dir, "missing", "out.txt")

	w := infra.NewFileWriter()
	if _, err := w.Write(dest, entity.Script{Text: "本文"}); err == nil {
		t.Fatal("expected error for missing parent directory, got nil")
	}
}

// ディレクトリ指定時のテスト

func TestFileWriter_Write_DirectoryTarget_CreatesAutoNamedFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	outDir := filepath.Join(dir, "out")
	if err := os.Mkdir(outDir, 0o755); err != nil {
		t.Fatalf("failed to create output dir: %v", err)
	}

	w := infra.NewFileWriter()
	written, err := w.Write(outDir, entity.Script{
		Text:       "ディレクトリ指定",
		SpeakerID:  ptrInt(2),
		SpeedScale: ptrFloat64(1.1),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 出力先は outDir 配下、かつ _say.json で終わる
	if !strings.HasPrefix(written, outDir) {
		t.Errorf("expected written path under %s, got %s", outDir, written)
	}
	if !strings.HasSuffix(written, "_say.json") {
		t.Errorf("expected filename ending with _say.json, got %s", filepath.Base(written))
	}

	// ファイルが実際に存在し、JSONで読める
	data, err := os.ReadFile(written)
	if err != nil {
		t.Fatalf("failed to read written file: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("invalid JSON written: %v", err)
	}
	if got["text"] != "ディレクトリ指定" {
		t.Errorf("text mismatch: %v", got["text"])
	}
	if got["speaker"].(float64) != 2 {
		t.Errorf("speaker mismatch: %v", got["speaker"])
	}
}

func TestFileWriter_Write_DirectoryTargetTrailingSlash_CreatesDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	outDir := filepath.Join(dir, "newdir") + string(os.PathSeparator)

	w := infra.NewFileWriter()
	written, err := w.Write(outDir, entity.Script{Text: "末尾スラッシュ指定"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// ディレクトリが作成され、その下にファイルが生成される
	if !strings.HasSuffix(written, "_say.json") {
		t.Errorf("expected filename ending with _say.json, got %s", filepath.Base(written))
	}

	data, err := os.ReadFile(written)
	if err != nil {
		t.Fatalf("failed to read written file: %v", err)
	}
	if !strings.Contains(string(data), "末尾スラッシュ指定") {
		t.Errorf("text not found in file: %s", string(data))
	}
}

// ファイル拡張子検証のテスト

func TestFileWriter_Write_NoExtension_ReturnsError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dest := filepath.Join(dir, "noext")

	w := infra.NewFileWriter()
	_, err := w.Write(dest, entity.Script{Text: "本文"})
	if err == nil {
		t.Fatal("expected error for no extension, got nil")
	}
	if !strings.Contains(err.Error(), "no extension") {
		t.Errorf("expected 'no extension' in error message, got: %v", err)
	}
}

func TestFileWriter_Write_UnsupportedExtension_ReturnsError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dest := filepath.Join(dir, "out.csv")

	w := infra.NewFileWriter()
	_, err := w.Write(dest, entity.Script{Text: "本文"})
	if err == nil {
		t.Fatal("expected error for unsupported extension, got nil")
	}
	errMsg := err.Error()
	if !strings.Contains(errMsg, "unsupported output extension") && !strings.Contains(errMsg, ".csv") {
		t.Errorf("expected error message mentioning unsupported extension, got: %v", err)
	}
}

func TestFileWriter_Write_SupportedExtensions_AllWork(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	exts := []string{".json", ".jsonl", ".txt"}

	for _, ext := range exts {
		t.Run(ext, func(t *testing.T) {
			dest := filepath.Join(dir, "out"+ext)
			w := infra.NewFileWriter()
			_, err := w.Write(dest, entity.Script{Text: "テスト"})
			if err != nil {
				t.Fatalf("unexpected error for %s: %v", ext, err)
			}
			if _, err := os.Stat(dest); err != nil {
				t.Fatalf("file not created: %v", err)
			}
		})
	}
}
