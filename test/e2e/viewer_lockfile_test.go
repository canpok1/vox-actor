//go:build e2e

package e2e

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestViewerE2E_Lockfile_CreatedOnStart は viewer 起動時にロックファイルが作成されることを検証する。
func TestViewerE2E_Lockfile_CreatedOnStart(t *testing.T) {
	t.Parallel()

	port := freePort(t)
	homeDir := t.TempDir()
	workspace := t.TempDir()

	vp, _ := startViewerGetAddr(t, map[string]string{
		"HOME":                homeDir,
		"VOX_ACTOR_WORKSPACE": workspace,
		"VOX_ENGINE_URL":      "http://127.0.0.1:1",
	}, port)

	lockPath := filepath.Join(homeDir, ".vox-actor", "viewer", "viewer.lock")
	if _, err := os.Stat(lockPath); err != nil {
		t.Errorf("expected lockfile to exist at %s: %v", lockPath, err)
	}

	vp.assertCleanExit(3 * time.Second)
}

// TestViewerE2E_Lockfile_UnderHome_NotWorkspace はロックファイルが HOME 配下に作成され、
// VOX_ACTOR_WORKSPACE の影響を受けないことを検証する。
func TestViewerE2E_Lockfile_UnderHome_NotWorkspace(t *testing.T) {
	t.Parallel()

	port := freePort(t)
	homeDir := t.TempDir()
	workspace := t.TempDir()

	vp, _ := startViewerGetAddr(t, map[string]string{
		"HOME":                homeDir,
		"VOX_ACTOR_WORKSPACE": workspace,
		"VOX_ENGINE_URL":      "http://127.0.0.1:1",
	}, port)

	expectedLock := filepath.Join(homeDir, ".vox-actor", "viewer", "viewer.lock")
	if _, err := os.Stat(expectedLock); err != nil {
		t.Errorf("expected lockfile at HOME path %s: %v", expectedLock, err)
	}

	entries, _ := os.ReadDir(workspace)
	for _, e := range entries {
		if e.Name() == "viewer.lock" {
			t.Errorf("lockfile must not be created under VOX_ACTOR_WORKSPACE=%s", workspace)
		}
	}

	vp.assertCleanExit(3 * time.Second)
}

// TestViewerE2E_Lockfile_DoubleStart_ErrorExit は二重起動時に2番目の viewer が
// 非ゼロ終了コードで終了することを検証する。
func TestViewerE2E_Lockfile_DoubleStart_ErrorExit(t *testing.T) {
	t.Parallel()

	homeDir := t.TempDir()
	workspace := t.TempDir()
	port1 := freePort(t)
	port2 := freePort(t)

	vp1, _ := startViewerGetAddr(t, map[string]string{
		"HOME":                homeDir,
		"VOX_ACTOR_WORKSPACE": workspace,
		"VOX_ENGINE_URL":      "http://127.0.0.1:1",
	}, port1)

	// 2番目の viewer を同じ HOME で起動する → ロック競合でエラー終了
	_, _, exitCode := runCLI(t, map[string]string{
		"HOME":                homeDir,
		"VOX_ACTOR_WORKSPACE": workspace,
		"VOX_ENGINE_URL":      "http://127.0.0.1:1",
	}, "viewer", "--port", fmt.Sprintf("%d", port2))

	if exitCode == 0 {
		t.Errorf("expected non-zero exit code for double-start, got 0")
	}

	vp1.assertCleanExit(3 * time.Second)
}

// TestViewerE2E_Lockfile_Restart_AfterStop は1番目の viewer 終了後に再起動が成功することを検証する。
func TestViewerE2E_Lockfile_Restart_AfterStop(t *testing.T) {
	t.Parallel()

	homeDir := t.TempDir()
	workspace := t.TempDir()
	port1 := freePort(t)
	port2 := freePort(t)

	// 1番目の viewer を起動
	vp1 := startViewer(t, map[string]string{
		"HOME":                homeDir,
		"VOX_ACTOR_WORKSPACE": workspace,
		"VOX_ENGINE_URL":      "http://127.0.0.1:1",
	}, "--port", fmt.Sprintf("%d", port1))
	vp1.waitForStderr("stream server listening", 10*time.Second)

	// 1番目を停止
	vp1.assertCleanExit(3 * time.Second)

	// 2番目の viewer を同じ HOME で起動 → 成功するはず
	vp2 := startViewer(t, map[string]string{
		"HOME":                homeDir,
		"VOX_ACTOR_WORKSPACE": workspace,
		"VOX_ENGINE_URL":      "http://127.0.0.1:1",
	}, "--port", fmt.Sprintf("%d", port2))
	vp2.waitForStderr("stream server listening", 10*time.Second)

	vp2.assertCleanExit(3 * time.Second)
}
