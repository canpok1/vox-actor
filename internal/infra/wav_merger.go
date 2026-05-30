package infra

import (
	"encoding/binary"
	"fmt"

	"github.com/canpok1/vox-actor/internal/app"
)

// wavFormat は WAV の fmt チャンクから読み取ったフォーマット情報。
type wavFormat struct {
	audioFormat   uint16
	numChannels   uint16
	sampleRate    uint32
	byteRate      uint32
	blockAlign    uint16
	bitsPerSample uint16
}

// WavMerger は複数の WAV バイト列を1つに結合する。
// app.WavMerger インターフェースを実装する。
type WavMerger struct{}

var _ app.WavMerger = (*WavMerger)(nil)

// NewWavMerger は WavMerger を生成する。
func NewWavMerger() *WavMerger {
	return &WavMerger{}
}

// Merge は複数の WAV バイト列を結合して1つの WAV を返す。
// gapMs > 0 の場合はクリップ間に無音 PCM を挿入する。
// 空クリップ（PCM データなし）はスキップする。
// フォーマットが異なるクリップが含まれる場合はエラーを返す。
// 有効なクリップが0件の場合はエラーを返す。
func (m *WavMerger) Merge(wavClips [][]byte, gapMs int) ([]byte, error) {
	type parsedClip struct {
		format  wavFormat
		pcmData []byte
	}

	var clips []parsedClip
	for i, wav := range wavClips {
		format, pcmData, err := parseWavChunks(wav)
		if err != nil {
			return nil, fmt.Errorf("clip[%d]: %w", i, err)
		}
		if len(pcmData) == 0 {
			continue
		}
		clips = append(clips, parsedClip{format: format, pcmData: pcmData})
	}

	if len(clips) == 0 {
		return nil, fmt.Errorf("no valid (non-empty) WAV clips to merge")
	}

	base := clips[0].format
	for i, c := range clips[1:] {
		if err := assertSameFormat(base, c.format, i+1); err != nil {
			return nil, err
		}
	}

	var gapPCM []byte
	if gapMs > 0 {
		gapBytes := int(base.byteRate) * gapMs / 1000
		gapPCM = make([]byte, gapBytes)
	}

	totalSize := 0
	for _, c := range clips {
		totalSize += len(c.pcmData)
	}
	if len(clips) > 1 {
		totalSize += len(gapPCM) * (len(clips) - 1)
	}

	pcmBuf := make([]byte, 0, totalSize)
	for i, c := range clips {
		if i > 0 {
			pcmBuf = append(pcmBuf, gapPCM...)
		}
		pcmBuf = append(pcmBuf, c.pcmData...)
	}

	return assembleWAV(base, pcmBuf), nil
}

// parseWavChunks は WAV バイト列を解析して fmt フォーマットと data PCM を返す。
func parseWavChunks(wav []byte) (wavFormat, []byte, error) {
	const riffHeaderSize = 12
	if len(wav) < riffHeaderSize {
		return wavFormat{}, nil, fmt.Errorf("WAV data too short: %d bytes", len(wav))
	}
	if string(wav[0:4]) != "RIFF" {
		return wavFormat{}, nil, fmt.Errorf("not a RIFF file")
	}
	if string(wav[8:12]) != "WAVE" {
		return wavFormat{}, nil, fmt.Errorf("not a WAVE file")
	}

	var fmt_ wavFormat
	var pcmData []byte
	var foundFmt, foundData bool

	offset := riffHeaderSize
	for offset+8 <= len(wav) {
		chunkID := string(wav[offset : offset+4])
		chunkSize := binary.LittleEndian.Uint32(wav[offset+4 : offset+8])
		bodyStart := offset + 8
		bodyEnd := min(bodyStart+int(chunkSize), len(wav))

		switch chunkID {
		case "fmt ":
			if bodyEnd-bodyStart < 16 {
				return wavFormat{}, nil, fmt.Errorf("fmt chunk too small")
			}
			fmt_.audioFormat = binary.LittleEndian.Uint16(wav[bodyStart : bodyStart+2])
			fmt_.numChannels = binary.LittleEndian.Uint16(wav[bodyStart+2 : bodyStart+4])
			fmt_.sampleRate = binary.LittleEndian.Uint32(wav[bodyStart+4 : bodyStart+8])
			fmt_.byteRate = binary.LittleEndian.Uint32(wav[bodyStart+8 : bodyStart+12])
			fmt_.blockAlign = binary.LittleEndian.Uint16(wav[bodyStart+12 : bodyStart+14])
			fmt_.bitsPerSample = binary.LittleEndian.Uint16(wav[bodyStart+14 : bodyStart+16])
			foundFmt = true
		case "data":
			pcmData = wav[bodyStart:bodyEnd]
			foundData = true
		}

		if foundFmt && foundData {
			break
		}
		next := bodyStart + int(chunkSize)
		if chunkSize%2 == 1 {
			next++
		}
		offset = next
	}

	if !foundFmt {
		return wavFormat{}, nil, fmt.Errorf("fmt chunk not found")
	}
	if !foundData {
		return wavFormat{}, nil, fmt.Errorf("data chunk not found")
	}
	return fmt_, pcmData, nil
}

// assertSameFormat は2つのフォーマットが同一であることを確認する。
func assertSameFormat(base, other wavFormat, index int) error {
	if base.numChannels != other.numChannels {
		return fmt.Errorf("clip[%d] numChannels mismatch: want %d, got %d", index, base.numChannels, other.numChannels)
	}
	if base.sampleRate != other.sampleRate {
		return fmt.Errorf("clip[%d] sampleRate mismatch: want %d, got %d", index, base.sampleRate, other.sampleRate)
	}
	if base.bitsPerSample != other.bitsPerSample {
		return fmt.Errorf("clip[%d] bitsPerSample mismatch: want %d, got %d", index, base.bitsPerSample, other.bitsPerSample)
	}
	if base.audioFormat != other.audioFormat {
		return fmt.Errorf("clip[%d] audioFormat mismatch: want %d, got %d", index, base.audioFormat, other.audioFormat)
	}
	return nil
}

// assembleWAV は fmt フォーマットと PCM データから WAV バイト列を構築する。
func assembleWAV(fmt_ wavFormat, pcmData []byte) []byte {
	const fmtChunkSize = 16
	dataSize := uint32(len(pcmData))
	riffSize := 4 + 8 + fmtChunkSize + 8 + dataSize

	buf := make([]byte, 12+8+fmtChunkSize+8+dataSize)

	copy(buf[0:4], "RIFF")
	binary.LittleEndian.PutUint32(buf[4:8], riffSize)
	copy(buf[8:12], "WAVE")

	copy(buf[12:16], "fmt ")
	binary.LittleEndian.PutUint32(buf[16:20], fmtChunkSize)
	binary.LittleEndian.PutUint16(buf[20:22], fmt_.audioFormat)
	binary.LittleEndian.PutUint16(buf[22:24], fmt_.numChannels)
	binary.LittleEndian.PutUint32(buf[24:28], fmt_.sampleRate)
	binary.LittleEndian.PutUint32(buf[28:32], fmt_.byteRate)
	binary.LittleEndian.PutUint16(buf[32:34], fmt_.blockAlign)
	binary.LittleEndian.PutUint16(buf[34:36], fmt_.bitsPerSample)

	copy(buf[36:40], "data")
	binary.LittleEndian.PutUint32(buf[40:44], dataSize)
	copy(buf[44:], pcmData)

	return buf
}
