package infra

// テストリスト: CheckAudioDevice
//
// DONE: 正常系: backend.Initが成功した場合、プラットフォームに対応するバックエンド名を返す
// DONE: 異常系: backend.Initが失敗した場合、バックエンド名は空・エラーを返す
// DONE: 異常系: 失敗時のエラーメッセージに元のエラー原因が含まれること
// DONE: 異常系: backend.Initがタイムアウトを超えてブロックした場合、エラーを返す
// DONE: 異常系: backend が nil の場合、エラーを返す

import (
	"context"
	"errors"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/gopxl/beep/v2"
)

// audioCheckBackend は audio_check のテスト専用 speakerBackend 実装。
// initErr / blockDuration をセットすることで挙動を制御できる。
type audioCheckBackend struct {
	initErr       error
	blockDuration time.Duration
}

func (b *audioCheckBackend) Init(_ beep.SampleRate, _ int) error {
	if b.blockDuration > 0 {
		time.Sleep(b.blockDuration)
	}
	return b.initErr
}

func (b *audioCheckBackend) PlayAndWait(_ context.Context, _ beep.Streamer) error {
	return errors.New("PlayAndWait should not be called in audio-check tests")
}

func TestCheckAudioDevice_Success_ReturnsPlatformBackend(t *testing.T) {
	backend, err := CheckAudioDevice(&audioCheckBackend{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var want string
	switch runtime.GOOS {
	case "darwin":
		want = "coreaudio"
	case "linux":
		want = "pulseaudio"
	case "windows":
		want = "wasapi"
	default:
		t.Skipf("unsupported platform for this test: %s", runtime.GOOS)
	}
	if backend != want {
		t.Errorf("backend = %q, want %q", backend, want)
	}
}

func TestCheckAudioDevice_Failure(t *testing.T) {
	reason := "no output device available"
	backend, err := CheckAudioDevice(&audioCheckBackend{initErr: errors.New(reason)})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if backend != "" {
		t.Errorf("expected empty backend on failure, got %q", backend)
	}
	if !strings.Contains(err.Error(), reason) {
		t.Errorf("error message should contain reason %q, got: %v", reason, err)
	}
}

func TestCheckAudioDevice_Timeout(t *testing.T) {
	start := time.Now()
	backend, err := CheckAudioDevice(
		&audioCheckBackend{blockDuration: 200 * time.Millisecond},
		WithAudioCheckTimeout(20*time.Millisecond),
	)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if backend != "" {
		t.Errorf("expected empty backend on timeout, got %q", backend)
	}
	if !strings.Contains(strings.ToLower(err.Error()), "timed out") {
		t.Errorf("error should mention timeout, got: %v", err)
	}
	if elapsed > 150*time.Millisecond {
		t.Errorf("CheckAudioDevice should return around timeout, took %s", elapsed)
	}
}

func TestCheckAudioDevice_NilBackend(t *testing.T) {
	_, err := CheckAudioDevice(nil)
	if err == nil {
		t.Fatal("expected error when backend is nil, got nil")
	}
}
