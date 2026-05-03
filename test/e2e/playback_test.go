//go:build e2e

package e2e

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// playbackStatusHandler は GET /api/playback/{id} に status を返すハンドラを生成する。
func playbackStatusHandler(status string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/api/playback/")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":     id,
			"status": status,
		})
	}
}

// setupFakeViewerWithPlayback は /api/status と /api/playback/{id} を実装した
// fake viewer サーバーを起動し、ロックファイルを書き込んだ homeDir を返す。
// playbackStatus に指定したステータスが GET /api/playback/{id} のレスポンスに使われる。
func setupFakeViewerWithPlayback(t *testing.T, playbackStatus string) string {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/api/playback/", playbackStatusHandler(playbackStatus))

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	homeDir := t.TempDir()
	writeViewerLockFile(t, homeDir, srv.Listener.Addr().String())
	return homeDir
}

func TestPlaybackWaitE2E_LocalPlayback_ExitsZero(t *testing.T) {
	t.Parallel()

	_, _, exitCode := runCLI(t, nil, "playback", "wait", "local_playback")
	if exitCode != 0 {
		t.Fatalf("expected exit code 0 for local_playback, got %d", exitCode)
	}
}

func TestPlaybackWaitE2E_Completed_ExitsZero(t *testing.T) {
	t.Parallel()

	homeDir := setupFakeViewerWithPlayback(t, "completed")

	_, _, exitCode := runCLI(t, map[string]string{"HOME": homeDir}, "playback", "wait", "test-id-123")
	if exitCode != 0 {
		t.Fatalf("expected exit code 0 when playback completed, got %d", exitCode)
	}
}

func TestPlaybackWaitE2E_ViewerURL_SkipsLockfile(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/playback/", playbackStatusHandler("completed"))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	homeDir := t.TempDir() // ロックファイルなし

	_, _, exitCode := runCLI(t, map[string]string{"HOME": homeDir},
		"playback", "wait", "test-id-456", "--viewer-url", srv.URL)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0 with --viewer-url, got %d", exitCode)
	}
}

func TestPlaybackWaitE2E_ServerDownTimeout_NonZero(t *testing.T) {
	t.Parallel()

	_, stderr, exitCode := runCLI(t, nil,
		"playback", "wait", "test-id",
		"--viewer-url", "http://127.0.0.1:1",
		"--server-down-timeout", "500ms",
	)
	if exitCode == 0 {
		t.Fatalf("expected non-zero exit when server is unreachable, got 0\nstderr:\n%s", stderr)
	}
}

func TestPlaybackWaitE2E_NoArgs_ExitCode2(t *testing.T) {
	t.Parallel()

	_, _, exitCode := runCLI(t, nil, "playback", "wait")
	if exitCode != 2 {
		t.Fatalf("expected exit code 2 for missing args, got %d", exitCode)
	}
}

func TestPlaybackWaitE2E_ViewerNotFound_Error(t *testing.T) {
	t.Parallel()

	homeDir := t.TempDir() // ロックファイルなし

	_, stderr, exitCode := runCLI(t, map[string]string{"HOME": homeDir}, "playback", "wait", "test-id")
	if exitCode == 0 {
		t.Fatalf("expected non-zero exit when viewer not found, got 0")
	}
	if !strings.Contains(stderr, "viewer not found") {
		t.Errorf("expected stderr to contain 'viewer not found'\nstderr:\n%s", stderr)
	}
}

func TestPlaybackWaitE2E_Help_ExitZeroWithAllFlags(t *testing.T) {
	t.Parallel()

	stdout, _, exitCode := runCLI(t, nil, "playback", "wait", "--help")
	if exitCode != 0 {
		t.Fatalf("expected exit code 0 for --help, got %d", exitCode)
	}
	for _, flag := range []string{"--viewer-url", "--server-down-timeout"} {
		if !strings.Contains(stdout, flag) {
			t.Errorf("expected --help output to contain %q\nstdout:\n%s", flag, stdout)
		}
	}
}
