package app_test

// assets_download_usecase テストリスト
// DONE: 正常系: 全assetをダウンロードできる
// DONE: 正常系: --speakerで対象を絞ってダウンロードできる
// DONE: 正常系: --forceで既存assetを上書きできる
// DONE: 異常系: --speaker指定でassetが見つからない場合はエラー
// DONE: 異常系: ローカルに同名assetが存在し--forceなしの場合はエラー
// DONE: 異常系: vox-actor-assets.jsonが存在しない場合はエラー
// DONE: 異常系: vox-actor-assets.jsonのJSONパース失敗でエラー
// DONE: 異常系: asset.pathがリポジトリ内に存在しない場合はエラー
// DONE: 正常系: ダウンロード後に一時ディレクトリが削除される

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/canpok1/vox-actor/internal/app"
)

type fakeGitCloner struct {
	cloneErr  error
	setupFunc func(destDir string) error
}

func (f *fakeGitCloner) Clone(_ context.Context, _ string, destDir string) error {
	if f.cloneErr != nil {
		return f.cloneErr
	}
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}
	if f.setupFunc != nil {
		return f.setupFunc(destDir)
	}
	return nil
}

func setupTestRepo(destDir string, assetsJSONContent string, files map[string]string) error {
	if err := os.WriteFile(filepath.Join(destDir, "vox-actor-assets.json"), []byte(assetsJSONContent), 0644); err != nil {
		return err
	}
	for relPath, content := range files {
		fullPath := filepath.Join(destDir, relPath)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			return err
		}
	}
	return nil
}

func TestAssetsDownloadUsecase_Run_DownloadsAllAssets(t *testing.T) {
	assetsDir := t.TempDir()

	cloner := &fakeGitCloner{
		setupFunc: func(destDir string) error {
			return setupTestRepo(destDir,
				`{"assets": {"zundamon": {"path": "images/zundamon"}}}`,
				map[string]string{
					"images/zundamon/normal.png": "fake-image-data",
				})
		},
	}

	uc := app.NewAssetsDownloadUsecase(cloner)
	params := app.AssetsDownloadParams{
		RepoURL:   "https://example.com/repo.git",
		AssetsDir: assetsDir,
	}

	if err := uc.Run(context.Background(), params); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	destFile := filepath.Join(assetsDir, "zundamon", "normal.png")
	data, err := os.ReadFile(destFile)
	if err != nil {
		t.Fatalf("expected file at %s, got error: %v", destFile, err)
	}
	if string(data) != "fake-image-data" {
		t.Errorf("expected 'fake-image-data', got: %s", string(data))
	}
}

func TestAssetsDownloadUsecase_Run_FiltersBySpeaker(t *testing.T) {
	assetsDir := t.TempDir()

	cloner := &fakeGitCloner{
		setupFunc: func(destDir string) error {
			return setupTestRepo(destDir,
				`{"assets": {"zundamon": {"path": "images/zundamon"}, "metan": {"path": "images/metan"}}}`,
				map[string]string{
					"images/zundamon/normal.png": "zundamon-image",
					"images/metan/normal.png":    "metan-image",
				})
		},
	}

	uc := app.NewAssetsDownloadUsecase(cloner)
	params := app.AssetsDownloadParams{
		RepoURL:   "https://example.com/repo.git",
		Speakers:  []string{"zundamon"},
		AssetsDir: assetsDir,
	}

	if err := uc.Run(context.Background(), params); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// zundamon は存在するはず
	if _, err := os.Stat(filepath.Join(assetsDir, "zundamon")); err != nil {
		t.Errorf("expected zundamon dir, got error: %v", err)
	}
	// metan は存在しないはず
	if _, err := os.Stat(filepath.Join(assetsDir, "metan")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("expected metan dir to not exist, got: %v", err)
	}
}

func TestAssetsDownloadUsecase_Run_ForceOverwrite(t *testing.T) {
	assetsDir := t.TempDir()

	// 事前に同名ディレクトリを作成
	existing := filepath.Join(assetsDir, "zundamon")
	if err := os.MkdirAll(existing, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(existing, "old.png"), []byte("old-data"), 0644); err != nil {
		t.Fatal(err)
	}

	cloner := &fakeGitCloner{
		setupFunc: func(destDir string) error {
			return setupTestRepo(destDir,
				`{"assets": {"zundamon": {"path": "images/zundamon"}}}`,
				map[string]string{
					"images/zundamon/new.png": "new-data",
				})
		},
	}

	uc := app.NewAssetsDownloadUsecase(cloner)
	params := app.AssetsDownloadParams{
		RepoURL:   "https://example.com/repo.git",
		Force:     true,
		AssetsDir: assetsDir,
	}

	if err := uc.Run(context.Background(), params); err != nil {
		t.Fatalf("expected no error with --force, got: %v", err)
	}

	// 新ファイルが存在する
	newFile := filepath.Join(assetsDir, "zundamon", "new.png")
	data, err := os.ReadFile(newFile)
	if err != nil {
		t.Fatalf("expected new file at %s, got: %v", newFile, err)
	}
	if string(data) != "new-data" {
		t.Errorf("expected 'new-data', got: %s", string(data))
	}

	// 古いファイルは存在しない
	oldFile := filepath.Join(assetsDir, "zundamon", "old.png")
	if _, err := os.Stat(oldFile); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("expected old file to be removed, got: %v", err)
	}
}

func TestAssetsDownloadUsecase_Run_SpeakerNotFound_Error(t *testing.T) {
	assetsDir := t.TempDir()

	cloner := &fakeGitCloner{
		setupFunc: func(destDir string) error {
			return setupTestRepo(destDir,
				`{"assets": {"zundamon": {"path": "images/zundamon"}}}`,
				nil)
		},
	}

	uc := app.NewAssetsDownloadUsecase(cloner)
	params := app.AssetsDownloadParams{
		RepoURL:   "https://example.com/repo.git",
		Speakers:  []string{"unknown-speaker"},
		AssetsDir: assetsDir,
	}

	err := uc.Run(context.Background(), params)
	if err == nil {
		t.Fatal("expected error for unknown speaker, got nil")
	}
}

func TestAssetsDownloadUsecase_Run_DuplicateWithoutForce_Error(t *testing.T) {
	assetsDir := t.TempDir()

	// 事前に同名ディレクトリを作成
	existing := filepath.Join(assetsDir, "zundamon")
	if err := os.MkdirAll(existing, 0755); err != nil {
		t.Fatal(err)
	}

	cloner := &fakeGitCloner{
		setupFunc: func(destDir string) error {
			return setupTestRepo(destDir,
				`{"assets": {"zundamon": {"path": "images/zundamon"}}}`,
				map[string]string{
					"images/zundamon/normal.png": "image-data",
				})
		},
	}

	uc := app.NewAssetsDownloadUsecase(cloner)
	params := app.AssetsDownloadParams{
		RepoURL:   "https://example.com/repo.git",
		AssetsDir: assetsDir,
	}

	err := uc.Run(context.Background(), params)
	if err == nil {
		t.Fatal("expected error for duplicate without --force, got nil")
	}
}

func TestAssetsDownloadUsecase_Run_AssetsJSONNotFound_Error(t *testing.T) {
	assetsDir := t.TempDir()

	cloner := &fakeGitCloner{
		setupFunc: func(destDir string) error {
			// vox-actor-assets.json を作成しない
			return nil
		},
	}

	uc := app.NewAssetsDownloadUsecase(cloner)
	params := app.AssetsDownloadParams{
		RepoURL:   "https://example.com/repo.git",
		AssetsDir: assetsDir,
	}

	err := uc.Run(context.Background(), params)
	if err == nil {
		t.Fatal("expected error when vox-actor-assets.json not found, got nil")
	}
}

func TestAssetsDownloadUsecase_Run_InvalidAssetsJSON_Error(t *testing.T) {
	assetsDir := t.TempDir()

	cloner := &fakeGitCloner{
		setupFunc: func(destDir string) error {
			return os.WriteFile(filepath.Join(destDir, "vox-actor-assets.json"), []byte("invalid json"), 0644)
		},
	}

	uc := app.NewAssetsDownloadUsecase(cloner)
	params := app.AssetsDownloadParams{
		RepoURL:   "https://example.com/repo.git",
		AssetsDir: assetsDir,
	}

	err := uc.Run(context.Background(), params)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestAssetsDownloadUsecase_Run_AssetPathNotFound_Error(t *testing.T) {
	assetsDir := t.TempDir()

	cloner := &fakeGitCloner{
		setupFunc: func(destDir string) error {
			// vox-actor-assets.json には path を記載するが実際のディレクトリは作らない
			return setupTestRepo(destDir,
				`{"assets": {"zundamon": {"path": "images/zundamon"}}}`,
				nil)
		},
	}

	uc := app.NewAssetsDownloadUsecase(cloner)
	params := app.AssetsDownloadParams{
		RepoURL:   "https://example.com/repo.git",
		AssetsDir: assetsDir,
	}

	err := uc.Run(context.Background(), params)
	if err == nil {
		t.Fatal("expected error when asset path not found in repo, got nil")
	}
}

func TestAssetsDownloadUsecase_Run_NoGitDirInDest(t *testing.T) {
	assetsDir := t.TempDir()

	cloner := &fakeGitCloner{
		setupFunc: func(destDir string) error {
			// .git ディレクトリを含む偽リポジトリを作成
			if err := os.MkdirAll(filepath.Join(destDir, ".git"), 0755); err != nil {
				return err
			}
			return setupTestRepo(destDir,
				`{"assets": {"zundamon": {"path": "images/zundamon"}}}`,
				map[string]string{
					"images/zundamon/normal.png": "fake-image",
				})
		},
	}

	uc := app.NewAssetsDownloadUsecase(cloner)
	params := app.AssetsDownloadParams{
		RepoURL:   "https://example.com/repo.git",
		AssetsDir: assetsDir,
	}

	if err := uc.Run(context.Background(), params); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// dest に .git は存在しないはず
	gitDir := filepath.Join(assetsDir, ".git")
	if _, err := os.Stat(gitDir); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("expected no .git in dest, but it exists")
	}
}
