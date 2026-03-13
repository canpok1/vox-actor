package infra_test

// テストリスト: FileReader
//
// DONE: 正常系: 単一ファイルパスを指定した場合、そのファイルの内容をScriptとして返す
// DONE: 正常系: ディレクトリパスを指定した場合、.txtファイルを辞書順で返す
// DONE: 正常系: 空ファイルを読み込んだ場合、IsEmptyがtrueのScriptを返す
// DONE: 正常系: UTF-8 BOM付きファイルを読み込んだ場合、BOMを除去して内容を返す
// DONE: 正常系: ディレクトリ内に.txt以外のファイルがある場合、.txtファイルのみ返す
// DONE: 正常系: ディレクトリ内に.txtファイルがない場合、空スライスを返す
// DONE: 異常系: 存在しないパスを指定した場合、エラーを返す
// DONE: 異常系: ファイルが.txt拡張子でない場合でも単一ファイル指定なら読み込む
// DONE: 異常系: 不正なUTF-8バイト列を含むファイルの場合、エラーを返す

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

func TestFileReader_Read_DirectoryFiltersTxtOnly(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "script.txt"), []byte("台本"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "image.png"), []byte("PNG data"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte("{}"), 0o644); err != nil {
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
