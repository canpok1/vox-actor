package app

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/canpok1/vox-actor/internal/domain/entity"
)

// AssetsDownloadParams はassetsダウンロードユースケースのパラメータ。
type AssetsDownloadParams struct {
	RepoURL   string
	Speakers  []string // 空の場合は全asset対象
	Force     bool
	AssetsDir string // コピー先のルートディレクトリ（.vox-actor/assets/）
}

// AssetsDownloadOption はAssetsDownloadUsecaseの生成時オプション。
type AssetsDownloadOption func(*AssetsDownloadUsecase)

// WithAssetsDownloadLogger はロガーを設定するオプション。
func WithAssetsDownloadLogger(logger *slog.Logger) AssetsDownloadOption {
	return func(u *AssetsDownloadUsecase) {
		if logger != nil {
			u.logger = logger
		}
	}
}

// AssetsDownloadUsecase はassets downloadサブコマンドのユースケース。
type AssetsDownloadUsecase struct {
	cloner GitCloner
	logger *slog.Logger
}

// NewAssetsDownloadUsecase は新しいAssetsDownloadUsecaseを生成する。
func NewAssetsDownloadUsecase(cloner GitCloner, opts ...AssetsDownloadOption) *AssetsDownloadUsecase {
	u := &AssetsDownloadUsecase{
		cloner: cloner,
		logger: discardLogger(),
	}
	for _, opt := range opts {
		opt(u)
	}
	return u
}

// Run はassets downloadユースケースを実行する。
func (u *AssetsDownloadUsecase) Run(ctx context.Context, params AssetsDownloadParams) error {
	tmpParent, err := os.MkdirTemp("", "vox-actor-assets-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpParent) }()

	repoDir := filepath.Join(tmpParent, "repo")

	u.logger.Info("cloning repository", "url", params.RepoURL)
	if err := u.cloner.Clone(ctx, params.RepoURL, repoDir); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	assetsJSONPath := filepath.Join(repoDir, "vox-actor-assets.json")
	data, err := os.ReadFile(assetsJSONPath)
	if err != nil {
		return fmt.Errorf("vox-actor-assets.json not found in repository: %w", err)
	}

	assetsJSON, err := entity.ParseAssetsJSON(data)
	if err != nil {
		return err
	}
	if err := assetsJSON.Validate(); err != nil {
		return err
	}

	targets, err := assetsJSON.FilterBySpeakers(params.Speakers)
	if err != nil {
		return err
	}

	for name, entry := range targets {
		srcPath := filepath.Join(repoDir, entry.Path)
		destPath := filepath.Join(params.AssetsDir, name)

		if _, err := os.Stat(srcPath); err != nil {
			return fmt.Errorf("assets[%q].path %q not found in repository", name, entry.Path)
		}

		if _, err := os.Stat(destPath); err == nil {
			if !params.Force {
				return fmt.Errorf("assets[%q] already exists at %s: use --force to overwrite", name, destPath)
			}
			if err := os.RemoveAll(destPath); err != nil {
				return fmt.Errorf("failed to remove existing assets[%q]: %w", name, err)
			}
		}

		u.logger.Info("copying asset", "name", name, "src", entry.Path)
		if err := copyDir(srcPath, destPath); err != nil {
			return fmt.Errorf("failed to copy assets[%q]: %w", name, err)
		}
	}

	return nil
}

func copyDir(src, dest string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		destPath := filepath.Join(dest, rel)
		if d.IsDir() {
			return os.MkdirAll(destPath, 0755)
		}
		return copyFile(path, destPath)
	})
}

func copyFile(src, dest string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = srcFile.Close() }()

	destFile, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer func() { _ = destFile.Close() }()

	_, err = io.Copy(destFile, srcFile)
	return err
}
