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
	content := `{"text": "感情込めて", "speaker": 5, "speedScale": 1.5, "pitchScale": 0.1, "intonationScale": 1.8}`
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
	content := `{"speaker": 5, "speedScale": 1.0}`
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
