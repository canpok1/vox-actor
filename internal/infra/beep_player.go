package infra

import (
	"bytes"
	"context"
	"fmt"

	"github.com/canpok1/vox-actor/internal/app"
	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/wav"
)

// BeepPlayerがapp.AudioPlayerインターフェースを満たすことをコンパイル時に検証する。
var _ app.AudioPlayer = (*BeepPlayer)(nil)

// speakerBackend はスピーカー再生を抽象化するインターフェース。
type speakerBackend interface {
	// Init はスピーカーを初期化する。
	Init(sampleRate beep.SampleRate, bufferSize int) error
	// PlayAndWait はStreamerを再生し、完了まで待機する。
	PlayAndWait(s beep.Streamer)
}

// BeepPlayer はgopxl/beepライブラリを使用してWAVバイト列を再生する。
type BeepPlayer struct {
	backend     speakerBackend
	initialized bool
}

// NewBeepPlayer は新しいBeepPlayerを生成する。
func NewBeepPlayer(backend speakerBackend) *BeepPlayer {
	return &BeepPlayer{
		backend: backend,
	}
}

// Play はWAVバイト列をスピーカーから再生する。
// 再生完了まで同期的に待機し、再生失敗時はエラーを返す。
func (p *BeepPlayer) Play(_ context.Context, wavData []byte) error {
	if len(wavData) == 0 {
		return fmt.Errorf("WAV data is empty")
	}

	reader := bytes.NewReader(wavData)

	streamer, format, err := wav.Decode(reader)
	if err != nil {
		return fmt.Errorf("failed to decode WAV data: %w", err)
	}
	defer func() { _ = streamer.Close() }()

	if !p.initialized {
		bufferSize := format.SampleRate.N(format.SampleRate.D(2048))
		if err := p.backend.Init(format.SampleRate, bufferSize); err != nil {
			return fmt.Errorf("failed to initialize speaker: %w", err)
		}
		p.initialized = true
	}

	p.backend.PlayAndWait(streamer)

	return nil
}
