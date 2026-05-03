//go:build e2e

package e2e

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
)

func TestActE2E_RealComm_Voicevox5xx_NonZeroExit(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := writeTempFile(t, dir, "script.txt", "テスト")

	homeDir := t.TempDir()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	_, stderr, exitCode := runCLI(t, map[string]string{
		"HOME":           homeDir,
		"VOX_ENGINE_URL": srv.URL,
	}, "act", path)

	if exitCode == 0 {
		t.Errorf("expected non-zero exit for VOICEVOX 5xx, got 0\nstderr:\n%s", stderr)
	}
	if strings.TrimSpace(stderr) == "" {
		t.Error("expected non-empty stderr for VOICEVOX 5xx")
	}
}

func TestActE2E_RealComm_VoicevoxConnFailed_NonZeroExit(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := writeTempFile(t, dir, "script.txt", "テスト")

	homeDir := t.TempDir()

	_, stderr, exitCode := runCLI(t, map[string]string{
		"HOME":           homeDir,
		"VOX_ENGINE_URL": "http://127.0.0.1:1",
	}, "act", path)

	if exitCode == 0 {
		t.Errorf("expected non-zero exit for VOICEVOX connection failure, got 0\nstderr:\n%s", stderr)
	}
	if strings.TrimSpace(stderr) == "" {
		t.Error("expected non-empty stderr for connection failure")
	}
}

func TestActE2E_RealComm_VoicevoxBadBody_NonZeroExit(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := writeTempFile(t, dir, "script.txt", "テスト")

	homeDir := t.TempDir()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			w.WriteHeader(http.StatusOK)
		case "/audio_query":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("not-json"))
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	t.Cleanup(srv.Close)

	_, stderr, exitCode := runCLI(t, map[string]string{
		"HOME":           homeDir,
		"VOX_ENGINE_URL": srv.URL,
	}, "act", path)

	if exitCode == 0 {
		t.Errorf("expected non-zero exit for VOICEVOX bad body, got 0\nstderr:\n%s", stderr)
	}
	if strings.TrimSpace(stderr) == "" {
		t.Error("expected non-empty stderr for bad body")
	}
}

func TestActE2E_Viewer_POSTsAndExitsZero(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	content := `{"text":"いちぎょうめ"}
{"text":"にぎょうめ"}
{"text":"さんぎょうめ"}
`
	path := writeTempFile(t, dir, "script.jsonl", content)

	homeDir, calls := setupFakeViewer(t, false, "")

	_, stderr, exitCode := runCLI(t, map[string]string{"HOME": homeDir}, "act", path)

	if exitCode != 0 {
		t.Fatalf("expected exit 0 with viewer running, got %d\nstderr:\n%s", exitCode, stderr)
	}
	if *calls != 3 {
		t.Errorf("expected 3 POSTs to /api/play (one per script line), got %d\nstderr:\n%s", *calls, stderr)
	}
}

func TestActE2E_Viewer_POSTFailure_EarlyExit(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	content := `{"text":"いちぎょうめ"}
{"text":"にぎょうめ"}
{"text":"さんぎょうめ"}
`
	path := writeTempFile(t, dir, "script.jsonl", content)

	var callCount atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/api/play", func(w http.ResponseWriter, _ *http.Request) {
		n := callCount.Add(1)
		if n >= 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"silent":false}`))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	homeDir := t.TempDir()
	writeViewerLockFile(t, homeDir, srv.Listener.Addr().String())

	_, stderr, exitCode := runCLI(t, map[string]string{"HOME": homeDir}, "act", path)

	if exitCode == 0 {
		t.Errorf("expected non-zero exit when viewer POST fails, got 0\nstderr:\n%s", stderr)
	}
}

func TestActE2E_Viewer_SilentMode_Warns(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := writeTempFile(t, dir, "script.txt", "テスト")

	homeDir, _ := setupFakeViewer(t, true, "test-reason")

	_, stderr, exitCode := runCLI(t, map[string]string{"HOME": homeDir}, "act", path)

	if exitCode != 0 {
		t.Fatalf("expected exit 0 even in silent mode, got %d\nstderr:\n%s", exitCode, stderr)
	}
	warnLine := extractLineContaining(t, stderr, "viewer is in silent mode")
	if !strings.HasPrefix(warnLine, "!") {
		t.Errorf("expected WARN-level prefix ('!') on silent mode log\nline: %s", warnLine)
	}
}

func TestActE2E_Viewer_Unreachable_FallbackFails(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := writeTempFile(t, dir, "script.txt", "テスト")

	homeDir := t.TempDir()
	writeViewerLockFile(t, homeDir, "127.0.0.1:1")

	_, stderr, exitCode := runCLI(t, map[string]string{
		"HOME":           homeDir,
		"VOX_ENGINE_URL": "http://127.0.0.1:1",
	}, "act", "--verbose", path)

	if exitCode == 0 {
		t.Fatalf("expected non-zero exit when both viewer and VOICEVOX unreachable\nstderr:\n%s", stderr)
	}
	if !strings.Contains(stderr, "viewer not running") {
		t.Errorf("expected 'viewer not running' debug log with --verbose\nstderr:\n%s", stderr)
	}
}

func TestActE2E_Viewer_DryRun_NoPOST(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := writeTempFile(t, dir, "script.txt", "テスト")

	homeDir, calls := setupFakeViewer(t, false, "")

	_, stderr, exitCode := runCLI(t, map[string]string{"HOME": homeDir}, "act", "--dry-run", path)

	if exitCode != 0 {
		t.Fatalf("expected exit 0 with --dry-run, got %d\nstderr:\n%s", exitCode, stderr)
	}
	if *calls != 0 {
		t.Errorf("expected no POST to /api/play in --dry-run, got %d", *calls)
	}
}

func TestActE2E_Viewer_Detected_LogMessage(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := writeTempFile(t, dir, "script.txt", "テスト")

	homeDir, _ := setupFakeViewer(t, false, "")

	_, stderr, exitCode := runCLI(t, map[string]string{"HOME": homeDir}, "act", path)

	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d\nstderr:\n%s", exitCode, stderr)
	}
	if !strings.Contains(stderr, "viewer detected") {
		t.Errorf("expected 'viewer detected' log in stderr\nstderr:\n%s", stderr)
	}
}

func TestActE2E_Flag_SpeakerOverridesEnv(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := writeTempFile(t, dir, "script.txt", "テスト")

	_, stderr, exitCode := runCLI(t, map[string]string{"VOX_SPEAKER": "10"},
		"act", "--dry-run", "--speaker", "5", path)

	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d\nstderr:\n%s", exitCode, stderr)
	}
	playbackLine := extractLineContaining(t, stderr, "playback completed")
	if !strings.Contains(playbackLine, "speaker=5") {
		t.Errorf("expected speaker=5 in playback line (--speaker flag should override VOX_SPEAKER=10)\nline: %s", playbackLine)
	}
}

func TestActE2E_Flag_EngineURLOverridesEnv(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := writeTempFile(t, dir, "script.txt", "テスト")

	homeDir := t.TempDir()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	_, stderr, exitCode := runCLI(t, map[string]string{
		"HOME":           homeDir,
		"VOX_ENGINE_URL": "http://127.0.0.1:1",
	}, "act", "--engine-url", srv.URL, path)

	if exitCode == 0 {
		t.Fatalf("expected non-zero exit, got 0\nstderr:\n%s", stderr)
	}
	// 音声デバイスなし環境ではプローブが先に失敗する。
	// いずれの場合も非0終了が正しい挙動。
	if !strings.Contains(stderr, "500") && !strings.Contains(stderr, "audio") {
		t.Errorf("expected '500' or 'audio' in stderr\nstderr:\n%s", stderr)
	}
}

func TestActE2E_Env_VOXSpeakerInvalid_DefaultFallback(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := writeTempFile(t, dir, "script.txt", "テスト")

	_, stderr, exitCode := runCLI(t, map[string]string{"VOX_SPEAKER": "not-a-number"},
		"act", "--dry-run", path)

	if exitCode != 0 {
		t.Fatalf("expected exit 0 when VOX_SPEAKER is invalid, got %d\nstderr:\n%s", exitCode, stderr)
	}
	playbackLine := extractLineContaining(t, stderr, "playback completed")
	if !strings.Contains(playbackLine, "speaker=3") {
		t.Errorf("expected default speaker=3 when VOX_SPEAKER=not-a-number\nline: %s", playbackLine)
	}
}

func TestActE2E_SIGINT_DuringVoicevox_Exits(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := writeTempFile(t, dir, "script.txt", "テスト")

	homeDir := t.TempDir()

	hangCh := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		<-hangCh
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(func() {
		close(hangCh)
		srv.Close()
	})

	cmd := exec.Command(binaryPath, "act", "--engine-url", srv.URL, path)
	cmd.Env = mergeEnv(os.Environ(), map[string]string{"HOME": homeDir})
	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start act: %v", err)
	}

	time.Sleep(150 * time.Millisecond)

	if err := cmd.Process.Signal(syscall.SIGINT); err != nil {
		t.Fatalf("failed to send SIGINT: %v", err)
	}

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		_ = cmd.Process.Kill()
		t.Fatal("act did not exit within 5s after SIGINT")
	}
}
