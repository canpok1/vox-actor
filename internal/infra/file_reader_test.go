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
