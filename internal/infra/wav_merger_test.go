package infra_test

import (
	"encoding/binary"
	"testing"

	"github.com/canpok1/vox-actor/internal/infra"
)

// WavMerger テストリスト
// DONE: 正常系: 複数クリップを結合して1つのWAVを出力できる
// DONE: 正常系: gap=0 指定時は無音なしで連結される
// DONE: 正常系: gap>0 指定時はクリップ間に無音PCMが挿入される
// DONE: 正常系: 空クリップ（PCMデータなし）はスキップされる
// DONE: 正常系: クリップが1件だけの場合は結合できる
// DONE: 異常系: フォーマット不一致のクリップが含まれる場合はエラーを返す
// DONE: 異常系: 有効なクリップが0件の場合はエラーを返す

// makeTestWAV は指定フォーマットで指定バイト数のPCMデータを持つWAVバイト列を生成する。
func makeTestWAV(numChannels uint16, sampleRate uint32, bitsPerSample uint16, pcmData []byte) []byte {
	byteRate := sampleRate * uint32(numChannels) * uint32(bitsPerSample) / 8
	blockAlign := numChannels * bitsPerSample / 8

	fmtSize := uint32(16)
	dataSize := uint32(len(pcmData))
	riffSize := 4 + 8 + fmtSize + 8 + dataSize

	wav := make([]byte, 12+8+fmtSize+8+dataSize)
	copy(wav[0:4], "RIFF")
	binary.LittleEndian.PutUint32(wav[4:8], riffSize)
	copy(wav[8:12], "WAVE")

	// fmt chunk
	copy(wav[12:16], "fmt ")
	binary.LittleEndian.PutUint32(wav[16:20], fmtSize)
	binary.LittleEndian.PutUint16(wav[20:22], 1) // PCM
	binary.LittleEndian.PutUint16(wav[22:24], numChannels)
	binary.LittleEndian.PutUint32(wav[24:28], sampleRate)
	binary.LittleEndian.PutUint32(wav[28:32], byteRate)
	binary.LittleEndian.PutUint16(wav[32:34], blockAlign)
	binary.LittleEndian.PutUint16(wav[34:36], bitsPerSample)

	// data chunk
	copy(wav[36:40], "data")
	binary.LittleEndian.PutUint32(wav[40:44], dataSize)
	copy(wav[44:], pcmData)

	return wav
}

// extractPCMData は WAV バイト列から data チャンクの PCM データを取り出す。
func extractPCMData(t *testing.T, wav []byte) []byte {
	t.Helper()
	if len(wav) < 44 {
		t.Fatalf("wav too short: %d bytes", len(wav))
	}
	dataSize := binary.LittleEndian.Uint32(wav[40:44])
	if int(44+dataSize) > len(wav) {
		t.Fatalf("data chunk claims %d bytes but wav is only %d bytes", dataSize, len(wav))
	}
	return wav[44 : 44+dataSize]
}

func TestWavMerger_Merge_MultipleClips(t *testing.T) {
	pcm1 := []byte{0x01, 0x02, 0x03, 0x04}
	pcm2 := []byte{0x05, 0x06, 0x07, 0x08}
	clip1 := makeTestWAV(1, 24000, 16, pcm1)
	clip2 := makeTestWAV(1, 24000, 16, pcm2)

	merger := infra.NewWavMerger()
	result, err := merger.Merge([][]byte{clip1, clip2}, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := extractPCMData(t, result)
	expected := append(pcm1, pcm2...)
	if string(got) != string(expected) {
		t.Errorf("expected PCM %v, got %v", expected, got)
	}
}

func TestWavMerger_Merge_GapZero_NoSilenceInserted(t *testing.T) {
	pcm1 := []byte{0x01, 0x02}
	pcm2 := []byte{0x03, 0x04}
	clip1 := makeTestWAV(1, 24000, 16, pcm1)
	clip2 := makeTestWAV(1, 24000, 16, pcm2)

	merger := infra.NewWavMerger()
	result, err := merger.Merge([][]byte{clip1, clip2}, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := extractPCMData(t, result)
	// gap=0: pcm1 + pcm2 のみ（無音なし）
	if len(got) != len(pcm1)+len(pcm2) {
		t.Errorf("expected PCM length %d, got %d", len(pcm1)+len(pcm2), len(got))
	}
}

func TestWavMerger_Merge_GapPositive_SilenceInserted(t *testing.T) {
	// 24000 Hz, 16bit, 1ch → 1ms = 24000/1000 * 2 = 48 bytes
	pcm1 := make([]byte, 48)
	pcm2 := make([]byte, 48)
	for i := range pcm1 {
		pcm1[i] = 0x01
		pcm2[i] = 0x02
	}
	clip1 := makeTestWAV(1, 24000, 16, pcm1)
	clip2 := makeTestWAV(1, 24000, 16, pcm2)

	gapMs := 10
	merger := infra.NewWavMerger()
	result, err := merger.Merge([][]byte{clip1, clip2}, gapMs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := extractPCMData(t, result)
	// gap PCM bytes = 24000 * 1 * 2 / 1000 * 10 = 480 bytes
	gapBytes := 24000 * 1 * 2 / 1000 * gapMs
	expectedLen := len(pcm1) + gapBytes + len(pcm2)
	if len(got) != expectedLen {
		t.Errorf("expected PCM length %d (gap=%d bytes), got %d", expectedLen, gapBytes, len(got))
	}

	// gap 部分はゼロバイトであることを確認
	gapStart := len(pcm1)
	for i := gapStart; i < gapStart+gapBytes; i++ {
		if got[i] != 0x00 {
			t.Errorf("expected silence (0x00) at byte %d, got 0x%02x", i, got[i])
			break
		}
	}
}

func TestWavMerger_Merge_EmptyClipsSkipped(t *testing.T) {
	pcm1 := []byte{0x01, 0x02, 0x03, 0x04}
	emptyPCM := []byte{}
	pcm2 := []byte{0x05, 0x06, 0x07, 0x08}
	clip1 := makeTestWAV(1, 24000, 16, pcm1)
	emptyClip := makeTestWAV(1, 24000, 16, emptyPCM)
	clip2 := makeTestWAV(1, 24000, 16, pcm2)

	merger := infra.NewWavMerger()
	result, err := merger.Merge([][]byte{clip1, emptyClip, clip2}, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := extractPCMData(t, result)
	expected := append(pcm1, pcm2...)
	if string(got) != string(expected) {
		t.Errorf("expected PCM %v (empty clip skipped), got %v", expected, got)
	}
}

func TestWavMerger_Merge_SingleClip(t *testing.T) {
	pcm := []byte{0x01, 0x02, 0x03, 0x04}
	clip := makeTestWAV(2, 44100, 16, pcm)

	merger := infra.NewWavMerger()
	result, err := merger.Merge([][]byte{clip}, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := extractPCMData(t, result)
	if string(got) != string(pcm) {
		t.Errorf("expected PCM %v, got %v", pcm, got)
	}
}

func TestWavMerger_Merge_FormatMismatch_ReturnsError(t *testing.T) {
	clip1 := makeTestWAV(1, 24000, 16, []byte{0x01, 0x02})
	clip2 := makeTestWAV(2, 44100, 16, []byte{0x03, 0x04}) // 異なるチャンネル数・サンプルレート

	merger := infra.NewWavMerger()
	_, err := merger.Merge([][]byte{clip1, clip2}, 0)
	if err == nil {
		t.Fatal("expected error for format mismatch, got nil")
	}
}

func TestWavMerger_Merge_NoValidClips_ReturnsError(t *testing.T) {
	merger := infra.NewWavMerger()
	_, err := merger.Merge([][]byte{}, 0)
	if err == nil {
		t.Fatal("expected error for empty input, got nil")
	}
}
