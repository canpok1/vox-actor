//go:build e2e

package e2e

import (
	"bytes"
	"encoding/binary"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// minimalSilentWAV は beep/wav が解析できる最小の無音 WAV バイト列を返す。
// 音声再生に失敗しても watch がファイルを done/ に移動することを確認するために使う。
func minimalSilentWAV() []byte {
	var buf bytes.Buffer
	buf.WriteString("RIFF")
	_ = binary.Write(&buf, binary.LittleEndian, uint32(36))
	buf.WriteString("WAVE")
	buf.WriteString("fmt ")
	_ = binary.Write(&buf, binary.LittleEndian, uint32(16))
	_ = binary.Write(&buf, binary.LittleEndian, uint16(1))
	_ = binary.Write(&buf, binary.LittleEndian, uint16(1))
	_ = binary.Write(&buf, binary.LittleEndian, uint32(24000))
	_ = binary.Write(&buf, binary.LittleEndian, uint32(48000))
	_ = binary.Write(&buf, binary.LittleEndian, uint16(2))
	_ = binary.Write(&buf, binary.LittleEndian, uint16(16))
	buf.WriteString("data")
	_ = binary.Write(&buf, binary.LittleEndian, uint32(0))
	return buf.Bytes()
}

func TestWatchE2E_RealComm_HealthCheck5xx_ExitsNonZero(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	homeDir := t.TempDir()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	wp := startWatch(t, map[string]string{
		"HOME":           homeDir,
		"VOX_ENGINE_URL": srv.URL,
	}, dir)

	select {
	case <-wp.done:
	case <-time.After(10 * time.Second):
		t.Fatal("watch did not exit within 10s after HealthCheck 5xx")
	}

	exitCode, _ := wp.stop(3 * time.Second)
	if exitCode == 0 {
		t.Errorf("expected non-zero exit for HealthCheck 5xx, got 0\nstderr:\n%s", wp.stderr.String())
	}
	if strings.TrimSpace(wp.stderr.String()) == "" {
		t.Error("expected non-empty stderr for HealthCheck 5xx")
	}
}

func TestWatchE2E_RealComm_HealthCheckConnFailed_ExitsNonZero(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	homeDir := t.TempDir()

	wp := startWatch(t, map[string]string{
		"HOME":           homeDir,
		"VOX_ENGINE_URL": "http://127.0.0.1:1",
	}, dir)

	select {
	case <-wp.done:
	case <-time.After(10 * time.Second):
		t.Fatal("watch did not exit within 10s after HealthCheck connection failure")
	}

	exitCode, _ := wp.stop(3 * time.Second)
	if exitCode == 0 {
		t.Errorf("expected non-zero exit for HealthCheck connection failure, got 0\nstderr:\n%s", wp.stderr.String())
	}
}

func TestWatchE2E_RealComm_QueryFails_MonitoringContinues(t *testing.T) {
	t.Parallel()
	skipIfNoAudioDevice(t)

	dir := t.TempDir()
	homeDir := t.TempDir()

	// HealthCheck が通った後に query/synthesis が 500 を返す場合、エラーを記録して監視継続する
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	t.Cleanup(srv.Close)

	wp := startWatch(t, map[string]string{
		"HOME":           homeDir,
		"VOX_ENGINE_URL": srv.URL,
	}, dir)
	wp.waitForStderr("watching directory", 5*time.Second)

	src1 := writeTempFile(t, dir, "a.txt", "テスト1")
	wp.waitForStderr("create query error", 10*time.Second)
	assertMovedToDone(t, src1, 5*time.Second)

	src2 := writeTempFile(t, dir, "b.txt", "テスト2")
	wp.waitForStderr("create query error", 10*time.Second)
	assertMovedToDone(t, src2, 5*time.Second)

	wp.assertCleanExit(3 * time.Second)
}

func TestWatchE2E_RealComm_SynthPipeline_Invoked(t *testing.T) {
	t.Parallel()
	skipIfNoAudioDevice(t)

	dir := t.TempDir()
	homeDir := t.TempDir()

	var queryCalls, synthCalls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			w.WriteHeader(http.StatusOK)
		case "/audio_query":
			queryCalls.Add(1)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		case "/synthesis":
			synthCalls.Add(1)
			w.Header().Set("Content-Type", "audio/wav")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(minimalSilentWAV())
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	t.Cleanup(srv.Close)

	wp := startWatch(t, map[string]string{
		"HOME":           homeDir,
		"VOX_ENGINE_URL": srv.URL,
	}, dir)
	wp.waitForStderr("watching directory", 5*time.Second)

	writeTempFile(t, dir, "a.txt", "テスト")

	// /audio_query と /synthesis の各エンドポイントが呼ばれることを確認する。
	// CI環境では音声再生がブロックするため、ファイル移動ではなくAPI呼び出しで検証する。
	waitUntil(t, 10*time.Second, "/audio_query should be called at least once",
		func() bool { return queryCalls.Load() >= 1 })
	waitUntil(t, 10*time.Second, "/synthesis should be called at least once",
		func() bool { return synthCalls.Load() >= 1 })

	stderr := wp.stderr.String()
	if strings.Contains(stderr, "create query error") {
		t.Errorf("unexpected 'create query error': synthesis pipeline should have succeeded\nstderr:\n%s", stderr)
	}
	if strings.Contains(stderr, "synthesize error") {
		t.Errorf("unexpected 'synthesize error': synthesis pipeline should have succeeded\nstderr:\n%s", stderr)
	}

	wp.assertCleanExit(3 * time.Second)
}

// TestWatchE2E_ViewerURL_ConnFailed_LogsError は --viewer-url で到達不能な URL を指定した場合に
// ローカル再生へのフォールバックなしで "silent play error" がログ出力されることを検証する。
func TestWatchE2E_ViewerURL_ConnFailed_LogsError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	wp := startWatch(t, nil, "--viewer-url", "http://127.0.0.1:1", dir)

	wp.waitForStderr("watching directory", 10*time.Second)

	writeTempFile(t, dir, "script.txt", "テスト")

	wp.waitForStderr("silent play error", 10*time.Second)

	doneDir := dir + "/done"
	waitForFileInDir(t, doneDir, "script", 5*time.Second)

	wp.assertCleanExit(3 * time.Second)
}
