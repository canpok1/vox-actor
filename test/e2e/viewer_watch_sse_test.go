//go:build e2e

package e2e

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestViewerE2E_Watch_FileDeliveredViaSSE(t *testing.T) {
	t.Parallel()

	fakeVox := startFakeVoicevox(t)
	watchDir := t.TempDir()
	port := freePort(t)
	homeDir := t.TempDir()
	workspace := t.TempDir()

	vp := startViewer(t, map[string]string{
		"HOME":                homeDir,
		"VOX_ACTOR_WORKSPACE": workspace,
		"VOX_ENGINE_URL":      fakeVox.URL,
	}, "--port", fmt.Sprintf("%d", port), "--watch", watchDir)

	line := vp.waitForStderr("stream server listening", 10*time.Second)
	addr := extractAddrFromLog(line)
	if addr == "" {
		t.Fatalf("failed to extract addr: %q", line)
	}
	vp.waitForStderr("watching directory", 10*time.Second)

	sseResp := connectSSE(t, addr)
	defer sseResp.Body.Close()
	time.Sleep(100 * time.Millisecond)

	if err := os.WriteFile(filepath.Join(watchDir, "hello.txt"), []byte("SSE配信テスト"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	ev := waitForSSEClipEvent(t, sseResp, 15*time.Second)
	assertSSEClipText(t, ev, "SSE配信テスト")

	vp.assertCleanExit(3 * time.Second)
}

func TestViewerE2E_Watch_DeleteMode_FileDeleted(t *testing.T) {
	t.Parallel()

	fakeVox := startFakeVoicevox(t)
	watchDir := t.TempDir()
	port := freePort(t)
	homeDir := t.TempDir()
	workspace := t.TempDir()

	vp := startViewer(t, map[string]string{
		"HOME":                homeDir,
		"VOX_ACTOR_WORKSPACE": workspace,
		"VOX_ENGINE_URL":      fakeVox.URL,
	}, "--port", fmt.Sprintf("%d", port), "--watch", watchDir, "--delete")

	vp.waitForStderr("watching directory", 10*time.Second)

	filePath := filepath.Join(watchDir, "del.txt")
	if err := os.WriteFile(filePath, []byte("削除モードテスト"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	waitForFileGone(t, filePath, 10*time.Second)

	doneDir := filepath.Join(watchDir, "done")
	if _, err := os.Stat(doneDir); err == nil {
		entries, _ := os.ReadDir(doneDir)
		if len(entries) > 0 {
			t.Errorf("expected done/ to be empty in --delete mode, got: %v", entries)
		}
	}

	vp.assertCleanExit(3 * time.Second)
}

func TestViewerE2E_WatchQueue_FileDeliveredViaSSE(t *testing.T) {
	t.Parallel()

	fakeVox := startFakeVoicevox(t)
	port := freePort(t)
	homeDir := t.TempDir()
	workspace := t.TempDir()

	vp := startViewer(t, map[string]string{
		"HOME":                homeDir,
		"VOX_ACTOR_WORKSPACE": workspace,
		"VOX_ENGINE_URL":      fakeVox.URL,
	}, "--port", fmt.Sprintf("%d", port), "--watch-queue")

	line := vp.waitForStderr("stream server listening", 10*time.Second)
	addr := extractAddrFromLog(line)
	if addr == "" {
		t.Fatalf("failed to extract addr: %q", line)
	}
	vp.waitForStderr("watching directory", 10*time.Second)

	queueDir := filepath.Join(workspace, "queue")
	if _, err := os.Stat(queueDir); err != nil {
		t.Fatalf("expected queue dir to be auto-created: %v", err)
	}

	sseResp := connectSSE(t, addr)
	defer sseResp.Body.Close()
	time.Sleep(100 * time.Millisecond)

	if err := os.WriteFile(filepath.Join(queueDir, "queue.txt"), []byte("キューテスト"), 0o644); err != nil {
		t.Fatalf("write queue file: %v", err)
	}

	ev := waitForSSEClipEvent(t, sseResp, 15*time.Second)
	assertSSEClipText(t, ev, "キューテスト")

	vp.assertCleanExit(3 * time.Second)
}

func TestViewerE2E_Watch_MultipleDirectories_BothWatched(t *testing.T) {
	t.Parallel()

	fakeVox := startFakeVoicevox(t)
	dir1 := t.TempDir()
	dir2 := t.TempDir()
	port := freePort(t)
	homeDir := t.TempDir()
	workspace := t.TempDir()

	vp := startViewer(t, map[string]string{
		"HOME":                homeDir,
		"VOX_ACTOR_WORKSPACE": workspace,
		"VOX_ENGINE_URL":      fakeVox.URL,
	}, "--port", fmt.Sprintf("%d", port), "--watch", dir1, "--watch", dir2)

	line := vp.waitForStderr("stream server listening", 10*time.Second)
	addr := extractAddrFromLog(line)
	if addr == "" {
		t.Fatalf("failed to extract addr: %q", line)
	}

	vp.waitForStderr("watching directory", 10*time.Second)
	stderr := vp.stderr.String()
	if strings.Count(stderr, "watching directory") < 2 {
		t.Errorf("expected at least 2 'watching directory' log entries, got:\n%s", stderr)
	}

	sseResp := connectSSE(t, addr)
	defer sseResp.Body.Close()
	time.Sleep(100 * time.Millisecond)

	eventCh := make(chan sseEvent, 2)
	go func() {
		r := bufio.NewReader(sseResp.Body)
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		for {
			ev, err := readSSEEvent(ctx, r)
			if err != nil {
				return
			}
			if ev.EventType == "clip" {
				eventCh <- ev
			}
		}
	}()

	if err := os.WriteFile(filepath.Join(dir1, "a.txt"), []byte("ディレクトリ1"), 0o644); err != nil {
		t.Fatalf("write dir1 file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir2, "b.txt"), []byte("ディレクトリ2"), 0o644); err != nil {
		t.Fatalf("write dir2 file: %v", err)
	}

	received := make(map[string]bool)
	timeout := time.After(15 * time.Second)
	for len(received) < 2 {
		select {
		case ev := <-eventCh:
			var data struct {
				Text string `json:"text"`
			}
			if err := json.Unmarshal([]byte(ev.Data), &data); err == nil {
				received[data.Text] = true
			}
		case <-timeout:
			t.Fatalf("timeout: only received events for %v, expected both dir1 and dir2", received)
		}
	}

	vp.assertCleanExit(3 * time.Second)
}

func TestViewerE2E_Watch_WatchAndQueueCombined(t *testing.T) {
	t.Parallel()

	watchDir := t.TempDir()
	port := freePort(t)
	homeDir := t.TempDir()
	workspace := t.TempDir()

	vp := startViewer(t, map[string]string{
		"HOME":                homeDir,
		"VOX_ACTOR_WORKSPACE": workspace,
		"VOX_ENGINE_URL":      "http://127.0.0.1:1",
	}, "--port", fmt.Sprintf("%d", port), "--watch", watchDir, "--watch-queue")

	vp.waitForStderr("stream server listening", 10*time.Second)
	vp.waitForStderr("watching directory", 10*time.Second)
	time.Sleep(200 * time.Millisecond)

	stderr := vp.stderr.String()
	if strings.Count(stderr, "watching directory") < 2 {
		t.Errorf("expected at least 2 'watching directory' log entries for --watch + --watch-queue, got:\n%s", stderr)
	}

	queueDir := filepath.Join(workspace, "queue")
	if _, err := os.Stat(queueDir); err != nil {
		t.Errorf("expected queue dir to be auto-created: %v", err)
	}

	vp.assertCleanExit(3 * time.Second)
}

func TestViewerE2E_Delete_WithoutWatch_StartsNormally(t *testing.T) {
	t.Parallel()

	port := freePort(t)
	homeDir := t.TempDir()
	workspace := t.TempDir()

	vp := startViewer(t, map[string]string{
		"HOME":                homeDir,
		"VOX_ACTOR_WORKSPACE": workspace,
		"VOX_ENGINE_URL":      "http://127.0.0.1:1",
	}, "--port", fmt.Sprintf("%d", port), "--delete")

	vp.waitForStderr("stream server listening", 10*time.Second)
	vp.assertCleanExit(3 * time.Second)
}
