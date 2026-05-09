package infra

import (
	"sync"
	"time"

	"github.com/ebitengine/oto/v3"
)

const (
	otoSharedSampleRate   = 44100
	otoSharedChannelCount = 2
	otoSharedBufferSize   = 50 * time.Millisecond
)

// otoContextFactory は oto.Context を生成する関数。テストで差し替え可能にするためパッケージ変数として定義する。
var otoContextFactory = defaultOtoContextFactory

var (
	otoCtxOnce sync.Once
	otoCtxVal  *oto.Context
	otoCtxErr  error
)

func defaultOtoContextFactory() (*oto.Context, error) {
	ctx, ready, err := oto.NewContext(&oto.NewContextOptions{
		SampleRate:   otoSharedSampleRate,
		ChannelCount: otoSharedChannelCount,
		Format:       oto.FormatSignedInt16LE,
		BufferSize:   otoSharedBufferSize,
	})
	if err != nil {
		return nil, err
	}
	<-ready
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return ctx, nil
}

// getOtoContext はプロセス内で共有する oto.Context を返す。初回呼び出し時のみ生成する。
func getOtoContext() (*oto.Context, error) {
	otoCtxOnce.Do(func() {
		otoCtxVal, otoCtxErr = otoContextFactory()
	})
	return otoCtxVal, otoCtxErr
}
