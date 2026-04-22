package infra

import (
	"fmt"
	"runtime"
	"time"

	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/speaker"
)

// audioInitFunc は音声デバイスの初期化関数。テストから差し替え可能にするためpackage変数にしている。
var audioInitFunc = func(sampleRate beep.SampleRate, bufferSize int) error {
	return speaker.Init(sampleRate, bufferSize)
}

// audioCheckTimeout はデバイス初期化がブロックした場合の上限。テストから短縮できる。
var audioCheckTimeout = 2 * time.Second

// audioCheckSampleRate はデバイス可用性チェックに使うサンプリングレート。
const audioCheckSampleRate = beep.SampleRate(44100)

// audioCheckBufferSize はデバイス可用性チェックに使うバッファサイズ。
const audioCheckBufferSize = 512

// CheckAudioDevice は音声出力デバイスを open → 即 close のみ行い、可用性を判定する。
// 成功時はバックエンド名（coreaudio / pulseaudio / wasapi など）を返す。
// 失敗時は空文字列とエラーを返す。
func CheckAudioDevice() (string, error) {
	done := make(chan error, 1)
	go func() {
		done <- audioInitFunc(audioCheckSampleRate, audioCheckBufferSize)
	}()

	select {
	case err := <-done:
		if err != nil {
			return "", fmt.Errorf("failed to initialize audio device: %w", err)
		}
		return audioBackendName()
	case <-time.After(audioCheckTimeout):
		return "", fmt.Errorf("audio device initialization timed out after %s", audioCheckTimeout)
	}
}

// audioBackendName は現在のプラットフォームに対応するバックエンド名を返す。
// 未対応プラットフォームではエラーを返す。
func audioBackendName() (string, error) {
	switch runtime.GOOS {
	case "darwin":
		return "coreaudio", nil
	case "linux":
		return "pulseaudio", nil
	case "windows":
		return "wasapi", nil
	default:
		return "", fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}
