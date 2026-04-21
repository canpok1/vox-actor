package infra

import (
	"encoding/binary"
	"fmt"
	"time"
)

// estimateWAVDuration はWAVバイト列のヘッダから推定再生時間を計算する。
// duration = data chunk のバイト数 / fmt chunk の ByteRate
// ヘッダが不正な場合や ByteRate=0 の場合はエラーを返す。
func estimateWAVDuration(wavData []byte) (time.Duration, error) {
	const (
		riffHeaderSize    = 12 // "RIFF" + size + "WAVE"
		fmtChunkMinBody   = 16 // PCM fmt chunk body の最小サイズ
		fmtByteRateOffset = 8  // fmt chunk body 内の ByteRate フィールドオフセット
	)
	if len(wavData) < riffHeaderSize {
		return 0, fmt.Errorf("WAV data too short: %d bytes", len(wavData))
	}
	if string(wavData[0:4]) != "RIFF" {
		return 0, fmt.Errorf("not a RIFF file")
	}
	if string(wavData[8:12]) != "WAVE" {
		return 0, fmt.Errorf("not a WAVE file")
	}

	var byteRate uint32
	var dataSize uint32
	var foundFmt, foundData bool

	offset := riffHeaderSize
	for offset+8 <= len(wavData) {
		chunkID := string(wavData[offset : offset+4])
		chunkSize := binary.LittleEndian.Uint32(wavData[offset+4 : offset+8])
		bodyStart := offset + 8
		bodyEnd := min(bodyStart+int(chunkSize), len(wavData))

		switch chunkID {
		case "fmt ":
			if bodyEnd-bodyStart < fmtChunkMinBody {
				return 0, fmt.Errorf("fmt chunk too small")
			}
			byteRate = binary.LittleEndian.Uint32(wavData[bodyStart+fmtByteRateOffset : bodyStart+fmtByteRateOffset+4])
			foundFmt = true
		case "data":
			dataSize = chunkSize
			foundData = true
		}

		if foundFmt && foundData {
			break
		}
		offset = bodyStart + int(chunkSize)
		if chunkSize%2 == 1 {
			offset++ // RIFF の chunk は偶数バイト境界に揃える
		}
	}

	if !foundFmt {
		return 0, fmt.Errorf("fmt chunk not found")
	}
	if !foundData {
		return 0, fmt.Errorf("data chunk not found")
	}
	if byteRate == 0 {
		return 0, fmt.Errorf("invalid byte rate: 0")
	}

	seconds := float64(dataSize) / float64(byteRate)
	return time.Duration(seconds * float64(time.Second)), nil
}
