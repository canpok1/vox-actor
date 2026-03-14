package infra_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/canpok1/vox-actor/internal/infra"
)

func TestFileMover_MoveToDone_Success(t *testing.T) {
	dir := t.TempDir()
	doneDir := filepath.Join(dir, "done")
	if err := os.Mkdir(doneDir, 0o755); err != nil {
		t.Fatal(err)
	}

	file := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(file, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	mover := infra.NewFileMover()
	if err := mover.MoveToDone(file); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// ファイルが元の場所に存在しないこと
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Error("expected file to be removed from original location")
	}

	// done/に移動していること
	movedFile := filepath.Join(doneDir, "test.txt")
	if _, err := os.Stat(movedFile); err != nil {
		t.Errorf("expected file to exist in done/, got error: %v", err)
	}
}

// TODO: Delete: 存在しないファイルの場合エラーを返す

func TestFileMover_Delete_Success(t *testing.T) {
	dir := t.TempDir()

	file := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(file, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	mover := infra.NewFileMover()
	if err := mover.Delete(file); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// ファイルが削除されていること
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Error("expected file to be deleted")
	}
}

func TestFileMover_MoveToDone_CreatesDoneDir(t *testing.T) {
	dir := t.TempDir()

	file := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(file, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	mover := infra.NewFileMover()
	if err := mover.MoveToDone(file); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// done/ディレクトリが自動作成されていること
	doneDir := filepath.Join(dir, "done")
	info, err := os.Stat(doneDir)
	if err != nil {
		t.Fatalf("expected done/ directory to be created, got error: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected done/ to be a directory")
	}

	// ファイルがdone/に存在すること
	movedFile := filepath.Join(doneDir, "test.txt")
	if _, err := os.Stat(movedFile); err != nil {
		t.Errorf("expected file to exist in done/, got error: %v", err)
	}
}
