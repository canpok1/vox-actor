//go:build e2e

package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"
)

// setupFakeViewer は fake viewer サーバーを起動し、ロックファイルを作成した homeDir を返す。
// silent=true の場合、/api/play は silentReason 付きの silent 応答を返す。
// postCalls は /api/play への POST 回数を追跡する。
func setupFakeViewer(t *testing.T, silent bool, silentReason string) (homeDir string, postCalls *int) {
	t.Helper()
	count := 0
	postCalls = &count

	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/api/play", func(w http.ResponseWriter, _ *http.Request) {
		count++
		w.Header().Set("Content-Type", "application/json")
		resp := map[string]interface{}{"silent": silent}
		if silentReason != "" {
			resp["silent_reason"] = silentReason
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	homeDir = t.TempDir()
	writeViewerLockFile(t, homeDir, srv.Listener.Addr().String())
	return homeDir, postCalls
}

func TestSayE2E_RealComm_Voicevox5xx_NonZeroExit(t *testing.T) {
	t.Parallel()

	homeDir := t.TempDir()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	_, stderr, exitCode := runCLI(t, map[string]string{
		"HOME":           homeDir,
		"VOX_ENGINE_URL": srv.URL,
	}, "say", "テスト")

	if exitCode == 0 {
		t.Errorf("expected non-zero exit for VOICEVOX 5xx, got 0\nstderr:\n%s", stderr)
	}
	// 音声デバイスなし環境ではプローブが先に失敗する。
	// いずれの場合も非0終了が正しい挙動。
	if !strings.Contains(stderr, "500") && !strings.Contains(stderr, "audio") {
		t.Errorf("expected stderr to contain '500' or 'audio', got:\n%s", stderr)
	}
}

func TestSayE2E_RealComm_VoicevoxConnFailed_NonZeroExit(t *testing.T) {
	t.Parallel()

	homeDir := t.TempDir()
	_, stderr, exitCode := runCLI(t, map[string]string{
		"HOME":           homeDir,
		"VOX_ENGINE_URL": "http://127.0.0.1:1",
	}, "say", "テスト")

	if exitCode == 0 {
		t.Errorf("expected non-zero exit for VOICEVOX connection failure, got 0\nstderr:\n%s", stderr)
	}
	if strings.TrimSpace(stderr) == "" {
		t.Error("expected non-empty stderr for connection failure")
	}
}

func TestSayE2E_RealComm_VoicevoxBadBody_NonZeroExit(t *testing.T) {
	t.Parallel()

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
	}, "say", "テスト")

	if exitCode == 0 {
		t.Errorf("expected non-zero exit for invalid VOICEVOX response body, got 0\nstderr:\n%s", stderr)
	}
	if strings.TrimSpace(stderr) == "" {
		t.Error("expected non-empty stderr for bad body")
	}
}

func TestSayE2E_Viewer_POSTsAndExitsZero(t *testing.T) {
	t.Parallel()

	homeDir, calls := setupFakeViewer(t, false, "")

	_, stderr, exitCode := runCLI(t, map[string]string{"HOME": homeDir}, "say", "こんにちは")

	if exitCode != 0 {
		t.Fatalf("expected exit 0 with viewer running, got %d\nstderr:\n%s", exitCode, stderr)
	}
	if *calls != 1 {
		t.Errorf("expected 1 POST to /api/play, got %d", *calls)
	}
}

func TestSayE2E_Viewer_SilentMode_Warns(t *testing.T) {
	t.Parallel()

	homeDir, _ := setupFakeViewer(t, true, "test-reason")

	_, stderr, exitCode := runCLI(t, map[string]string{"HOME": homeDir}, "say", "テスト")

	if exitCode != 0 {
		t.Fatalf("expected exit 0 even in silent mode, got %d\nstderr:\n%s", exitCode, stderr)
	}
	warnLine := extractLineContaining(t, stderr, "viewer is in silent mode")
	if !strings.HasPrefix(warnLine, "!") {
		t.Errorf("expected WARN-level prefix ('!') on silent mode log\nline: %s", warnLine)
	}
}

func TestSayE2E_Viewer_Unreachable_FallbackLogged(t *testing.T) {
	t.Parallel()

	homeDir := t.TempDir()
	writeViewerLockFile(t, homeDir, "127.0.0.1:1")

	_, stderr, exitCode := runCLI(t, map[string]string{
		"HOME":           homeDir,
		"VOX_ENGINE_URL": "http://127.0.0.1:1",
	}, "say", "--verbose", "テスト")

	if exitCode == 0 {
		t.Fatalf("expected non-zero exit when both viewer and VOICEVOX are unreachable\nstderr:\n%s", stderr)
	}
	if !strings.Contains(stderr, "viewer not running") {
		t.Errorf("expected 'viewer not running' debug log with --verbose\nstderr:\n%s", stderr)
	}
}

func TestSayE2E_Viewer_DryRun_NoPOST(t *testing.T) {
	t.Parallel()

	homeDir, calls := setupFakeViewer(t, false, "")

	_, stderr, exitCode := runCLI(t, map[string]string{"HOME": homeDir}, "say", "--dry-run", "テスト")

	if exitCode != 0 {
		t.Fatalf("expected exit 0 with --dry-run, got %d\nstderr:\n%s", exitCode, stderr)
	}
	if *calls != 0 {
		t.Errorf("expected no POST to /api/play in --dry-run, got %d", *calls)
	}
}

func TestSayE2E_Viewer_Detected_LogMessage(t *testing.T) {
	t.Parallel()

	homeDir, _ := setupFakeViewer(t, false, "")

	_, stderr, exitCode := runCLI(t, map[string]string{"HOME": homeDir}, "say", "テスト")

	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d\nstderr:\n%s", exitCode, stderr)
	}
	if !strings.Contains(stderr, "viewer detected") {
		t.Errorf("expected 'viewer detected' log in stderr\nstderr:\n%s", stderr)
	}
}

func TestSayE2E_Flag_SpeakerOverridesEnv(t *testing.T) {
	t.Parallel()

	_, stderr, exitCode := runCLI(t, map[string]string{"VOX_SPEAKER": "10"},
		"say", "--dry-run", "--speaker", "5", "テスト")

	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d\nstderr:\n%s", exitCode, stderr)
	}
	playbackLine := extractLineContaining(t, stderr, "playback completed")
	if !strings.Contains(playbackLine, "speaker=5") {
		t.Errorf("expected speaker=5 in playback line (--speaker flag should override VOX_SPEAKER=10)\nline: %s", playbackLine)
	}
}

func TestSayE2E_Flag_EngineURLOverridesEnv(t *testing.T) {
	t.Parallel()

	homeDir := t.TempDir()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	_, stderr, exitCode := runCLI(t, map[string]string{
		"HOME":           homeDir,
		"VOX_ENGINE_URL": "http://127.0.0.1:1",
	}, "say", "--engine-url", srv.URL, "テスト")

	if exitCode == 0 {
		t.Fatalf("expected non-zero exit, got 0\nstderr:\n%s", stderr)
	}
	// 音声デバイスなし環境ではプローブが先に失敗する。
	// いずれの場合も非0終了が正しい挙動。
	if !strings.Contains(stderr, "500") && !strings.Contains(stderr, "audio") {
		t.Errorf("expected '500' or 'audio' in stderr\nstderr:\n%s", stderr)
	}
}

func TestSayE2E_Env_VOXSpeakerInvalid_DefaultFallback(t *testing.T) {
	t.Parallel()

	_, stderr, exitCode := runCLI(t, map[string]string{"VOX_SPEAKER": "not-a-number"},
		"say", "--dry-run", "テスト")

	if exitCode != 0 {
		t.Fatalf("expected exit 0 when VOX_SPEAKER is invalid, got %d\nstderr:\n%s", exitCode, stderr)
	}
	playbackLine := extractLineContaining(t, stderr, "playback completed")
	if !strings.Contains(playbackLine, "speaker=3") {
		t.Errorf("expected default speaker=3 when VOX_SPEAKER=not-a-number\nline: %s", playbackLine)
	}
}

func TestSayE2E_SIGINT_DuringVoicevox_Exits(t *testing.T) {
	t.Parallel()

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

	cmd := exec.Command(binaryPath, "say", "--engine-url", srv.URL, "テスト")
	cmd.Env = mergeEnv(os.Environ(), map[string]string{"HOME": homeDir})
	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start say: %v", err)
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
		t.Fatal("say did not exit within 5s after SIGINT")
	}
}

func TestSayE2E_Help_ExitZeroWithAllFlags(t *testing.T) {
	t.Parallel()

	stdout, _, exitCode := runCLI(t, nil, "say", "--help")

	if exitCode != 0 {
		t.Fatalf("expected exit 0 for say --help, got %d", exitCode)
	}
	for _, flag := range []string{"--engine-url", "--speaker", "--speed", "--pitch", "--intonation", "--verbose", "--dry-run"} {
		if !strings.Contains(stdout, flag) {
			t.Errorf("expected say --help to contain %q\nstdout:\n%s", flag, stdout)
		}
	}
}

func TestSayE2E_EmptyString_DryRun_ExitsZero(t *testing.T) {
	t.Parallel()

	_, stderr, exitCode := runCLI(t, nil, "say", "--dry-run", "")

	if exitCode != 0 {
		t.Fatalf("expected exit 0 for say --dry-run \"\", got %d\nstderr:\n%s", exitCode, stderr)
	}
	if !strings.Contains(stderr, "playback completed") {
		t.Errorf("expected 'playback completed' in stderr\nstderr:\n%s", stderr)
	}
}

func TestSayE2E_BoundaryValues_DryRun_ExitsZero(t *testing.T) {
	cases := []struct {
		name string
		args []string
	}{
		{"speaker negative", []string{"say", "--dry-run", "--speaker", "-1", "テスト"}},
		{"speed large", []string{"say", "--dry-run", "--speed", "999.0", "テスト"}},
		{"pitch large", []string{"say", "--dry-run", "--pitch", "1.0", "テスト"}},
		{"intonation large", []string{"say", "--dry-run", "--intonation", "999.0", "テスト"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, stderr, exitCode := runCLI(t, nil, tc.args...)
			if exitCode != 0 {
				t.Errorf("expected exit 0 for boundary value %q, got %d\nstderr:\n%s", tc.name, exitCode, stderr)
			}
		})
	}
}

func TestSayE2E_Verbose_ShowsSpecificDebugLogs(t *testing.T) {
	t.Parallel()

	_, stderr, exitCode := runCLI(t, nil, "say", "--dry-run", "--verbose", "テスト内容")

	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d\nstderr:\n%s", exitCode, stderr)
	}
	debugLine := extractLineContaining(t, stderr, "say starting")
	if !strings.Contains(debugLine, "text=テスト内容") {
		t.Errorf("expected debug line to contain 'text=テスト内容'\nline: %s", debugLine)
	}
}

// TestSayE2E_ViewerURL_ConnFailed_NoFallback は --viewer-url で到達不能な URL を指定した場合に
// ローカル再生へのフォールバックなしでエラー終了することを検証する。
func TestSayE2E_ViewerURL_ConnFailed_NoFallback(t *testing.T) {
	t.Parallel()

	homeDir := t.TempDir()

	_, stderr, exitCode := runCLI(t, map[string]string{"HOME": homeDir},
		"say", "--viewer-url", "http://127.0.0.1:1", "テスト")

	if exitCode == 0 {
		t.Fatalf("expected non-zero exit when --viewer-url target is unreachable, got 0\nstderr:\n%s", stderr)
	}
	if strings.TrimSpace(stderr) == "" {
		t.Error("expected non-empty stderr when viewer connection failed")
	}
}
