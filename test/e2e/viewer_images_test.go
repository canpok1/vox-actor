//go:build e2e

package e2e

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

// makeImageFile は assetsDir/<charID>/<file> にダミーバイナリを書き込む。
// 親ディレクトリが存在しない場合は自動作成する。
func makeImageFile(t *testing.T, assetsDir, charID, file string, content []byte) {
	t.Helper()
	charDir := filepath.Join(assetsDir, charID)
	if err := os.MkdirAll(charDir, 0o755); err != nil {
		t.Fatalf("mkdir charDir %s: %v", charDir, err)
	}
	if err := os.WriteFile(filepath.Join(charDir, file), content, 0o644); err != nil {
		t.Fatalf("write image file: %v", err)
	}
}

// getImage は addr の /assets/images/<charID>/<file> に GET してレスポンスを返す。
func getImage(t *testing.T, addr, charID, file string) *http.Response {
	t.Helper()
	url := fmt.Sprintf("http://%s/assets/images/%s/%s", addr, charID, file)
	return getURLWithRetry(t, url)
}

func TestViewerE2E_Image_ProjectAssets_200(t *testing.T) {
	t.Parallel()

	homeDir := t.TempDir()
	workspace := t.TempDir()
	port := freePort(t)

	imgContent := []byte{0x89, 0x50, 0x4e, 0x47} // fake PNG header
	projectAssetsDir := filepath.Join(workspace, "assets")
	makeImageFile(t, projectAssetsDir, "zundamon", "face.png", imgContent)

	_, addr := startViewerGetAddr(t, map[string]string{
		"HOME":                homeDir,
		"VOX_ACTOR_WORKSPACE": workspace,
	}, port)

	resp := getImage(t, addr, "zundamon", "face.png")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if !bytes.Equal(body, imgContent) {
		t.Errorf("expected body %v, got %v", imgContent, body)
	}
}

func TestViewerE2E_Image_HomeAssets_200(t *testing.T) {
	t.Parallel()

	homeDir := t.TempDir()
	workspace := t.TempDir()
	port := freePort(t)

	imgContent := []byte{0x89, 0x50, 0x4e, 0x47} // fake PNG header
	homeAssetsDir := filepath.Join(homeDir, ".vox-actor", "assets")
	makeImageFile(t, homeAssetsDir, "metan", "face.png", imgContent)

	_, addr := startViewerGetAddr(t, map[string]string{
		"HOME":                homeDir,
		"VOX_ACTOR_WORKSPACE": workspace,
	}, port)

	resp := getImage(t, addr, "metan", "face.png")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if !bytes.Equal(body, imgContent) {
		t.Errorf("expected body %v, got %v", imgContent, body)
	}
}

func TestViewerE2E_Image_ProjectPriorityOverHome(t *testing.T) {
	t.Parallel()

	homeDir := t.TempDir()
	workspace := t.TempDir()
	port := freePort(t)

	projectContent := []byte("project-image-data")
	homeContent := []byte("home-image-data")

	// 同じ charID "zundamon" を project と home 両方に配置し、内容を変える
	projectAssetsDir := filepath.Join(workspace, "assets")
	makeImageFile(t, projectAssetsDir, "zundamon", "face.png", projectContent)

	homeAssetsDir := filepath.Join(homeDir, ".vox-actor", "assets")
	makeImageFile(t, homeAssetsDir, "zundamon", "face.png", homeContent)

	_, addr := startViewerGetAddr(t, map[string]string{
		"HOME":                homeDir,
		"VOX_ACTOR_WORKSPACE": workspace,
	}, port)

	resp := getImage(t, addr, "zundamon", "face.png")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if !bytes.Equal(body, projectContent) {
		t.Errorf("expected project image content %q, got %q", projectContent, body)
	}
}

func TestViewerE2E_Image_NotFound_404(t *testing.T) {
	t.Parallel()

	homeDir := t.TempDir()
	workspace := t.TempDir()
	port := freePort(t)

	// assets dir は存在するが、リクエストする charID は存在しない
	projectAssetsDir := filepath.Join(workspace, "assets")
	makeImageFile(t, projectAssetsDir, "zundamon", "face.png", []byte("data"))

	_, addr := startViewerGetAddr(t, map[string]string{
		"HOME":                homeDir,
		"VOX_ACTOR_WORKSPACE": workspace,
	}, port)

	resp := getImage(t, addr, "nonexistent-char", "face.png")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestViewerE2E_Image_PathTraversal_Rejected(t *testing.T) {
	t.Parallel()

	homeDir := t.TempDir()
	workspace := t.TempDir()
	port := freePort(t)

	projectAssetsDir := filepath.Join(workspace, "assets")
	makeImageFile(t, projectAssetsDir, "zundamon", "face.png", []byte("data"))

	_, addr := startViewerGetAddr(t, map[string]string{
		"HOME":                homeDir,
		"VOX_ACTOR_WORKSPACE": workspace,
	}, port)

	url := fmt.Sprintf("http://%s/assets/images/zundamon/../../../etc/passwd", addr)
	resp := getURLWithRetry(t, url)
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		t.Errorf("expected non-200 for path traversal, got %d", resp.StatusCode)
	}
}

func TestViewerE2E_Image_NoAssetsDir_404(t *testing.T) {
	t.Parallel()

	homeDir := t.TempDir()
	workspace := t.TempDir()
	port := freePort(t)

	// assets dir を一切作成しない

	_, addr := startViewerGetAddr(t, map[string]string{
		"HOME":                homeDir,
		"VOX_ACTOR_WORKSPACE": workspace,
	}, port)

	resp := getImage(t, addr, "zundamon", "face.png")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 when no assets dir, got %d", resp.StatusCode)
	}
}
