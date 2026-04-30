//go:build e2e

package e2e

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"testing"
)

var indexAssetRe = regexp.MustCompile(`/assets/[^"'\s]+`)

func TestViewerE2E_Index_StatusCode(t *testing.T) {
	t.Parallel()

	port := freePort(t)
	homeDir := t.TempDir()
	workspace := t.TempDir()

	_, addr := startViewerGetAddr(t, map[string]string{
		"HOME":                homeDir,
		"VOX_ACTOR_WORKSPACE": workspace,
	}, port)

	url := fmt.Sprintf("http://%s/", addr)
	resp := getURLWithRetry(t, url)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /: expected 200, got %d", resp.StatusCode)
	}
}

func TestViewerE2E_Index_ContentType(t *testing.T) {
	t.Parallel()

	port := freePort(t)
	homeDir := t.TempDir()
	workspace := t.TempDir()

	_, addr := startViewerGetAddr(t, map[string]string{
		"HOME":                homeDir,
		"VOX_ACTOR_WORKSPACE": workspace,
	}, port)

	url := fmt.Sprintf("http://%s/", addr)
	resp := getURLWithRetry(t, url)
	defer resp.Body.Close()

	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Errorf("Content-Type: expected to contain %q, got %q", "text/html", ct)
	}
}

func TestViewerE2E_Index_HashedAssets(t *testing.T) {
	t.Parallel()

	port := freePort(t)
	homeDir := t.TempDir()
	workspace := t.TempDir()

	_, addr := startViewerGetAddr(t, map[string]string{
		"HOME":                homeDir,
		"VOX_ACTOR_WORKSPACE": workspace,
	}, port)

	url := fmt.Sprintf("http://%s/", addr)
	resp := getURLWithRetry(t, url)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	matches := indexAssetRe.FindAllString(string(body), -1)
	if len(matches) == 0 {
		t.Fatalf("expected index.html to contain hashed asset references (/assets/<hash>.*), got none\nbody:\n%s", body)
	}

	for _, assetPath := range matches {
		assetURL := fmt.Sprintf("http://%s%s", addr, assetPath)
		assetResp := getURLWithRetry(t, assetURL)
		assetResp.Body.Close()
		if assetResp.StatusCode != http.StatusOK {
			t.Errorf("expected 200 for asset %s, got %d", assetPath, assetResp.StatusCode)
		}
	}
}

func TestViewerE2E_Index_KeyStrings(t *testing.T) {
	t.Parallel()

	port := freePort(t)
	homeDir := t.TempDir()
	workspace := t.TempDir()

	_, addr := startViewerGetAddr(t, map[string]string{
		"HOME":                homeDir,
		"VOX_ACTOR_WORKSPACE": workspace,
	}, port)

	url := fmt.Sprintf("http://%s/", addr)
	resp := getURLWithRetry(t, url)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	bodyStr := string(body)

	wants := []string{
		"<!DOCTYPE html>",
		`id="root"`,
		"vox-actor",
	}
	for _, w := range wants {
		if !strings.Contains(bodyStr, w) {
			t.Errorf("expected index.html to contain %q\nbody:\n%s", w, bodyStr)
		}
	}
}

// TestViewerE2E_Index_UnknownPath は存在しないパスへの GET が 404 を返すことを確認する。
// http.FileServer ベースの静的配信のため SPA フォールバックはなく、404 が仕様となる。
func TestViewerE2E_Index_UnknownPath(t *testing.T) {
	t.Parallel()

	port := freePort(t)
	homeDir := t.TempDir()
	workspace := t.TempDir()

	_, addr := startViewerGetAddr(t, map[string]string{
		"HOME":                homeDir,
		"VOX_ACTOR_WORKSPACE": workspace,
	}, port)

	url := fmt.Sprintf("http://%s/unknown-path", addr)
	resp := getURLWithRetry(t, url)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("GET /unknown-path: expected 404, got %d", resp.StatusCode)
	}
}
