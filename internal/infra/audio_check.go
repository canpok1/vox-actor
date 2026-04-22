package infra

import (
	"fmt"
	"runtime"
	"time"

	"github.com/gopxl/beep/v2"
)

const (
	defaultAudioCheckTimeout = 2 * time.Second
	audioCheckSampleRate     = beep.SampleRate(44100)
	audioCheckBufferSize     = 512
)

// audioCheckConfig は CheckAudioDevice の調整可能なパラメータ。
type audioCheckConfig struct {
	timeout time.Duration
}

// AudioCheckOption は CheckAudioDevice の挙動をカスタマイズする関数型オプション。
type AudioCheckOption func(*audioCheckConfig)

// WithAudioCheckTimeout は CheckAudioDevice の初期化タイムアウトを上書きする。
func WithAudioCheckTimeout(d time.Duration) AudioCheckOption {
	return func(c *audioCheckConfig) { c.timeout = d }
}

// CheckAudioDevice は backend.Init を呼び出して音声出力デバイスの可用性を判定する。
// 成功時はプラットフォームに対応するバックエンド名（coreaudio / pulseaudio / wasapi）を返す。
// 失敗時・タイムアウト時は空文字列とエラーを返す。
//
// 設計メモ: goroutine内のbackend.Initが戻らない場合、timeoutでselectを抜けて戻るが、
// goroutine自体はInitが戻るまで残留する。done channel は buffered (cap=1) のため、
// レシーバが居なくても送信は成功し goroutine は正常終了する。CLIは直後に終了するため
// 短時間なら常駐しない。
func CheckAudioDevice(backend speakerBackend, opts ...AudioCheckOption) (string, error) {
	if backend == nil {
		return "", fmt.Errorf("speaker backend must not be nil")
	}
	cfg := audioCheckConfig{timeout: defaultAudioCheckTimeout}
	for _, o := range opts {
		o(&cfg)
	}

	done := make(chan error, 1)
	go func() {
		done <- backend.Init(audioCheckSampleRate, audioCheckBufferSize)
	}()

	select {
	case err := <-done:
		if err != nil {
			return "", fmt.Errorf("failed to initialize audio device: %w", err)
		}
		return audioBackendName()
	case <-time.After(cfg.timeout):
		return "", fmt.Errorf("audio device initialization timed out after %s", cfg.timeout)
	}
}

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
