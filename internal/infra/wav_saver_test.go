package infra_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/canpok1/vox-actor/internal/infra"
)

// WavSaver テストリスト
// DONE: 正常系: wav データをファイルに保存できる
// DONE: 正常系: 既存ファイルを上書きできる
// DONE: 正常系: 親ディレクトリが存在しない場合に自動作成される

func TestWavSaver_Save_CreatesFileWithContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "output.wav")
	data := []byte("RIFF fake wav data")

	saver := infra.NewWavSaver()
	if err := saver.Save(path, data); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected file to exist, got error: %v", err)
	}
	if string(got) != string(data) {
		t.Errorf("expected file content %q, got %q", data, got)
	}
}

func TestWavSaver_Save_OverwritesExistingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "output.wav")

	// 既存ファイルを作成
	if err := os.WriteFile(path, []byte("old content"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	newData := []byte("new wav content")
	saver := infra.NewWavSaver()
	if err := saver.Save(path, newData); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected file to exist, got error: %v", err)
	}
	if string(got) != string(newData) {
		t.Errorf("expected overwritten content %q, got %q", newData, got)
	}
}

func TestWavSaver_Save_CreatesParentDirs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a", "b", "c", "output.wav")
	data := []byte("RIFF nested wav")

	saver := infra.NewWavSaver()
	if err := saver.Save(path, data); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected file to exist after mkdir-p, got error: %v", err)
	}
	if string(got) != string(data) {
		t.Errorf("expected file content %q, got %q", data, got)
	}
}
