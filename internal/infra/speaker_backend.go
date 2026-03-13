package infra

import (
	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/speaker"
)

// realSpeakerBackend はbeep/speakerパッケージを使用した実際のスピーカーバックエンド。
type realSpeakerBackend struct{}

// NewRealSpeakerBackend は実際のスピーカーバックエンドを生成する。
func NewRealSpeakerBackend() *realSpeakerBackend {
	return &realSpeakerBackend{}
}

func (r *realSpeakerBackend) Init(sampleRate beep.SampleRate, bufferSize int) error {
	return speaker.Init(sampleRate, bufferSize)
}

func (r *realSpeakerBackend) PlayAndWait(s beep.Streamer) {
	done := make(chan struct{})
	speaker.Play(beep.Seq(s, beep.Callback(func() {
		close(done)
	})))
	<-done
}
