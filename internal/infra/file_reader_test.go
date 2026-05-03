package infra_test

// テストリスト: FileReader
//
// DONE: 正常系: 単一ファイルパスを指定した場合、そのファイルの内容をScriptとして返す
// DONE: 正常系: ディレクトリパスを指定した場合、.txtファイルを辞書順で返す
// DONE: 正常系: 空ファイルを読み込んだ場合、IsEmptyがtrueのScriptを返す
// DONE: 正常系: UTF-8 BOM付きファイルを読み込んだ場合、BOMを除去して内容を返す
// DONE: 正常系: ディレクトリ内に.txt/.json以外のファイルがある場合、対象ファイルのみ返す
// DONE: 正常系: ディレクトリ内に対象ファイルがない場合、空スライスを返す
// DONE: 異常系: 存在しないパスを指定した場合、エラーを返す
// DONE: 異常系: ファイルが.txt拡張子でない場合でも単一ファイル指定なら読み込む
// DONE: 異常系: 不正なUTF-8バイト列を含むファイルの場合、エラーを返す
//
// JSON単一台本モード:
// DONE: 正常系: .jsonファイルを単一ファイルとして指定した場合、感情制御パラメータ付きScriptを返す
// DONE: 正常系: .jsonファイルでtextのみ指定した場合、任意パラメータはnilのScriptを返す
// DONE: 正常系: .jsonファイルで全パラメータ指定した場合、全て反映されたScriptを返す
// DONE: 正常系: ディレクトリ読み込み時に.jsonファイルも対象になる
// DONE: 異常系: .jsonファイルが不正なJSONの場合、エラーを返す
// DONE: 異常系: .jsonファイルにtextフィールドがない場合、エラーを返す
// DONE: 異常系: .jsonファイルに未知フィールドがある場合、エラーを返す
//
// JSONL朗読劇モード:
// DONE: 正常系: .jsonlファイルを行ごとに解析し複数のScriptを返す
// DONE: 正常系: .jsonlファイルでtextのみ指定した場合、任意パラメータはnilのScriptを返す
// DONE: 正常系: .jsonlファイルで全パラメータ指定した場合、全て反映されたScriptを返す
// DONE: 正常系: .jsonlファイルの空行はスキップされる
// DONE: 正常系: ディレクトリ読み込み時に.jsonlファイルも対象になる
// DONE: 正常系: .jsonlファイルが1行のみでも正しく解析される
// DONE: 異常系: 不正なJSON行はスキップされる
// DONE: 異常系: textフィールドがない行はスキップされる
// DONE: 異常系: 全行が不正な場合は空スライスを返す
// DONE: 異常系: trailing dataがある行はスキップされる

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/canpok1/vox-actor/internal/infra"
)

func TestFileReader_Read_SingleFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "hello.txt")
	if err := os.WriteFile(filePath, []byte("こんにちは"), 0o644); err != nil {
		t.Fatal(err)
	}

	reader := infra.NewFileReader()
	scripts, err := reader.Read(filePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(scripts) != 1 {
		t.Fatalf("expected 1 script, got %d", len(scripts))
	}

	if scripts[0].Path != filePath {
		t.Errorf("expected path %q, got %q", filePath, scripts[0].Path)
	}

	if scripts[0].Text != "こんにちは" {
		t.Errorf("expected text %q, got %q", "こんにちは", scripts[0].Text)
	}

	if scripts[0].IsEmpty {
		t.Error("expected IsEmpty to be false")
	}
}

func TestFileReader_Read_UTF8BOM(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "bom.txt")
	// UTF-8 BOM (0xEF, 0xBB, 0xBF) + テキスト
	content := append([]byte{0xEF, 0xBB, 0xBF}, []byte("BOM付きテキスト")...)
	if err := os.WriteFile(filePath, content, 0o644); err != nil {
		t.Fatal(err)
	}

	reader := infra.NewFileReader()
	scripts, err := reader.Read(filePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(scripts) != 1 {
		t.Fatalf("expected 1 script, got %d", len(scripts))
	}

	if scripts[0].Text != "BOM付きテキスト" {
		t.Errorf("expected text %q, got %q", "BOM付きテキスト", scripts[0].Text)
	}
}

func TestFileReader_Read_EmptyFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "empty.txt")
	if err := os.WriteFile(filePath, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	reader := infra.NewFileReader()
	scripts, err := reader.Read(filePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(scripts) != 1 {
		t.Fatalf("expected 1 script, got %d", len(scripts))
	}

	if !scripts[0].IsEmpty {
		t.Error("expected IsEmpty to be true")
	}

	if scripts[0].Text != "" {
		t.Errorf("expected empty text, got %q", scripts[0].Text)
	}
}

func TestFileReader_Read_Directory(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	// 辞書順でb, a だが結果はa, bの順で返る
	if err := os.WriteFile(filepath.Join(dir, "b.txt"), []byte("2番目"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("1番目"), 0o644); err != nil {
		t.Fatal(err)
	}

	reader := infra.NewFileReader()
	scripts, err := reader.Read(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(scripts) != 2 {
		t.Fatalf("expected 2 scripts, got %d", len(scripts))
	}

	if scripts[0].Text != "1番目" {
		t.Errorf("expected first text %q, got %q", "1番目", scripts[0].Text)
	}

	if scripts[1].Text != "2番目" {
		t.Errorf("expected second text %q, got %q", "2番目", scripts[1].Text)
	}

	expectedPath0 := filepath.Join(dir, "a.txt")
	if scripts[0].Path != expectedPath0 {
		t.Errorf("expected path %q, got %q", expectedPath0, scripts[0].Path)
	}
}

func TestFileReader_Read_DirectoryFiltersSupportedOnly(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "script.txt"), []byte("台本"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "image.png"), []byte("PNG data"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "readme.md"), []byte("# README"), 0o644); err != nil {
		t.Fatal(err)
	}

	reader := infra.NewFileReader()
	scripts, err := reader.Read(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(scripts) != 1 {
		t.Fatalf("expected 1 script, got %d", len(scripts))
	}

	if scripts[0].Text != "台本" {
		t.Errorf("expected text %q, got %q", "台本", scripts[0].Text)
	}
}

func TestFileReader_Read_DirectoryNoTxtFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "readme.md"), []byte("# README"), 0o644); err != nil {
		t.Fatal(err)
	}

	reader := infra.NewFileReader()
	scripts, err := reader.Read(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(scripts) != 0 {
		t.Fatalf("expected 0 scripts, got %d", len(scripts))
	}
}

func TestFileReader_Read_SingleFileNonTxtExtension(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "script.md")
	if err := os.WriteFile(filePath, []byte("マークダウン"), 0o644); err != nil {
		t.Fatal(err)
	}

	reader := infra.NewFileReader()
	scripts, err := reader.Read(filePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(scripts) != 1 {
		t.Fatalf("expected 1 script, got %d", len(scripts))
	}

	if scripts[0].Text != "マークダウン" {
		t.Errorf("expected text %q, got %q", "マークダウン", scripts[0].Text)
	}
}

func TestFileReader_Read_InvalidUTF8(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "invalid.txt")
	// 不正なUTF-8バイト列
	if err := os.WriteFile(filePath, []byte{0xFF, 0xFE, 0x80}, 0o644); err != nil {
		t.Fatal(err)
	}

	reader := infra.NewFileReader()
	_, err := reader.Read(filePath)
	if err == nil {
		t.Fatal("expected error for invalid UTF-8, got nil")
	}
}

func TestFileReader_Read_NotExistPath(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	missingPath := filepath.Join(dir, "no-such-file.txt")

	reader := infra.NewFileReader()
	_, err := reader.Read(missingPath)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// JSONL朗読劇モード:
// DONE: 正常系: .jsonlファイルで複数行のJSONを解析し、複数のScriptを返す
// DONE: 正常系: .jsonlファイルでtextのみの行はオプショナルパラメータがnilになる
// DONE: 正常系: .jsonlファイルで全パラメータ指定の行は全て反映される
// DONE: 正常系: .jsonlファイルで空行はスキップされる
// DONE: 正常系: ディレクトリ読み込み時に.jsonlファイルも対象になる
// DONE: 正常系: .jsonlファイルで1行だけの場合、1つのScriptを返す
// DONE: 異常系: .jsonlファイルで不正なJSON行はログ出力してスキップされる
// DONE: 異常系: .jsonlファイルでtextフィールドがない行はログ出力してスキップされる
// DONE: 異常系: .jsonlファイルで全行が不正な場合、空スライスを返す

// --- JSON単一台本モード テスト ---

func TestFileReader_Read_JSONFile_TextOnly(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "script.json")
	content := `{"text": "こんにちは"}`
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	reader := infra.NewFileReader()
	scripts, err := reader.Read(filePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(scripts) != 1 {
		t.Fatalf("expected 1 script, got %d", len(scripts))
	}

	s := scripts[0]
	if s.Path != filePath {
		t.Errorf("expected path %q, got %q", filePath, s.Path)
	}
	if s.Text != "こんにちは" {
		t.Errorf("expected text %q, got %q", "こんにちは", s.Text)
	}
	if s.IsEmpty {
		t.Error("expected IsEmpty to be false")
	}
	if s.SpeakerID != nil {
		t.Errorf("expected SpeakerID nil, got %v", *s.SpeakerID)
	}
	if s.SpeedScale != nil {
		t.Errorf("expected SpeedScale nil, got %v", *s.SpeedScale)
	}
	if s.PitchScale != nil {
		t.Errorf("expected PitchScale nil, got %v", *s.PitchScale)
	}
	if s.IntonationScale != nil {
		t.Errorf("expected IntonationScale nil, got %v", *s.IntonationScale)
	}
}

func TestFileReader_Read_JSONFile_AllParams(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "script.json")
	content := `{"text": "感情込めて", "speaker": 5, "speed": 1.5, "pitch": 0.1, "intonation": 1.8}`
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	reader := infra.NewFileReader()
	scripts, err := reader.Read(filePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(scripts) != 1 {
		t.Fatalf("expected 1 script, got %d", len(scripts))
	}

	s := scripts[0]
	if s.Text != "感情込めて" {
		t.Errorf("expected text %q, got %q", "感情込めて", s.Text)
	}
	if s.SpeakerID == nil || *s.SpeakerID != 5 {
		t.Errorf("expected SpeakerID 5, got %v", s.SpeakerID)
	}
	if s.SpeedScale == nil || *s.SpeedScale != 1.5 {
		t.Errorf("expected SpeedScale 1.5, got %v", s.SpeedScale)
	}
	if s.PitchScale == nil || *s.PitchScale != 0.1 {
		t.Errorf("expected PitchScale 0.1, got %v", s.PitchScale)
	}
	if s.IntonationScale == nil || *s.IntonationScale != 1.8 {
		t.Errorf("expected IntonationScale 1.8, got %v", s.IntonationScale)
	}
}

func TestFileReader_Read_JSONFile_InvalidJSON(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(filePath, []byte("{invalid json"), 0o644); err != nil {
		t.Fatal(err)
	}

	reader := infra.NewFileReader()
	_, err := reader.Read(filePath)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestFileReader_Read_JSONFile_MissingText(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "notext.json")
	content := `{"speaker": 5, "speed": 1.0}`
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	reader := infra.NewFileReader()
	_, err := reader.Read(filePath)
	if err == nil {
		t.Fatal("expected error for missing text field, got nil")
	}
}

func TestFileReader_Read_DirectoryIncludesJSON(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("テキスト"), 0o644); err != nil {
		t.Fatal(err)
	}
	jsonContent := `{"text": "JSON台本"}`
	if err := os.WriteFile(filepath.Join(dir, "b.json"), []byte(jsonContent), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "c.png"), []byte("PNG"), 0o644); err != nil {
		t.Fatal(err)
	}

	reader := infra.NewFileReader()
	scripts, err := reader.Read(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(scripts) != 2 {
		t.Fatalf("expected 2 scripts, got %d", len(scripts))
	}

	// 辞書順: a.txt, b.json
	if scripts[0].Text != "テキスト" {
		t.Errorf("expected first text %q, got %q", "テキスト", scripts[0].Text)
	}
	if scripts[1].Text != "JSON台本" {
		t.Errorf("expected second text %q, got %q", "JSON台本", scripts[1].Text)
	}
}

func TestFileReader_Read_JSONFile_UnknownField(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "unknown.json")
	content := `{"text": "こんにちは", "unknownField": 123}`
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	reader := infra.NewFileReader()
	_, err := reader.Read(filePath)
	if err == nil {
		t.Fatal("expected error for unknown field in JSON, got nil")
	}
}

// --- JSONL朗読劇モード テスト ---

func TestFileReader_Read_JSONLFile_MultipleLines(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "script.jsonl")
	content := `{"text": "1行目"}
{"text": "2行目"}
{"text": "3行目"}`
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	reader := infra.NewFileReader()
	scripts, err := reader.Read(filePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(scripts) != 3 {
		t.Fatalf("expected 3 scripts, got %d", len(scripts))
	}

	expectedTexts := []string{"1行目", "2行目", "3行目"}
	for i, expected := range expectedTexts {
		if scripts[i].Text != expected {
			t.Errorf("scripts[%d].Text: expected %q, got %q", i, expected, scripts[i].Text)
		}
		if scripts[i].Path != filePath {
			t.Errorf("scripts[%d].Path: expected %q, got %q", i, filePath, scripts[i].Path)
		}
		if scripts[i].IsEmpty {
			t.Errorf("scripts[%d].IsEmpty: expected false", i)
		}
	}
}

func TestFileReader_Read_JSONLFile_TextOnly(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "script.jsonl")
	content := `{"text": "こんにちは"}`
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	reader := infra.NewFileReader()
	scripts, err := reader.Read(filePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(scripts) != 1 {
		t.Fatalf("expected 1 script, got %d", len(scripts))
	}

	s := scripts[0]
	if s.Text != "こんにちは" {
		t.Errorf("expected text %q, got %q", "こんにちは", s.Text)
	}
	if s.SpeakerID != nil {
		t.Errorf("expected SpeakerID nil, got %v", *s.SpeakerID)
	}
	if s.SpeedScale != nil {
		t.Errorf("expected SpeedScale nil, got %v", *s.SpeedScale)
	}
	if s.PitchScale != nil {
		t.Errorf("expected PitchScale nil, got %v", *s.PitchScale)
	}
	if s.IntonationScale != nil {
		t.Errorf("expected IntonationScale nil, got %v", *s.IntonationScale)
	}
}

func TestFileReader_Read_JSONLFile_AllParams(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "script.jsonl")
	content := `{"text": "感情込めて", "speaker": 5, "speed": 1.5, "pitch": 0.1, "intonation": 1.8}`
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	reader := infra.NewFileReader()
	scripts, err := reader.Read(filePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(scripts) != 1 {
		t.Fatalf("expected 1 script, got %d", len(scripts))
	}

	s := scripts[0]
	if s.Text != "感情込めて" {
		t.Errorf("expected text %q, got %q", "感情込めて", s.Text)
	}
	if s.SpeakerID == nil || *s.SpeakerID != 5 {
		t.Errorf("expected SpeakerID 5, got %v", s.SpeakerID)
	}
	if s.SpeedScale == nil || *s.SpeedScale != 1.5 {
		t.Errorf("expected SpeedScale 1.5, got %v", s.SpeedScale)
	}
	if s.PitchScale == nil || *s.PitchScale != 0.1 {
		t.Errorf("expected PitchScale 0.1, got %v", s.PitchScale)
	}
	if s.IntonationScale == nil || *s.IntonationScale != 1.8 {
		t.Errorf("expected IntonationScale 1.8, got %v", s.IntonationScale)
	}
}

func TestFileReader_Read_JSONLFile_EmptyLines(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "script.jsonl")
	content := `{"text": "1行目"}

{"text": "2行目"}

{"text": "3行目"}`
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	reader := infra.NewFileReader()
	scripts, err := reader.Read(filePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(scripts) != 3 {
		t.Fatalf("expected 3 scripts, got %d", len(scripts))
	}

	expectedTexts := []string{"1行目", "2行目", "3行目"}
	for i, expected := range expectedTexts {
		if scripts[i].Text != expected {
			t.Errorf("scripts[%d].Text: expected %q, got %q", i, expected, scripts[i].Text)
		}
	}
}

func TestFileReader_Read_DirectoryIncludesJSONL(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("テキスト"), 0o644); err != nil {
		t.Fatal(err)
	}
	jsonlContent := "{\"text\": \"JSONL1行目\"}\n{\"text\": \"JSONL2行目\"}"
	if err := os.WriteFile(filepath.Join(dir, "b.jsonl"), []byte(jsonlContent), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "c.png"), []byte("PNG"), 0o644); err != nil {
		t.Fatal(err)
	}

	reader := infra.NewFileReader()
	scripts, err := reader.Read(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(scripts) != 3 {
		t.Fatalf("expected 3 scripts, got %d", len(scripts))
	}

	// 辞書順: a.txt, b.jsonl (2行)
	if scripts[0].Text != "テキスト" {
		t.Errorf("expected first text %q, got %q", "テキスト", scripts[0].Text)
	}
	if scripts[1].Text != "JSONL1行目" {
		t.Errorf("expected second text %q, got %q", "JSONL1行目", scripts[1].Text)
	}
	if scripts[2].Text != "JSONL2行目" {
		t.Errorf("expected third text %q, got %q", "JSONL2行目", scripts[2].Text)
	}
}

func TestFileReader_Read_JSONLFile_SingleLine(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "script.jsonl")
	content := `{"text": "1行だけ"}`
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	reader := infra.NewFileReader()
	scripts, err := reader.Read(filePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(scripts) != 1 {
		t.Fatalf("expected 1 script, got %d", len(scripts))
	}

	if scripts[0].Text != "1行だけ" {
		t.Errorf("expected text %q, got %q", "1行だけ", scripts[0].Text)
	}
}

func TestFileReader_Read_JSONLFile_InvalidLineSkipped(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "script.jsonl")
	content := `{"text": "valid1"}
{invalid json}
{"text": "valid2"}`
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	reader := infra.NewFileReader()
	scripts, err := reader.Read(filePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(scripts) != 2 {
		t.Fatalf("expected 2 scripts (invalid line skipped), got %d", len(scripts))
	}

	if scripts[0].Text != "valid1" {
		t.Errorf("expected text %q, got %q", "valid1", scripts[0].Text)
	}
	if scripts[1].Text != "valid2" {
		t.Errorf("expected text %q, got %q", "valid2", scripts[1].Text)
	}
}

func TestFileReader_Read_JSONLFile_MissingTextSkipped(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "script.jsonl")
	content := `{"text": "valid"}
{"speaker": 5}
{"text": "also valid"}`
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	reader := infra.NewFileReader()
	scripts, err := reader.Read(filePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(scripts) != 2 {
		t.Fatalf("expected 2 scripts (missing text skipped), got %d", len(scripts))
	}

	if scripts[0].Text != "valid" {
		t.Errorf("expected text %q, got %q", "valid", scripts[0].Text)
	}
	if scripts[1].Text != "also valid" {
		t.Errorf("expected text %q, got %q", "also valid", scripts[1].Text)
	}
}

func TestFileReader_Read_JSONLFile_TrailingDataSkipped(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "script.jsonl")
	content := `{"text": "valid"}
{"text": "ok"} garbage
{"text": "also valid"}`
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	reader := infra.NewFileReader()
	scripts, err := reader.Read(filePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(scripts) != 2 {
		t.Fatalf("expected 2 scripts (trailing data line skipped), got %d", len(scripts))
	}

	if scripts[0].Text != "valid" {
		t.Errorf("expected text %q, got %q", "valid", scripts[0].Text)
	}
	if scripts[1].Text != "also valid" {
		t.Errorf("expected text %q, got %q", "also valid", scripts[1].Text)
	}
}

func TestFileReader_Read_JSONLFile_AllInvalid(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "script.jsonl")
	content := `{invalid}
{also invalid}
{"speaker": 1}`
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	reader := infra.NewFileReader()
	scripts, err := reader.Read(filePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(scripts) != 0 {
		t.Fatalf("expected 0 scripts (all invalid), got %d", len(scripts))
	}
}

func TestFileReader_Read_JSONFile_EmptyText(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "empty.json")
	content := `{"text": ""}`
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	reader := infra.NewFileReader()
	scripts, err := reader.Read(filePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(scripts) != 1 {
		t.Fatalf("expected 1 script, got %d", len(scripts))
	}

	if !scripts[0].IsEmpty {
		t.Error("expected IsEmpty to be true for empty text")
	}
}
