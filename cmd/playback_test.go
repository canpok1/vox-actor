package cmd

// playback wait コマンド テストリスト (#490)
//
// DONE: local_playback を渡すと即 exit 0                          → TestRunPlaybackWait_LocalPlayback_ImmediateSuccess
// DONE: completed で exit 0                                       → TestRunPlaybackWait_Completed_Success
// DONE: failed で exit≠0 かつ stderr に reason                    → TestRunPlaybackWait_Failed_ReturnsError
// DONE: unknown で exit≠0                                         → TestRunPlaybackWait_Unknown_ReturnsError
// DONE: 30 秒連続接続失敗で exit≠0                                → TestRunPlaybackWait_ServerDown_Timeout

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

func newPlaybackWaitCmd(t *testing.T) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	return cmd
}

func TestRunPlaybackWait_LocalPlayback_ImmediateSuccess(t *testing.T) {
	t.Parallel()
	cmd := newPlaybackWaitCmd(t)
	cmd.Flags().String("viewer-url", "", "")
	cmd.Flags().Duration("server-down-timeout", 30*time.Second, "")
	_ = cmd.ParseFlags([]string{})

	deps := &PlaybackWaitDeps{
		LockPathResolver: func() (string, error) { return "", nil },
	}

	err := runPlaybackWait(cmd, []string{"local_playback"}, deps)
	if err != nil {
		t.Errorf("expected nil error for local_playback, got: %v", err)
	}
}

func TestRunPlaybackWait_Completed_Success(t *testing.T) {
	t.Parallel()
	id := "33333333-3333-4333-8333-333333333333"
	var callCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := callCount.Add(1)
		var status string
		if n < 3 {
			status = "playing"
		} else {
			status = "completed"
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":              id,
			"status":          status,
			"clip_count":      1,
			"completed_clips": 0,
		})
	}))
	defer srv.Close()

	cmd := newPlaybackWaitCmd(t)
	cmd.Flags().String("viewer-url", srv.URL, "")
	cmd.Flags().Duration("server-down-timeout", 5*time.Second, "")
	_ = cmd.ParseFlags([]string{"--viewer-url", srv.URL})

	deps := &PlaybackWaitDeps{}

	err := runPlaybackWait(cmd, []string{id}, deps)
	if err != nil {
		t.Errorf("expected nil error for completed, got: %v", err)
	}
}

func TestRunPlaybackWait_Failed_ReturnsError(t *testing.T) {
	t.Parallel()
	id := "44444444-4444-4444-8444-444444444444"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":            id,
			"status":        "failed",
			"clip_count":    1,
			"failed_reason": "synthesis error",
		})
	}))
	defer srv.Close()

	var stderr bytes.Buffer
	cmd := newPlaybackWaitCmd(t)
	cmd.Flags().String("viewer-url", srv.URL, "")
	cmd.Flags().Duration("server-down-timeout", 5*time.Second, "")
	_ = cmd.ParseFlags([]string{"--viewer-url", srv.URL})
	cmd.SetErr(&stderr)

	deps := &PlaybackWaitDeps{}

	err := runPlaybackWait(cmd, []string{id}, deps)
	if err == nil {
		t.Fatal("expected error for failed status, got nil")
	}
	if !strings.Contains(stderr.String(), "synthesis error") {
		t.Errorf("expected stderr to contain reason, got: %q", stderr.String())
	}
}

func TestRunPlaybackWait_Unknown_ReturnsError(t *testing.T) {
	t.Parallel()
	id := "55555555-5555-4555-8555-555555555555"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":     id,
			"status": "unknown",
		})
	}))
	defer srv.Close()

	cmd := newPlaybackWaitCmd(t)
	cmd.Flags().String("viewer-url", srv.URL, "")
	cmd.Flags().Duration("server-down-timeout", 5*time.Second, "")
	_ = cmd.ParseFlags([]string{"--viewer-url", srv.URL})

	deps := &PlaybackWaitDeps{}

	err := runPlaybackWait(cmd, []string{id}, deps)
	if err == nil {
		t.Fatal("expected error for unknown status, got nil")
	}
}

func TestRunPlaybackWait_ServerDown_Timeout(t *testing.T) {
	t.Parallel()
	id := "66666666-6666-4666-8666-666666666666"

	// Do not start a server, so connection will fail
	cmd := newPlaybackWaitCmd(t)
	cmd.Flags().String("viewer-url", "http://127.0.0.1:1", "")
	cmd.Flags().Duration("server-down-timeout", 100*time.Millisecond, "")
	_ = cmd.ParseFlags([]string{"--viewer-url", "http://127.0.0.1:1", "--server-down-timeout", "100ms"})

	deps := &PlaybackWaitDeps{}

	start := time.Now()
	err := runPlaybackWait(cmd, []string{id}, deps)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected error after server-down timeout, got nil")
	}
	if elapsed > 3*time.Second {
		t.Errorf("expected to fail within 3s, took %v", elapsed)
	}
}
