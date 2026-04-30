//go:build e2e

package e2e

import (
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// indexAssetRe（viewer_index_test.go）と異なり、/assets/images/ 配下を除外するため別定義。
var hashedFrontendAssetRe = regexp.MustCompile(`/assets/[^/"'\s]+\.(js|css)`)

// fetchIndexBody は addr の / レスポンスボディを返す。
func fetchIndexBody(t *testing.T, addr string) []byte {
	t.Helper()
	url := fmt.Sprintf("http://%s/", addr)
	resp := getURLWithRetry(t, url)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read index.html body: %v", err)
	}
	return body
}

func extractFirstHashedAsset(t *testing.T, addr string) string {
	t.Helper()
	body := fetchIndexBody(t, addr)
	match := hashedFrontendAssetRe.FindString(string(body))
	if match == "" {
		t.Fatalf("expected index.html to contain hashed asset reference (/assets/<hash>.js|css), got none\nbody:\n%s", body)
	}
	return match
}

func extractHashedAssetsByExt(t *testing.T, addr, ext string) []string {
	t.Helper()
	body := fetchIndexBody(t, addr)
	all := hashedFrontendAssetRe.FindAllString(string(body), -1)
	var result []string
	for _, p := range all {
		if strings.HasSuffix(p, ext) {
			result = append(result, p)
		}
	}
	return result
}

func TestViewerE2E_Assets_Hashed_200(t *testing.T) {
	t.Parallel()

	port := freePort(t)
	homeDir := t.TempDir()
	workspace := t.TempDir()

	_, addr := startViewerGetAddr(t, map[string]string{
		"HOME":                homeDir,
		"VOX_ACTOR_WORKSPACE": workspace,
	}, port)

	assetPath := extractFirstHashedAsset(t, addr)
	assetURL := fmt.Sprintf("http://%s%s", addr, assetPath)

	resp := getURLWithRetry(t, assetURL)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET %s: expected 200, got %d", assetPath, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if len(body) == 0 {
		t.Errorf("GET %s: expected non-empty body", assetPath)
	}
}

func TestViewerE2E_Assets_Hashed_CacheControl(t *testing.T) {
	t.Parallel()

	port := freePort(t)
	homeDir := t.TempDir()
	workspace := t.TempDir()

	_, addr := startViewerGetAddr(t, map[string]string{
		"HOME":                homeDir,
		"VOX_ACTOR_WORKSPACE": workspace,
	}, port)

	assetPath := extractFirstHashedAsset(t, addr)
	assetURL := fmt.Sprintf("http://%s%s", addr, assetPath)

	resp := getURLWithRetry(t, assetURL)
	defer resp.Body.Close()

	want := "public, max-age=31536000, immutable"
	got := resp.Header.Get("Cache-Control")
	if got != want {
		t.Errorf("GET %s Cache-Control: expected %q, got %q", assetPath, want, got)
	}
}

func TestViewerE2E_Assets_Hashed_ContentType_JS(t *testing.T) {
	t.Parallel()

	port := freePort(t)
	homeDir := t.TempDir()
	workspace := t.TempDir()

	_, addr := startViewerGetAddr(t, map[string]string{
		"HOME":                homeDir,
		"VOX_ACTOR_WORKSPACE": workspace,
	}, port)

	jsPaths := extractHashedAssetsByExt(t, addr, ".js")
	if len(jsPaths) == 0 {
		t.Fatal("expected index.html to contain at least one .js asset reference")
	}

	assetURL := fmt.Sprintf("http://%s%s", addr, jsPaths[0])
	resp := getURLWithRetry(t, assetURL)
	defer resp.Body.Close()

	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "javascript") {
		t.Errorf("GET %s Content-Type: expected to contain \"javascript\", got %q", jsPaths[0], ct)
	}
}

func TestViewerE2E_Assets_Hashed_ContentType_CSS(t *testing.T) {
	t.Parallel()

	port := freePort(t)
	homeDir := t.TempDir()
	workspace := t.TempDir()

	_, addr := startViewerGetAddr(t, map[string]string{
		"HOME":                homeDir,
		"VOX_ACTOR_WORKSPACE": workspace,
	}, port)

	cssPaths := extractHashedAssetsByExt(t, addr, ".css")
	if len(cssPaths) == 0 {
		t.Skip("index.html contains no .css asset references; skipping Content-Type CSS check")
	}

	assetURL := fmt.Sprintf("http://%s%s", addr, cssPaths[0])
	resp := getURLWithRetry(t, assetURL)
	defer resp.Body.Close()

	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "text/css") {
		t.Errorf("GET %s Content-Type: expected to contain \"text/css\", got %q", cssPaths[0], ct)
	}
}

func TestViewerE2E_Assets_NotFound_404(t *testing.T) {
	t.Parallel()

	port := freePort(t)
	homeDir := t.TempDir()
	workspace := t.TempDir()

	_, addr := startViewerGetAddr(t, map[string]string{
		"HOME":                homeDir,
		"VOX_ACTOR_WORKSPACE": workspace,
	}, port)

	url := fmt.Sprintf("http://%s/assets/nonexistent-abc12345.js", addr)
	resp := getURLWithRetry(t, url)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("GET /assets/nonexistent-abc12345.js: expected 404, got %d", resp.StatusCode)
	}
}

func TestViewerE2E_Assets_RoutingMix(t *testing.T) {
	t.Parallel()

	port := freePort(t)
	homeDir := t.TempDir()
	workspace := t.TempDir()

	imgContent := []byte{0x89, 0x50, 0x4e, 0x47} // fake PNG header
	projectAssetsDir := filepath.Join(workspace, "assets")
	makeImageFile(t, projectAssetsDir, "zundamon", "face.png", imgContent)

	_, addr := startViewerGetAddr(t, map[string]string{
		"HOME":                homeDir,
		"VOX_ACTOR_WORKSPACE": workspace,
	}, port)

	assetPath := extractFirstHashedAsset(t, addr)
	assetURL := fmt.Sprintf("http://%s%s", addr, assetPath)
	assetResp := getURLWithRetry(t, assetURL)
	defer assetResp.Body.Close()
	if assetResp.StatusCode != http.StatusOK {
		t.Errorf("GET %s (hashed asset): expected 200, got %d", assetPath, assetResp.StatusCode)
	}

	imgResp := getImage(t, addr, "zundamon", "face.png")
	defer imgResp.Body.Close()
	if imgResp.StatusCode != http.StatusOK {
		t.Errorf("GET /assets/images/zundamon/face.png: expected 200, got %d", imgResp.StatusCode)
	}
}
