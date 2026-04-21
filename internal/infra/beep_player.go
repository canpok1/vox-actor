package infra

import (
	"bytes"
	"context"
	"fmt"
	"sync"

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
	// contextがキャンセルされた場合は再生を中断しエラーを返す。
	PlayAndWait(ctx context.Context, s beep.Streamer) error
}

// BeepPlayer はgopxl/beepライブラリを使用してWAVバイト列を再生する。
type BeepPlayer struct {
	backend  speakerBackend
	initOnce sync.Once
	initErr  error
}

// NewBeepPlayer は新しいBeepPlayerを生成する。
// backendがnilの場合はエラーを返す。
func NewBeepPlayer(backend speakerBackend) (*BeepPlayer, error) {
	if backend == nil {
		return nil, fmt.Errorf("speaker backend must not be nil")
	}
	return &BeepPlayer{
		backend: backend,
	}, nil
}

// Play はWAVバイト列をスピーカーから再生する。
// 再生完了まで同期的に待機し、再生失敗時はエラーを返す。
// コンテキストがキャンセルされている場合は即座にエラーを返す。
// meta はスピーカー再生では利用しない。
func (p *BeepPlayer) Play(ctx context.Context, wavData []byte, _ app.PlayMeta) error {
	// コンテキストのキャンセルチェック
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if len(wavData) == 0 {
		return fmt.Errorf("WAV data is empty")
	}

	reader := bytes.NewReader(wavData)

	streamer, format, err := wav.Decode(reader)
	if err != nil {
		return fmt.Errorf("failed to decode WAV data: %w", err)
	}
	defer func() { _ = streamer.Close() }()

	p.initOnce.Do(func() {
		p.initErr = p.backend.Init(format.SampleRate, 2048)
	})
	if p.initErr != nil {
		return fmt.Errorf("failed to initialize speaker: %w", p.initErr)
	}

	// 再生前にもキャンセルチェック
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	return p.backend.PlayAndWait(ctx, streamer)
}
