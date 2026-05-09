package infra

import (
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/ebitengine/oto/v3"
	"github.com/gopxl/beep/v2"
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

// --- otoSampleReader テスト ---

// fixedStreamer は固定サンプル列を返す beep.Streamer モック。
type fixedStreamer struct {
	samples [][2]float64
	err     error
	pos     int
}

func (s *fixedStreamer) Stream(buf [][2]float64) (int, bool) {
	n := copy(buf, s.samples[s.pos:])
	s.pos += n
	return n, n > 0
}

func (s *fixedStreamer) Err() error {
	return s.err
}

// TestOtoSampleReader_Read_ReturnsEOFWhenExhausted はストリーマー枯渇時に io.EOF を返すことを確認する。
func TestOtoSampleReader_Read_ReturnsEOFWhenExhausted(t *testing.T) {
	t.Parallel()

	r := &otoSampleReader{s: &fixedStreamer{samples: nil}}
	buf := make([]byte, 4)
	n, err := r.Read(buf)
	if n != 0 {
		t.Fatalf("expected 0 bytes, got %d", n)
	}
	if !errors.Is(err, io.EOF) {
		t.Fatalf("expected io.EOF, got %v", err)
	}
}

// TestOtoSampleReader_Read_EncodesSamplesAsInt16LE は float64 サンプルが int16 リトルエンディアンに正しくエンコードされることを確認する。
func TestOtoSampleReader_Read_EncodesSamplesAsInt16LE(t *testing.T) {
	t.Parallel()

	// サンプル値 1.0 → int16(32767) = 0x7FFF → little-endian: [0xFF, 0x7F]
	// サンプル値 0.0 → int16(0) = 0x0000 → little-endian: [0x00, 0x00]
	r := &otoSampleReader{s: &fixedStreamer{samples: [][2]float64{{1.0, 0.0}}}}
	buf := make([]byte, 4)
	n, err := r.Read(buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 4 {
		t.Fatalf("expected 4 bytes, got %d", n)
	}
	// ch0 = 1.0 → 0xFF 0x7F
	if buf[0] != 0xFF || buf[1] != 0x7F {
		t.Errorf("ch0: expected [0xFF, 0x7F], got [0x%02X, 0x%02X]", buf[0], buf[1])
	}
	// ch1 = 0.0 → 0x00 0x00
	if buf[2] != 0x00 || buf[3] != 0x00 {
		t.Errorf("ch1: expected [0x00, 0x00], got [0x%02X, 0x%02X]", buf[2], buf[3])
	}
}

// TestOtoSampleReader_Read_ClampsValuesOutsideRange は [-1, 1] 範囲外の値がクランプされることを確認する。
func TestOtoSampleReader_Read_ClampsValuesOutsideRange(t *testing.T) {
	t.Parallel()

	// サンプル値 2.0 → クランプ後 1.0 → int16(32767) = 0x7FFF
	// サンプル値 -2.0 → クランプ後 -1.0 → int16(-32767) = 0x8001 → little-endian: [0x01, 0x80]
	r := &otoSampleReader{s: &fixedStreamer{samples: [][2]float64{{2.0, -2.0}}}}
	buf := make([]byte, 4)
	n, err := r.Read(buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 4 {
		t.Fatalf("expected 4 bytes, got %d", n)
	}
	// ch0 = clamped 1.0 → 0xFF 0x7F
	if buf[0] != 0xFF || buf[1] != 0x7F {
		t.Errorf("ch0 (clamped 2.0): expected [0xFF, 0x7F], got [0x%02X, 0x%02X]", buf[0], buf[1])
	}
	// ch1 = clamped -1.0 → -32767 = 0x8001 → [0x01, 0x80]
	if buf[2] != 0x01 || buf[3] != 0x80 {
		t.Errorf("ch1 (clamped -2.0): expected [0x01, 0x80], got [0x%02X, 0x%02X]", buf[2], buf[3])
	}
}

// TestOtoSampleReader_Read_PropagatesStreamerError はストリーマーのエラーが伝播することを確認する。
func TestOtoSampleReader_Read_PropagatesStreamerError(t *testing.T) {
	t.Parallel()

	expectedErr := fmt.Errorf("audio decode failed")
	r := &otoSampleReader{s: &fixedStreamer{samples: nil, err: expectedErr}}
	buf := make([]byte, 4)
	_, err := r.Read(buf)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected %v, got %v", expectedErr, err)
	}
}

// TestOtoSampleReader_Read_ErrorOnUnalignedBuffer はサンプルアライメント外のバッファサイズでエラーを返すことを確認する。
func TestOtoSampleReader_Read_ErrorOnUnalignedBuffer(t *testing.T) {
	t.Parallel()

	r := &otoSampleReader{s: &fixedStreamer{}}
	buf := make([]byte, 3) // 4の倍数でない
	_, err := r.Read(buf)
	if err == nil {
		t.Fatal("expected error for unaligned buffer, got nil")
	}
}

// TestOtoSampleReader_Read_ImplementsIOReader はioReaderインターフェースを満たすことをコンパイル時に確認する。
func TestOtoSampleReader_Read_ImplementsIOReader(t *testing.T) {
	t.Parallel()

	var _ io.Reader = (*otoSampleReader)(nil)
}

// assertSampleRate は beep.Streamer のサンプルレートを確認するヘルパー。
var _ beep.Streamer = (*fixedStreamer)(nil)
