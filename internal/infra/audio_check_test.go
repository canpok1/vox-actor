package infra

// テストリスト: CheckAudioDevice
//
// DONE: 正常系: audioInitFuncが成功した場合、バックエンド名を返しエラーはnil
// DONE: 異常系: audioInitFuncが失敗した場合、バックエンド名は空・エラーを返す
// DONE: 異常系: audioInitFuncがタイムアウトを超えてブロックした場合、エラーを返す
// DONE: 異常系: 失敗時のエラーメッセージに元のエラー原因が含まれること

import (
	"errors"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/gopxl/beep/v2"
)

func TestCheckAudioDevice_Success(t *testing.T) {
	orig := audioInitFunc
	audioInitFunc = func(_ beep.SampleRate, _ int) error { return nil }
	t.Cleanup(func() { audioInitFunc = orig })

	backend, err := CheckAudioDevice()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if backend == "" {
		t.Error("expected non-empty backend name")
	}
}

func TestCheckAudioDevice_Success_ReturnsPlatformBackend(t *testing.T) {
	orig := audioInitFunc
	audioInitFunc = func(_ beep.SampleRate, _ int) error { return nil }
	t.Cleanup(func() { audioInitFunc = orig })

	backend, err := CheckAudioDevice()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 現在のGOOSに対応するバックエンド名が返る
	var want string
	switch runtime.GOOS {
	case "darwin":
		want = "coreaudio"
	case "linux":
		want = "pulseaudio"
	case "windows":
		want = "wasapi"
	default:
		// 未対応プラットフォームでは成功でもエラー扱いになる設計のためスキップ
		t.Skipf("unsupported platform for this test: %s", runtime.GOOS)
	}
	if backend != want {
		t.Errorf("backend = %q, want %q", backend, want)
	}
}

func TestCheckAudioDevice_Failure(t *testing.T) {
	orig := audioInitFunc
	audioInitFunc = func(_ beep.SampleRate, _ int) error {
		return errors.New("no output device available")
	}
	t.Cleanup(func() { audioInitFunc = orig })

	backend, err := CheckAudioDevice()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if backend != "" {
		t.Errorf("expected empty backend on failure, got %q", backend)
	}
}

func TestCheckAudioDevice_Failure_ErrorMessageContainsReason(t *testing.T) {
	orig := audioInitFunc
	reason := "no output device available"
	audioInitFunc = func(_ beep.SampleRate, _ int) error {
		return errors.New(reason)
	}
	t.Cleanup(func() { audioInitFunc = orig })

	_, err := CheckAudioDevice()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), reason) {
		t.Errorf("error message should contain reason %q, got: %v", reason, err)
	}
}

func TestCheckAudioDevice_Timeout(t *testing.T) {
	orig := audioInitFunc
	origTimeout := audioCheckTimeout
	audioInitFunc = func(_ beep.SampleRate, _ int) error {
		// タイムアウトより長くブロックする
		time.Sleep(200 * time.Millisecond)
		return nil
	}
	audioCheckTimeout = 20 * time.Millisecond
	t.Cleanup(func() {
		audioInitFunc = orig
		audioCheckTimeout = origTimeout
	})

	start := time.Now()
	_, err := CheckAudioDevice()
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "timeout") &&
		!strings.Contains(strings.ToLower(err.Error()), "timed out") {
		t.Errorf("error should mention timeout, got: %v", err)
	}
	if elapsed > 150*time.Millisecond {
		t.Errorf("CheckAudioDevice should return around timeout, took %s", elapsed)
	}
}
