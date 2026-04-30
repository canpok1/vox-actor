//go:build e2e

package e2e

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// TestViewerE2E_Help_ExitZero は `viewer --help` が exit 0 で終了し、
// 主要フラグが表示されることを検証する。
func TestViewerE2E_Help_ExitZero(t *testing.T) {
	t.Parallel()

	stdout, _, exitCode := runCLI(t, nil, "viewer", "--help")

	if exitCode != 0 {
		t.Fatalf("expected exit code 0 for viewer --help, got %d", exitCode)
	}

	wantFlags := []string{"--port", "--watch", "--watch-queue", "--delete", "--host", "--engine-url", "--speaker"}
	for _, flag := range wantFlags {
		if !strings.Contains(stdout, flag) {
			t.Errorf("expected --help output to contain flag %q\nstdout:\n%s", flag, stdout)
		}
	}
}

// TestViewerE2E_Env_VOXEngineURL_Applied は VOX_ENGINE_URL 環境変数で指定したサーバーに
// viewer が接続することを検証する（speakers loaded ログで確認）。
func TestViewerE2E_Env_VOXEngineURL_Applied(t *testing.T) {
	t.Parallel()

	fakeVox := startFakeVoicevox(t)
	port := freePort(t)
	homeDir := t.TempDir()
	workspace := t.TempDir()

	// VOX_ENGINE_URL を環境変数で指定（--engine-url フラグは使わない）
	vp := startViewer(t, map[string]string{
		"HOME":                homeDir,
		"VOX_ACTOR_WORKSPACE": workspace,
		"VOX_ENGINE_URL":      fakeVox.URL,
	}, "--port", fmt.Sprintf("%d", port))

	// VOICEVOX に接続成功 → "speakers loaded" ログが出る
	vp.waitForStderr("speakers loaded", 10*time.Second)

	vp.assertCleanExit(3 * time.Second)
}

// TestViewerE2E_Env_VOXSpeaker_Applied は VOX_SPEAKER 環境変数で指定したスピーカーID が
// --watch 時のファイル処理に使われることを検証する。
// fake VOICEVOX サーバーへの /audio_query リクエストに speaker パラメータが含まれることで確認する。
func TestViewerE2E_Env_VOXSpeaker_Applied(t *testing.T) {
	t.Parallel()

	// リクエストを受け取った speaker パラメータを記録する fake VOICEVOX
	var capturedSpeaker atomic.Value
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/speakers", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `[{"name":"テスト","styles":[{"name":"ノーマル","id":5}]}]`) //nolint:errcheck
	})
	mux.HandleFunc("/audio_query", func(w http.ResponseWriter, r *http.Request) {
		capturedSpeaker.Store(r.URL.Query().Get("speaker"))
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"accent_phrases":[],"speedScale":1.0,"pitchScale":0.0,"intonationScale":1.0,"volumeScale":1.0,"prePhonemeLength":0.1,"postPhonemeLength":0.1,"outputSamplingRate":24000,"outputStereo":false,"kana":""}`) //nolint:errcheck
	})
	mux.HandleFunc("/synthesis", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "audio/wav")
		w.Write(make([]byte, 64)) //nolint:errcheck
	})
	fakeVox := httptest.NewServer(mux)
	t.Cleanup(fakeVox.Close)

	watchDir := t.TempDir()
	port := freePort(t)
	homeDir := t.TempDir()
	workspace := t.TempDir()

	// VOX_SPEAKER=5 を環境変数で指定（--speaker フラグは使わない）
	vp := startViewer(t, map[string]string{
		"HOME":                homeDir,
		"VOX_ACTOR_WORKSPACE": workspace,
		"VOX_ENGINE_URL":      fakeVox.URL,
		"VOX_SPEAKER":         "5",
	}, "--port", fmt.Sprintf("%d", port), "--watch", watchDir)

	vp.waitForStderr("watching directory", 10*time.Second)

	// ファイルを投入してスピーカーID の使用を検証
	writeTempFile(t, watchDir, "speak.txt", "スピーカーテスト")

	found := false
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if s, ok := capturedSpeaker.Load().(string); ok && s != "" {
			if s != "5" {
				t.Errorf("expected speaker=5 from VOX_SPEAKER env, got %q", s)
			}
			found = true
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !found {
		t.Error("timeout: /audio_query was not called, VOX_SPEAKER env may not have been applied")
	}

	vp.assertCleanExit(3 * time.Second)
}

// TestViewerE2E_Host_0000_Binds は --host 0.0.0.0 を指定したとき
// 127.0.0.1 でも接続できることを検証する。
// Go の net.Listen("tcp", "0.0.0.0:N") は IPv6 dual-stack では [::] として表示されるため
// ログ内の addr 表現は 0.0.0.0 または [::] のいずれかを許容する。
func TestViewerE2E_Host_0000_Binds(t *testing.T) {
	t.Parallel()

	port := freePort(t)
	homeDir := t.TempDir()
	workspace := t.TempDir()

	vp := startViewer(t, map[string]string{
		"HOME":                homeDir,
		"VOX_ACTOR_WORKSPACE": workspace,
		"VOX_ENGINE_URL":      "http://127.0.0.1:1",
	}, "--port", fmt.Sprintf("%d", port), "--host", "0.0.0.0")

	vp.waitForStderr("stream server listening", 10*time.Second)

	// 127.0.0.1 でアクセス可能か確認（0.0.0.0 バインドの主目的）
	url := fmt.Sprintf("http://127.0.0.1:%d/api/status", port)
	resp := getURLWithRetry(t, url)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 from 127.0.0.1 with --host 0.0.0.0, got %d", resp.StatusCode)
	}

	vp.assertCleanExit(3 * time.Second)
}
