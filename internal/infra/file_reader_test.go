package infra_test

// テストリスト: FileReader
//
// TODO: 正常系: 単一ファイルパスを指定した場合、そのファイルの内容をScriptとして返す
// TODO: 正常系: ディレクトリパスを指定した場合、.txtファイルを辞書順で返す
// TODO: 正常系: 空ファイルを読み込んだ場合、IsEmptyがtrueのScriptを返す
// TODO: 正常系: UTF-8 BOM付きファイルを読み込んだ場合、BOMを除去して内容を返す
// TODO: 正常系: ディレクトリ内に.txt以外のファイルがある場合、.txtファイルのみ返す
// TODO: 正常系: ディレクトリ内に.txtファイルがない場合、空スライスを返す
// TODO: 異常系: 存在しないパスを指定した場合、エラーを返す
// TODO: 異常系: ファイルが.txt拡張子でない場合でも単一ファイル指定なら読み込む

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

func TestFileReader_Read_NotExistPath(t *testing.T) {
	t.Parallel()

	reader := infra.NewFileReader()
	_, err := reader.Read("/no/such/path/file.txt")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
