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

// OtoAudioProbe は ebitengine/oto を直接使って音声出力デバイスを検査する audioProbe 実装。
// beep/speaker は oto.Context 生成後のエラーを外部に公開しないため、
// デバイス不在を正しく検知するには oto を直接扱う必要がある。
type OtoAudioProbe struct{}

// NewOtoAudioProbe は OtoAudioProbe を生成する。
func NewOtoAudioProbe() *OtoAudioProbe {
	return &OtoAudioProbe{}
}

// Probe は oto.NewContext でデバイスを open し、ready 待機後に Err() を参照して
// 実際に open できたかを判定する。open 失敗時は non-nil を返す。
func (p *OtoAudioProbe) Probe() error {
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
