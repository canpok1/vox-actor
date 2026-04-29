//go:build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// fakeAsset は fake git リポジトリに含めるアセットエントリを表す。
type fakeAsset struct {
	name  string            // assets キー（speaker 名）
	path  string            // vox-actor-assets.json の path フィールド
	files map[string]string // path 配下のファイル（相対パス → 内容）
}

// makeGitRepoWithFiles は t.TempDir() 配下に git リポジトリを作成し、
// files（リポジトリルートからの相対パス → 内容）をコミットして file:// URL を返す。
// git が利用できない場合は t.Skip() する。
func makeGitRepoWithFiles(t *testing.T, files map[string]string) string {
	t.Helper()

	dir := t.TempDir()

	run := func(name string, args ...string) {
		t.Helper()
		cmd := exec.Command(name, args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Skipf("%s %s failed: %v\n%s", name, strings.Join(args, " "), err, out)
		}
	}

	run("git", "init")
	run("git", "config", "user.email", "test@example.com")
	run("git", "config", "user.name", "Test")

	for relPath, content := range files {
		p := filepath.Join(dir, filepath.FromSlash(relPath))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(p), err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", p, err)
		}
	}

	run("git", "add", ".")
	run("git", "commit", "-m", "init")

	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatalf("EvalSymlinks %q: %v", dir, err)
	}
	return fmt.Sprintf("file://%s", resolved)
}

// makeFakeAssetsRepo は fakeAsset スライスから vox-actor-assets.json とアセットファイルを
// 含む fake git リポジトリを作成し、file:// URL を返す。
func makeFakeAssetsRepo(t *testing.T, assets []fakeAsset) string {
	t.Helper()

	type jsonEntry struct {
		Path string `json:"path"`
	}
	type assetsJSON struct {
		Assets map[string]jsonEntry `json:"assets"`
	}

	aj := assetsJSON{Assets: make(map[string]jsonEntry)}
	for _, a := range assets {
		aj.Assets[a.name] = jsonEntry{Path: a.path}
	}
	data, err := json.Marshal(aj)
	if err != nil {
		t.Fatalf("marshal vox-actor-assets.json: %v", err)
	}

	files := map[string]string{
		"vox-actor-assets.json": string(data),
	}
	for _, a := range assets {
		for relPath, content := range a.files {
			files[a.path+"/"+relPath] = content
		}
	}

	return makeGitRepoWithFiles(t, files)
}

// --- Group A: 親コマンド / ヘルプ / 引数バリデーション ---

func TestAssetsE2E_NoSubcmd_ShowsHelp(t *testing.T) {
	t.Parallel()

	stdout, stderr, exitCode := runCLI(t, nil, "assets")
	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d\nstderr:\n%s", exitCode, stderr)
	}
	for _, want := range []string{"assets", "download"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("expected stdout to contain %q\nstdout:\n%s", want, stdout)
		}
	}
}

func TestAssetsE2E_DownloadHelp_ExitZeroWithAllFlags(t *testing.T) {
	t.Parallel()

	stdout, stderr, exitCode := runCLI(t, nil, "assets", "download", "--help")
	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d\nstderr:\n%s", exitCode, stderr)
	}
	for _, want := range []string{"--speaker", "--force", "--scope"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("expected --help to contain %q\nstdout:\n%s", want, stdout)
		}
	}
}

func TestAssetsE2E_DownloadNoArgs_ExitCode2(t *testing.T) {
	t.Parallel()

	_, _, exitCode := runCLI(t, nil, "assets", "download")
	if exitCode != 2 {
		t.Fatalf("expected exit 2, got %d", exitCode)
	}
}

func TestAssetsE2E_DownloadInvalidScope_ExitCode2(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	_, stderr, exitCode := runCLI(t,
		map[string]string{"VOX_ACTOR_WORKSPACE": workspace},
		"assets", "download", "--scope", "invalid", "file:///dummy",
	)
	if exitCode != 2 {
		t.Fatalf("expected exit 2 for invalid --scope, got %d\nstderr:\n%s", exitCode, stderr)
	}
	if !strings.Contains(stderr, "invalid") && !strings.Contains(stderr, "scope") {
		t.Errorf("expected stderr to mention scope error\nstderr:\n%s", stderr)
	}
}

// --- Group B: ローカル fake git リポジトリでの clone 動作 ---

func TestAssetsE2E_Download_NormalClone(t *testing.T) {
	t.Parallel()

	repoURL := makeFakeAssetsRepo(t, []fakeAsset{
		{
			name: "zundamon",
			path: "speakers/zundamon",
			files: map[string]string{
				"icon.png": "fake-icon",
				"face.png": "fake-face",
			},
		},
	})

	workspace := t.TempDir()
	_, stderr, exitCode := runCLI(t,
		map[string]string{"VOX_ACTOR_WORKSPACE": workspace},
		"assets", "download", repoURL,
	)
	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d\nstderr:\n%s", exitCode, stderr)
	}

	assertFileExists(t, filepath.Join(workspace, "assets", "zundamon", "icon.png"))
	assertFileExists(t, filepath.Join(workspace, "assets", "zundamon", "face.png"))
}

func TestAssetsE2E_Download_ScopeProject(t *testing.T) {
	t.Parallel()

	repoURL := makeFakeAssetsRepo(t, []fakeAsset{
		{
			name:  "kiritan",
			path:  "speakers/kiritan",
			files: map[string]string{"icon.png": "fake"},
		},
	})

	workspace := t.TempDir()
	_, stderr, exitCode := runCLI(t,
		map[string]string{"VOX_ACTOR_WORKSPACE": workspace},
		"assets", "download", "--scope", "project", repoURL,
	)
	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d\nstderr:\n%s", exitCode, stderr)
	}

	assertFileExists(t, filepath.Join(workspace, "assets", "kiritan", "icon.png"))
}

func TestAssetsE2E_Download_ScopeHome(t *testing.T) {
	t.Parallel()

	repoURL := makeFakeAssetsRepo(t, []fakeAsset{
		{
			name:  "metan",
			path:  "speakers/metan",
			files: map[string]string{"icon.png": "fake"},
		},
	})

	homeDir := t.TempDir()
	_, stderr, exitCode := runCLI(t,
		map[string]string{"HOME": homeDir},
		"assets", "download", "--scope", "home", repoURL,
	)
	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d\nstderr:\n%s", exitCode, stderr)
	}

	assertFileExists(t, filepath.Join(homeDir, ".vox-actor", "assets", "metan", "icon.png"))
}

func TestAssetsE2E_Download_MultipleSpeakersRepeat(t *testing.T) {
	t.Parallel()

	repoURL := makeThreeSpeakerRepo(t)

	workspace := t.TempDir()
	_, stderr, exitCode := runCLI(t,
		map[string]string{"VOX_ACTOR_WORKSPACE": workspace},
		"assets", "download",
		"--speaker", "zundamon",
		"--speaker", "metan",
		repoURL,
	)
	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d\nstderr:\n%s", exitCode, stderr)
	}

	assertFileExists(t, filepath.Join(workspace, "assets", "zundamon", "icon.png"))
	assertFileExists(t, filepath.Join(workspace, "assets", "metan", "icon.png"))
	assertFileNotExist(t, filepath.Join(workspace, "assets", "kiritan", "icon.png"))
}

func TestAssetsE2E_Download_MultipleSpeakersComma(t *testing.T) {
	t.Parallel()

	repoURL := makeThreeSpeakerRepo(t)

	workspace := t.TempDir()
	_, stderr, exitCode := runCLI(t,
		map[string]string{"VOX_ACTOR_WORKSPACE": workspace},
		"assets", "download",
		"--speaker", "zundamon,metan",
		repoURL,
	)
	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d\nstderr:\n%s", exitCode, stderr)
	}

	assertFileExists(t, filepath.Join(workspace, "assets", "zundamon", "icon.png"))
	assertFileExists(t, filepath.Join(workspace, "assets", "metan", "icon.png"))
	assertFileNotExist(t, filepath.Join(workspace, "assets", "kiritan", "icon.png"))
}

// makeThreeSpeakerRepo は zundamon / metan / kiritan の 3 speaker を含む fake repo を返す。
// MultipleSpeakers テスト間で共有する。
func makeThreeSpeakerRepo(t *testing.T) string {
	t.Helper()
	return makeFakeAssetsRepo(t, []fakeAsset{
		{name: "zundamon", path: "speakers/zundamon", files: map[string]string{"icon.png": "fake"}},
		{name: "metan", path: "speakers/metan", files: map[string]string{"icon.png": "fake"}},
		{name: "kiritan", path: "speakers/kiritan", files: map[string]string{"icon.png": "fake"}},
	})
}

func TestAssetsE2E_Download_Force_OverwritesExisting(t *testing.T) {
	t.Parallel()

	repoURL := makeFakeAssetsRepo(t, []fakeAsset{
		{
			name:  "zundamon",
			path:  "speakers/zundamon",
			files: map[string]string{"icon.png": "new-content"},
		},
	})

	workspace := t.TempDir()
	existingDir := filepath.Join(workspace, "assets", "zundamon")
	if err := os.MkdirAll(existingDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(existingDir, "old.png"), []byte("old"), 0o644); err != nil {
		t.Fatalf("write old file: %v", err)
	}

	_, stderr, exitCode := runCLI(t,
		map[string]string{"VOX_ACTOR_WORKSPACE": workspace},
		"assets", "download", "--force", repoURL,
	)
	if exitCode != 0 {
		t.Fatalf("expected exit 0 with --force, got %d\nstderr:\n%s", exitCode, stderr)
	}

	assertFileExists(t, filepath.Join(workspace, "assets", "zundamon", "icon.png"))
	assertFileNotExist(t, filepath.Join(workspace, "assets", "zundamon", "old.png"))
}

func TestAssetsE2E_Download_NoForce_ExistingSpeaker_ExitNonZero(t *testing.T) {
	t.Parallel()

	repoURL := makeFakeAssetsRepo(t, []fakeAsset{
		{
			name:  "zundamon",
			path:  "speakers/zundamon",
			files: map[string]string{"icon.png": "fake"},
		},
	})

	workspace := t.TempDir()
	existingDir := filepath.Join(workspace, "assets", "zundamon")
	if err := os.MkdirAll(existingDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	_, stderr, exitCode := runCLI(t,
		map[string]string{"VOX_ACTOR_WORKSPACE": workspace},
		"assets", "download", repoURL,
	)
	if exitCode == 0 {
		t.Fatalf("expected non-zero exit when speaker exists without --force, got 0\nstderr:\n%s", stderr)
	}
}

// --- Group C: エラー系 ---

func TestAssetsE2E_Download_InvalidRepoURL_ExitNonZero(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	_, stderr, exitCode := runCLI(t,
		map[string]string{"VOX_ACTOR_WORKSPACE": workspace},
		"assets", "download", "file:///nonexistent/path/to/repo",
	)
	if exitCode == 0 {
		t.Fatalf("expected non-zero exit for invalid repo URL, got 0")
	}
	if strings.TrimSpace(stderr) == "" {
		t.Error("expected non-empty stderr for clone failure")
	}
}

func TestAssetsE2E_Download_NoAssetsJSON_ExitNonZero(t *testing.T) {
	t.Parallel()

	repoURL := makeGitRepoWithFiles(t, map[string]string{"dummy.txt": "dummy"})

	workspace := t.TempDir()
	_, stderr, exitCode := runCLI(t,
		map[string]string{"VOX_ACTOR_WORKSPACE": workspace},
		"assets", "download", repoURL,
	)
	if exitCode == 0 {
		t.Fatalf("expected non-zero exit when vox-actor-assets.json missing, got 0")
	}
	if strings.TrimSpace(stderr) == "" {
		t.Error("expected non-empty stderr")
	}
}

func TestAssetsE2E_Download_InvalidJSON_ExitNonZero(t *testing.T) {
	t.Parallel()

	repoURL := makeGitRepoWithFiles(t, map[string]string{
		"vox-actor-assets.json": "not-valid-json",
	})

	workspace := t.TempDir()
	_, stderr, exitCode := runCLI(t,
		map[string]string{"VOX_ACTOR_WORKSPACE": workspace},
		"assets", "download", repoURL,
	)
	if exitCode == 0 {
		t.Fatalf("expected non-zero exit for invalid JSON, got 0")
	}
	if strings.TrimSpace(stderr) == "" {
		t.Error("expected non-empty stderr")
	}
}

func TestAssetsE2E_Download_MissingAssetFile_ExitNonZero(t *testing.T) {
	t.Parallel()

	repoURL := makeGitRepoWithFiles(t, map[string]string{
		"vox-actor-assets.json": `{"assets":{"zundamon":{"path":"speakers/zundamon"}}}`,
	})

	workspace := t.TempDir()
	_, stderr, exitCode := runCLI(t,
		map[string]string{"VOX_ACTOR_WORKSPACE": workspace},
		"assets", "download", repoURL,
	)
	if exitCode == 0 {
		t.Fatalf("expected non-zero exit when asset path missing, got 0")
	}
	if strings.TrimSpace(stderr) == "" {
		t.Error("expected non-empty stderr")
	}
}

// --- Group D: ログ ---

func TestAssetsE2E_Download_ProgressLogs(t *testing.T) {
	t.Parallel()

	repoURL := makeFakeAssetsRepo(t, []fakeAsset{
		{
			name:  "zundamon",
			path:  "speakers/zundamon",
			files: map[string]string{"icon.png": "fake"},
		},
	})

	workspace := t.TempDir()
	_, stderr, exitCode := runCLI(t,
		map[string]string{"VOX_ACTOR_WORKSPACE": workspace},
		"assets", "download", repoURL,
	)
	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d\nstderr:\n%s", exitCode, stderr)
	}

	for _, want := range []string{"cloning", "copying"} {
		if !strings.Contains(stderr, want) {
			t.Errorf("expected stderr to contain progress log %q\nstderr:\n%s", want, stderr)
		}
	}
}

func TestAssetsE2E_Download_VerboseFlag_MoreLogs(t *testing.T) {
	t.Parallel()

	repoURL := makeFakeAssetsRepo(t, []fakeAsset{
		{
			name:  "zundamon",
			path:  "speakers/zundamon",
			files: map[string]string{"icon.png": "fake"},
		},
	})

	workspace1 := t.TempDir()
	_, stderrNormal, exitCode := runCLI(t,
		map[string]string{"VOX_ACTOR_WORKSPACE": workspace1},
		"assets", "download", repoURL,
	)
	if exitCode != 0 {
		t.Fatalf("expected exit 0 (normal), got %d\nstderr:\n%s", exitCode, stderrNormal)
	}

	workspace2 := t.TempDir()
	_, stderrVerbose, exitCode := runCLI(t,
		map[string]string{"VOX_ACTOR_WORKSPACE": workspace2},
		"assets", "download", "--verbose", repoURL,
	)
	if exitCode != 0 {
		t.Fatalf("expected exit 0 (verbose), got %d\nstderr:\n%s", exitCode, stderrVerbose)
	}

	normalLines := countNonEmptyLines(stderrNormal)
	verboseLines := countNonEmptyLines(stderrVerbose)
	if verboseLines <= normalLines {
		t.Errorf("expected more log lines with --verbose (%d) than without (%d)\nverbose stderr:\n%s\nnormal stderr:\n%s",
			verboseLines, normalLines, stderrVerbose, stderrNormal)
	}
}

// --- ヘルパー ---

func assertFileExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("expected file to exist: %s", path)
	}
}

func assertFileNotExist(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err == nil {
		t.Errorf("expected file NOT to exist: %s", path)
	}
}
