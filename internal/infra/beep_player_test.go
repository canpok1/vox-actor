package infra

// テストリスト: BeepPlayer
//
// DONE: 異常系: nilのWAVデータを渡した場合、エラーを返す
// DONE: 異常系: 空のWAVデータを渡した場合、エラーを返す
// DONE: 異常系: 不正なWAVデータを渡した場合、エラーを返す
// DONE: インターフェース準拠: BeepPlayerがapp.AudioPlayerを実装していることを確認
// DONE: 異常系: backendがnilの場合、NewBeepPlayerがエラーを返す
// DONE: 正常系: 有効なWAVデータでInit/PlayAndWaitが呼ばれること
// DONE: 異常系: キャンセル済みcontextで即時エラーを返すこと

import (
	"context"
	"encoding/binary"
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

func TestNewBeepPlayer_NilBackend(t *testing.T) {
	t.Parallel()

	_, err := NewBeepPlayer(nil)
	if err == nil {
		t.Fatal("expected error for nil backend, got nil")
	}
}

func TestBeepPlayer_Play_NilData(t *testing.T) {
	t.Parallel()

	player, err := NewBeepPlayer(&mockSpeakerBackend{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	err = player.Play(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for nil WAV data, got nil")
	}
}

func TestBeepPlayer_Play_EmptyData(t *testing.T) {
	t.Parallel()

	player, err := NewBeepPlayer(&mockSpeakerBackend{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	err = player.Play(context.Background(), []byte{})
	if err == nil {
		t.Fatal("expected error for empty WAV data, got nil")
	}
}

func TestBeepPlayer_Play_InvalidWAVData(t *testing.T) {
	t.Parallel()

	player, err := NewBeepPlayer(&mockSpeakerBackend{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	invalidData := []byte("this is not a WAV file")
	err = player.Play(context.Background(), invalidData)
	if err == nil {
		t.Fatal("expected error for invalid WAV data, got nil")
	}
}

func TestBeepPlayer_Play_ValidWAV(t *testing.T) {
	t.Parallel()

	mock := &mockSpeakerBackend{}
	player, err := NewBeepPlayer(mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wavData := createMinimalWAV(t)
	err = player.Play(context.Background(), wavData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !mock.initCalled {
		t.Error("expected Init to be called on backend")
	}
	if !mock.playCalled {
		t.Error("expected PlayAndWait to be called on backend")
	}
}

func TestBeepPlayer_Play_CancelledContext(t *testing.T) {
	t.Parallel()

	player, err := NewBeepPlayer(&mockSpeakerBackend{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 即キャンセル

	wavData := createMinimalWAV(t)
	err = player.Play(ctx, wavData)
	if err == nil {
		t.Fatal("expected error for cancelled context, got nil")
	}
	if err != context.Canceled {
		t.Fatalf("expected context.Canceled, got: %v", err)
	}
}

// createMinimalWAV はテスト用の最小限のWAVバイト列を生成する。
// PCM 16bit, mono, 44100Hz, 1サンプルのWAVファイル。
func createMinimalWAV(t *testing.T) []byte {
	t.Helper()

	// WAVファイルフォーマット:
	// RIFF header (12 bytes) + fmt chunk (24 bytes) + data chunk header (8 bytes) + data (2 bytes) = 46 bytes
	buf := make([]byte, 0, 46)

	// RIFF header
	buf = append(buf, []byte("RIFF")...)
	sizeBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(sizeBytes, 38) // file size - 8
	buf = append(buf, sizeBytes...)
	buf = append(buf, []byte("WAVE")...)

	// fmt chunk
	buf = append(buf, []byte("fmt ")...)
	binary.LittleEndian.PutUint32(sizeBytes, 16) // chunk size
	buf = append(buf, sizeBytes...)
	fmtBytes := make([]byte, 16)
	binary.LittleEndian.PutUint16(fmtBytes[0:2], 1)      // PCM format
	binary.LittleEndian.PutUint16(fmtBytes[2:4], 1)      // mono
	binary.LittleEndian.PutUint32(fmtBytes[4:8], 44100)  // sample rate
	binary.LittleEndian.PutUint32(fmtBytes[8:12], 88200) // byte rate (44100 * 1 * 2)
	binary.LittleEndian.PutUint16(fmtBytes[12:14], 2)    // block align (1 * 2)
	binary.LittleEndian.PutUint16(fmtBytes[14:16], 16)   // bits per sample
	buf = append(buf, fmtBytes...)

	// data chunk
	buf = append(buf, []byte("data")...)
	binary.LittleEndian.PutUint32(sizeBytes, 2) // data size (1 sample * 2 bytes)
	buf = append(buf, sizeBytes...)
	sampleBytes := make([]byte, 2)
	binary.LittleEndian.PutUint16(sampleBytes, 0) // silent sample
	buf = append(buf, sampleBytes...)

	return buf
}
