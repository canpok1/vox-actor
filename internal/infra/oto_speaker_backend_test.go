package infra

import (
	"errors"
	"fmt"
	"testing"

	"github.com/ebitengine/oto/v3"
)

// TestOtoSpeakerBackend_ImplementsSpeakerBackend はコンパイル時にインターフェース準拠を確認する。
func TestOtoSpeakerBackend_ImplementsSpeakerBackend(t *testing.T) {
	t.Parallel()

	var _ speakerBackend = (*otoSpeakerBackend)(nil)
}

// TestOtoSpeakerBackend_Init_ErrorWhenContextFails はotoコンテキスト取得失敗時にエラーを返すことを確認する。
func TestOtoSpeakerBackend_Init_ErrorWhenContextFails(t *testing.T) {
	t.Parallel()

	expectedErr := fmt.Errorf("no audio device")
	b := &otoSpeakerBackend{
		getCtx: func() (*oto.Context, error) {
			return nil, expectedErr
		},
	}

	err := b.Init(44100, 2048)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected %v, got %v", expectedErr, err)
	}
}

// TestOtoSpeakerBackend_Init_SuccessWithValidContext はotoコンテキスト取得成功時にnilを返すことを確認する。
func TestOtoSpeakerBackend_Init_SuccessWithValidContext(t *testing.T) {
	t.Parallel()

	b := &otoSpeakerBackend{
		getCtx: func() (*oto.Context, error) {
			return nil, nil // contextはnilだがエラーなし
		},
	}

	err := b.Init(24000, 2048)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if b.wavSampleRate != 24000 {
		t.Fatalf("expected wavSampleRate=24000, got %d", b.wavSampleRate)
	}
}
