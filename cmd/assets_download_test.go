package cmd

// assets download コマンド テストリスト
// DONE: assetsサブコマンドがrootに登録されている
// DONE: downloadサブコマンドがassetsに登録されている
// DONE: 引数なし（URL未指定）でErrUsageを返す
// DONE: --speakerフラグが存在する
// DONE: --forceフラグが存在する（デフォルトfalse）
// DONE: deps=nilでエラーを返す
// DONE: --speaker a,b のカンマ区切りが正しく分割される（両speakerがDLされる）
// DONE: --speaker a --speaker b のリピートが正しく動作する（両speakerがDLされる）

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// testGitCloner はテスト用の GitCloner 実装。
// Clone 時に指定の vox-actor-assets.json とファイル群を dest に作成する。
type testGitCloner struct {
	assetsJSONContent string
	files             map[string]string
}

func (c *testGitCloner) Clone(_ context.Context, _ string, destDir string) error {
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}
	if c.assetsJSONContent != "" {
		if err := os.WriteFile(filepath.Join(destDir, "vox-actor-assets.json"), []byte(c.assetsJSONContent), 0644); err != nil {
			return err
		}
	}
	for rel, content := range c.files {
		fullPath := filepath.Join(destDir, rel)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			return err
		}
	}
	return nil
}

func findAssetsDownloadCmdFromRoot(t *testing.T, rootCmd *cobra.Command) *cobra.Command {
	t.Helper()
	for _, c := range rootCmd.Commands() {
		if c.Name() == "assets" {
			for _, sc := range c.Commands() {
				if sc.Name() == "download" {
					return sc
				}
			}
		}
	}
	t.Fatal("assets download subcommand not found")
	return nil
}

func TestAssetsCmd_RegisteredAsSubcommand(t *testing.T) {
	rootCmd := makeRootCmd()
	found := false
	for _, c := range rootCmd.Commands() {
		if c.Name() == "assets" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'assets' subcommand to be registered")
	}
}

func TestAssetsDownloadCmd_RegisteredUnderAssets(t *testing.T) {
	rootCmd := makeRootCmd()
	_ = findAssetsDownloadCmdFromRoot(t, rootCmd)
}

func TestAssetsDownloadCmd_NoArgs_ReturnsUsageError(t *testing.T) {
	rootCmd := makeRootCmd()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"assets", "download"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error with no args, got nil")
	}
	if !errors.Is(err, ErrUsage) {
		t.Errorf("expected ErrUsage, got: %v", err)
	}
}

func TestAssetsDownloadCmd_HasSpeakerFlag(t *testing.T) {
	rootCmd := makeRootCmd()
	cmd := findAssetsDownloadCmdFromRoot(t, rootCmd)
	if cmd.Flags().Lookup("speaker") == nil {
		t.Error("expected --speaker flag to exist")
	}
}

func TestAssetsDownloadCmd_HasForceFlag(t *testing.T) {
	rootCmd := makeRootCmd()
	cmd := findAssetsDownloadCmdFromRoot(t, rootCmd)
	force, err := cmd.Flags().GetBool("force")
	if err != nil {
		t.Fatalf("expected --force flag to exist, got error: %v", err)
	}
	if force {
		t.Error("expected --force default to be false")
	}
}

func TestAssetsDownloadCmd_NilDeps_ReturnsError(t *testing.T) {
	deps := &Deps{
		Assets: nil,
	}
	rootCmd := makeRootCmd(deps)
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"assets", "download", "https://example.com/repo.git"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error with nil AssetsDownloadDeps, got nil")
	}
}

func TestAssetsDownloadCmd_CommaSeparatedSpeakers(t *testing.T) {
	assetsDir := t.TempDir()

	cloner := &testGitCloner{
		assetsJSONContent: `{"assets": {"a": {"path": "img/a"}, "b": {"path": "img/b"}}}`,
		files: map[string]string{
			"img/a/f.png": "data-a",
			"img/b/f.png": "data-b",
		},
	}

	deps := &Deps{
		Assets: &AssetsDownloadDeps{
			Cloner:        cloner,
			AssetsDirFunc: func() (string, error) { return assetsDir, nil },
		},
	}
	rootCmd := makeRootCmd(deps)
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"assets", "download", "--speaker", "a,b", "https://example.com/repo.git"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	for _, name := range []string{"a", "b"} {
		if _, err := os.Stat(filepath.Join(assetsDir, name)); err != nil {
			t.Errorf("expected speaker %q dir to exist, got: %v", name, err)
		}
	}
}

func TestAssetsDownloadCmd_RepeatSpeakerFlag(t *testing.T) {
	assetsDir := t.TempDir()

	cloner := &testGitCloner{
		assetsJSONContent: `{"assets": {"a": {"path": "img/a"}, "b": {"path": "img/b"}}}`,
		files: map[string]string{
			"img/a/f.png": "data-a",
			"img/b/f.png": "data-b",
		},
	}

	deps := &Deps{
		Assets: &AssetsDownloadDeps{
			Cloner:        cloner,
			AssetsDirFunc: func() (string, error) { return assetsDir, nil },
		},
	}
	rootCmd := makeRootCmd(deps)
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"assets", "download", "--speaker", "a", "--speaker", "b", "https://example.com/repo.git"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	for _, name := range []string{"a", "b"} {
		if _, err := os.Stat(filepath.Join(assetsDir, name)); err != nil {
			t.Errorf("expected speaker %q dir to exist, got: %v", name, err)
		}
	}
}

func TestAssetsDownloadCmd_HelpContainsExpectedFlags(t *testing.T) {
	rootCmd := makeRootCmd()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"assets", "download", "--help"})
	_ = rootCmd.Execute()

	output := buf.String()
	for _, flag := range []string{"--speaker", "--force"} {
		if !strings.Contains(output, flag) {
			t.Errorf("expected %q in help output, got: %s", flag, output)
		}
	}
}

func TestResolveAssetsDir_WithAssetsDirFunc_ReturnsCustomPath(t *testing.T) {
	expectedPath := "/custom/assets"
	deps := &AssetsDownloadDeps{
		AssetsDirFunc: func() (string, error) { return expectedPath, nil },
	}

	result, err := resolveAssetsDir(deps, "project")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result != expectedPath {
		t.Errorf("expected %q, got: %q", expectedPath, result)
	}
}

func TestResolveAssetsDir_WithoutAssetsDirFunc_ReturnsWorkspacePlusAssets(t *testing.T) {
	workspacePath := "/repo/.vox-actor"
	deps := &AssetsDownloadDeps{
		AssetsDirFunc: nil,
	}

	originalResolveFunc := resolveWorkspaceFunc
	resolveWorkspaceFunc = func() (string, error) { return workspacePath, nil }
	defer func() { resolveWorkspaceFunc = originalResolveFunc }()

	result, err := resolveAssetsDir(deps, "project")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	expectedPath := filepath.Join(workspacePath, "assets")
	if result != expectedPath {
		t.Errorf("expected %q, got: %q", expectedPath, result)
	}
}

func TestAssetsDownloadCmd_HasScopeFlag(t *testing.T) {
	rootCmd := makeRootCmd()
	cmd := findAssetsDownloadCmdFromRoot(t, rootCmd)
	f := cmd.Flags().Lookup("scope")
	if f == nil {
		t.Fatal("expected --scope flag to exist")
	}
	if f.DefValue != "project" {
		t.Errorf("expected --scope default to be %q, got %q", "project", f.DefValue)
	}
}

func TestAssetsDownloadCmd_InvalidScope_ReturnsUsageError(t *testing.T) {
	assetsDir := t.TempDir()
	deps := &Deps{
		Assets: &AssetsDownloadDeps{
			Cloner:        &testGitCloner{},
			AssetsDirFunc: func() (string, error) { return assetsDir, nil },
		},
	}
	rootCmd := makeRootCmd(deps)
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"assets", "download", "--scope", "invalid", "https://example.com/repo.git"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid scope, got nil")
	}
	if !errors.Is(err, ErrUsage) {
		t.Errorf("expected ErrUsage, got: %v", err)
	}
}

func TestAssetsDownloadCmd_ScopeHome_UsesHomeAssetsDir(t *testing.T) {
	homeAssetsDir := t.TempDir()
	cloner := &testGitCloner{
		assetsJSONContent: `{"assets": {"charA": {"path": "img/charA"}}}`,
		files:             map[string]string{"img/charA/icon.png": "data"},
	}

	deps := &Deps{
		Assets: &AssetsDownloadDeps{
			Cloner: cloner,
		},
	}

	origHome := resolveHomeAssetsFunc
	resolveHomeAssetsFunc = func() (string, error) { return homeAssetsDir, nil }
	defer func() { resolveHomeAssetsFunc = origHome }()

	rootCmd := makeRootCmd(deps)
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"assets", "download", "--scope", "home", "https://example.com/repo.git"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if _, err := os.Stat(filepath.Join(homeAssetsDir, "charA")); err != nil {
		t.Errorf("expected charA dir in home assets, got: %v", err)
	}
}

func TestAssetsDownloadCmd_ScopeProject_UsesProjectAssetsDir(t *testing.T) {
	projectWorkspace := t.TempDir()
	projectAssetsDir := filepath.Join(projectWorkspace, "assets")
	cloner := &testGitCloner{
		assetsJSONContent: `{"assets": {"charB": {"path": "img/charB"}}}`,
		files:             map[string]string{"img/charB/icon.png": "data"},
	}

	deps := &Deps{
		Assets: &AssetsDownloadDeps{
			Cloner: cloner,
		},
	}

	origWorkspace := resolveWorkspaceFunc
	resolveWorkspaceFunc = func() (string, error) { return projectWorkspace, nil }
	defer func() { resolveWorkspaceFunc = origWorkspace }()

	rootCmd := makeRootCmd(deps)
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"assets", "download", "--scope", "project", "https://example.com/repo.git"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if _, err := os.Stat(filepath.Join(projectAssetsDir, "charB")); err != nil {
		t.Errorf("expected charB dir in project assets, got: %v", err)
	}
}

func TestAssetsDownloadCmd_HelpContainsScopeFlag(t *testing.T) {
	rootCmd := makeRootCmd()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"assets", "download", "--help"})
	_ = rootCmd.Execute()

	output := buf.String()
	if !strings.Contains(output, "--scope") {
		t.Errorf("expected --scope in help output, got: %s", output)
	}
}
