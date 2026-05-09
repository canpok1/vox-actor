package infra

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/ebitengine/oto/v3"
	"github.com/gopxl/beep/v2"
)

// otoSpeakerBackend が speakerBackend インターフェースを満たすことをコンパイル時に検証する。
var _ speakerBackend = (*otoSpeakerBackend)(nil)

// otoSpeakerBackend は共有 oto.Context を直接使うスピーカーバックエンド。
// beep/speaker.Init を呼ばないため、audio_probe との oto.Context 二重生成問題を回避する。
type otoSpeakerBackend struct {
	getCtx        func() (*oto.Context, error)
	wavSampleRate beep.SampleRate
}

// NewOtoSpeakerBackend は共有 oto.Context を使うスピーカーバックエンドを生成する。
func NewOtoSpeakerBackend() *otoSpeakerBackend {
	return &otoSpeakerBackend{getCtx: getOtoContext}
}

// Init はスピーカーを初期化する。oto.Context が利用可能かを確認し、WAV のサンプルレートを保存する。
func (b *otoSpeakerBackend) Init(sampleRate beep.SampleRate, _ int) error {
	if _, err := b.getCtx(); err != nil {
		return err
	}
	b.wavSampleRate = sampleRate
	return nil
}

// PlayAndWait は Streamer を再生し、完了まで待機する。
// WAV のサンプルレートが oto コンテキストのレートと異なる場合はリサンプルする。
func (b *otoSpeakerBackend) PlayAndWait(ctx context.Context, s beep.Streamer) error {
	otoCtx, err := b.getCtx()
	if err != nil {
		return err
	}

	finalStreamer := s
	if b.wavSampleRate != 0 && int(b.wavSampleRate) != otoSharedSampleRate {
		finalStreamer = beep.Resample(3, b.wavSampleRate, beep.SampleRate(otoSharedSampleRate), s)
	}

	done := make(chan struct{})
	seqStreamer := beep.Seq(finalStreamer, beep.Callback(func() {
		close(done)
	}))

	reader := &otoSampleReader{s: seqStreamer}
	player := otoCtx.NewPlayer(reader)
	defer func() { _ = player.Close() }()
	player.Play()

	select {
	case <-done:
		// バッファに残っているサンプルの再生が完了するまで待機する
		for player.IsPlaying() {
			select {
			case <-ctx.Done():
				return fmt.Errorf("playback cancelled: %w", ctx.Err())
			case <-time.After(10 * time.Millisecond):
			}
		}
		return player.Err()
	case <-ctx.Done():
		return fmt.Errorf("playback cancelled: %w", ctx.Err())
	}
}

// otoSampleReader は beep.Streamer を oto.Player が要求する io.Reader に変換する。
type otoSampleReader struct {
	s   beep.Streamer
	buf [][2]float64
}

func (r *otoSampleReader) Read(buf []byte) (int, error) {
	const bytesPerSample = 4 // 2ch * 2bytes(int16)
	if len(buf)%bytesPerSample != 0 {
		return 0, fmt.Errorf("read size not aligned with samples")
	}
	ns := len(buf) / bytesPerSample
	if len(r.buf) < ns {
		r.buf = make([][2]float64, ns)
	}
	n, ok := r.s.Stream(r.buf[:ns])
	if !ok {
		if err := r.s.Err(); err != nil {
			return 0, err
		}
		if n == 0 {
			return 0, io.EOF
		}
	}
	for i := range r.buf[:n] {
		for c := 0; c < 2; c++ {
			val := r.buf[i][c]
			if val < -1 {
				val = -1
			}
			if val > 1 {
				val = 1
			}
			v := int16(val * (1<<15 - 1))
			buf[i*bytesPerSample+c*2+0] = byte(v)
			buf[i*bytesPerSample+c*2+1] = byte(v >> 8)
		}
	}
	return n * bytesPerSample, nil
}
