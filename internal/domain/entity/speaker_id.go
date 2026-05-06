package entity

import "fmt"

// SpeakerID はVOICEVOXのスタイルIDを表す値オブジェクト。非負を保証する。
type SpeakerID struct {
	value int
}

// NewSpeakerID はSpeakerIDを生成する。負値の場合はエラーを返す。
func NewSpeakerID(v int) (SpeakerID, error) {
	if v < 0 {
		return SpeakerID{}, fmt.Errorf("speakerID must be non-negative, got %d", v)
	}
	return SpeakerID{value: v}, nil
}

// MustNewSpeakerID はSpeakerIDを生成する。負値の場合はパニックする。
func MustNewSpeakerID(v int) SpeakerID {
	id, err := NewSpeakerID(v)
	if err != nil {
		panic(err)
	}
	return id
}

// Value は内部の整数値を返す。
func (id SpeakerID) Value() int {
	return id.value
}
