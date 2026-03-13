package infra

// テストリスト: BeepPlayer
//
// DONE: 異常系: nilのWAVデータを渡した場合、エラーを返す
// DONE: 異常系: 空のWAVデータを渡した場合、エラーを返す
// DONE: 異常系: 不正なWAVデータを渡した場合、エラーを返す
// DONE: インターフェース準拠: BeepPlayerがapp.AudioPlayerを実装していることを確認

import (
	"context"
	"testing"

	"github.com/canpok1/vox-actor/internal/app"
	"github.com/gopxl/beep/v2"
)

// mockSpeakerBackend はテスト用のスピーカーバックエンド。
type mockSpeakerBackend struct {
	initCalled bool
	playCalled bool
}

func (m *mockSpeakerBackend) Init(_ beep.SampleRate, _ int) error {
	m.initCalled = true
	return nil
}

func (m *mockSpeakerBackend) PlayAndWait(_ beep.Streamer) {
	m.playCalled = true
}

func TestBeepPlayer_ImplementsAudioPlayer(t *testing.T) {
	t.Parallel()

	var _ app.AudioPlayer = (*BeepPlayer)(nil)
}

func TestBeepPlayer_Play_NilData(t *testing.T) {
	t.Parallel()

	player := NewBeepPlayer(&mockSpeakerBackend{})
	err := player.Play(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for nil WAV data, got nil")
	}
}

func TestBeepPlayer_Play_EmptyData(t *testing.T) {
	t.Parallel()

	player := NewBeepPlayer(&mockSpeakerBackend{})
	err := player.Play(context.Background(), []byte{})
	if err == nil {
		t.Fatal("expected error for empty WAV data, got nil")
	}
}

func TestBeepPlayer_Play_InvalidWAVData(t *testing.T) {
	t.Parallel()

	player := NewBeepPlayer(&mockSpeakerBackend{})
	invalidData := []byte("this is not a WAV file")
	err := player.Play(context.Background(), invalidData)
	if err == nil {
		t.Fatal("expected error for invalid WAV data, got nil")
	}
}
