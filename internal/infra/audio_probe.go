package infra

import (
	"time"

	"github.com/ebitengine/oto/v3"
)

const (
	otoProbeSampleRate   = 44100
	otoProbeChannelCount = 2
	otoProbeBufferSize   = 50 * time.Millisecond
)

// otoAudioProbe は oto を直接使う audioProbe 実装。
// beep/speaker は oto.Context 生成後のエラーを外部に公開しないため、
// デバイス不在を正しく検知するには oto を直接扱う必要がある。
type otoAudioProbe struct{}

// NewOtoAudioProbe は oto を使う audioProbe 実装を生成する。
func NewOtoAudioProbe() *otoAudioProbe {
	return &otoAudioProbe{}
}

// Probe は oto.NewContext でデバイス open 後、ready を待ってから ctx.Err() を参照する。
// ready 待機前の Err() は初期化中のため信頼できない。
func (p *otoAudioProbe) Probe() error {
	ctx, ready, err := oto.NewContext(&oto.NewContextOptions{
		SampleRate:   otoProbeSampleRate,
		ChannelCount: otoProbeChannelCount,
		Format:       oto.FormatSignedInt16LE,
		BufferSize:   otoProbeBufferSize,
	})
	if err != nil {
		return err
	}
	<-ready
	return ctx.Err()
}
