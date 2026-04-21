package infra

import (
	"bytes"
	"encoding/binary"
	"testing"
	"time"
)

// buildWAV はテスト用の最小WAVバイト列を生成する。
// byteRate と dataSize を指定することで、推定duration を制御できる。
func buildWAV(byteRate uint32, dataSize uint32) []byte {
	buf := &bytes.Buffer{}
	buf.WriteString("RIFF")
	_ = binary.Write(buf, binary.LittleEndian, 36+dataSize)
	buf.WriteString("WAVE")
	buf.WriteString("fmt ")
	_ = binary.Write(buf, binary.LittleEndian, uint32(16))    // fmt chunk size
	_ = binary.Write(buf, binary.LittleEndian, uint16(1))     // PCM
	_ = binary.Write(buf, binary.LittleEndian, uint16(1))     // mono
	_ = binary.Write(buf, binary.LittleEndian, uint32(24000)) // sample rate
	_ = binary.Write(buf, binary.LittleEndian, byteRate)
	_ = binary.Write(buf, binary.LittleEndian, uint16(2))  // block align
	_ = binary.Write(buf, binary.LittleEndian, uint16(16)) // bits per sample
	buf.WriteString("data")
	_ = binary.Write(buf, binary.LittleEndian, dataSize)
	buf.Write(make([]byte, dataSize))
	return buf.Bytes()
}

func TestEstimateWAVDuration_ValidHeader(t *testing.T) {
	t.Parallel()
	// byteRate=48000, dataSize=96000 → duration=2秒
	wavData := buildWAV(48000, 96000)

	got, err := estimateWAVDuration(wavData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := 2 * time.Second
	if got != want {
		t.Errorf("estimateWAVDuration: got %v, want %v", got, want)
	}
}

func TestEstimateWAVDuration_FractionalSeconds(t *testing.T) {
	t.Parallel()
	// byteRate=48000, dataSize=12000 → duration=0.25秒 = 250ms
	wavData := buildWAV(48000, 12000)

	got, err := estimateWAVDuration(wavData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := 250 * time.Millisecond
	if got != want {
		t.Errorf("estimateWAVDuration: got %v, want %v", got, want)
	}
}

func TestEstimateWAVDuration_InvalidRIFFHeader(t *testing.T) {
	t.Parallel()
	wavData := []byte("NOT_A_WAV")

	if _, err := estimateWAVDuration(wavData); err == nil {
		t.Fatal("expected error for non-RIFF header, got nil")
	}
}

func TestEstimateWAVDuration_MissingWAVE(t *testing.T) {
	t.Parallel()
	// RIFF header but no WAVE marker
	buf := &bytes.Buffer{}
	buf.WriteString("RIFF")
	_ = binary.Write(buf, binary.LittleEndian, uint32(4))
	buf.WriteString("XXXX")

	if _, err := estimateWAVDuration(buf.Bytes()); err == nil {
		t.Fatal("expected error for missing WAVE marker, got nil")
	}
}

func TestEstimateWAVDuration_MissingFmtChunk(t *testing.T) {
	t.Parallel()
	// RIFF/WAVE ヘッダのみで fmt/data がない
	buf := &bytes.Buffer{}
	buf.WriteString("RIFF")
	_ = binary.Write(buf, binary.LittleEndian, uint32(4))
	buf.WriteString("WAVE")

	if _, err := estimateWAVDuration(buf.Bytes()); err == nil {
		t.Fatal("expected error for missing fmt chunk, got nil")
	}
}

func TestEstimateWAVDuration_MissingDataChunk(t *testing.T) {
	t.Parallel()
	// fmt chunk まではあるが data chunk がない
	buf := &bytes.Buffer{}
	buf.WriteString("RIFF")
	_ = binary.Write(buf, binary.LittleEndian, uint32(28))
	buf.WriteString("WAVE")
	buf.WriteString("fmt ")
	_ = binary.Write(buf, binary.LittleEndian, uint32(16))
	_ = binary.Write(buf, binary.LittleEndian, uint16(1))
	_ = binary.Write(buf, binary.LittleEndian, uint16(1))
	_ = binary.Write(buf, binary.LittleEndian, uint32(24000))
	_ = binary.Write(buf, binary.LittleEndian, uint32(48000))
	_ = binary.Write(buf, binary.LittleEndian, uint16(2))
	_ = binary.Write(buf, binary.LittleEndian, uint16(16))

	if _, err := estimateWAVDuration(buf.Bytes()); err == nil {
		t.Fatal("expected error for missing data chunk, got nil")
	}
}

func TestEstimateWAVDuration_ZeroByteRate(t *testing.T) {
	t.Parallel()
	// byteRate=0 は不正
	wavData := buildWAV(0, 100)

	if _, err := estimateWAVDuration(wavData); err == nil {
		t.Fatal("expected error for zero byte rate, got nil")
	}
}

func TestEstimateWAVDuration_SkipsUnknownChunks(t *testing.T) {
	t.Parallel()
	// fmt の後、data の前に LIST チャンクが挟まっているケース
	buf := &bytes.Buffer{}
	buf.WriteString("RIFF")
	_ = binary.Write(buf, binary.LittleEndian, uint32(52))
	buf.WriteString("WAVE")
	// fmt
	buf.WriteString("fmt ")
	_ = binary.Write(buf, binary.LittleEndian, uint32(16))
	_ = binary.Write(buf, binary.LittleEndian, uint16(1))
	_ = binary.Write(buf, binary.LittleEndian, uint16(1))
	_ = binary.Write(buf, binary.LittleEndian, uint32(24000))
	_ = binary.Write(buf, binary.LittleEndian, uint32(48000)) // byteRate
	_ = binary.Write(buf, binary.LittleEndian, uint16(2))
	_ = binary.Write(buf, binary.LittleEndian, uint16(16))
	// LIST
	buf.WriteString("LIST")
	_ = binary.Write(buf, binary.LittleEndian, uint32(4))
	buf.Write([]byte{0, 0, 0, 0})
	// data
	buf.WriteString("data")
	_ = binary.Write(buf, binary.LittleEndian, uint32(24000))
	buf.Write(make([]byte, 24000))

	got, err := estimateWAVDuration(buf.Bytes())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := 500 * time.Millisecond
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}
