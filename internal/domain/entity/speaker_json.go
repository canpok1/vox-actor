package entity

import (
	"encoding/json"
	"fmt"
)

// SpeakerJSON は speaker.json のトップレベル構造を表す。
type SpeakerJSON struct {
	SpeakerName string           `json:"speakerName"`
	Styles      []CharacterStyle `json:"styles"`
}

// CharacterStyle は speaker.json の styles 配列の各要素を表す。
type CharacterStyle struct {
	StyleName   string `json:"styleName"`
	MouthClosed string `json:"mouthClosed"`
	MouthOpened string `json:"mouthOpened"`
}

// ParseSpeakerJSON は JSON バイト列を SpeakerJSON にパースする。
func ParseSpeakerJSON(data []byte) (*SpeakerJSON, error) {
	var s SpeakerJSON
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("failed to parse speaker JSON: %w", err)
	}
	return &s, nil
}

// Validate は SpeakerJSON の必須フィールドを検証する。
func (s *SpeakerJSON) Validate() error {
	if s.SpeakerName == "" {
		return fmt.Errorf("speakerName is required")
	}
	for i, st := range s.Styles {
		if st.StyleName == "" {
			return fmt.Errorf("styles[%d].styleName is required", i)
		}
		if st.MouthClosed == "" {
			return fmt.Errorf("styles[%d].mouthClosed is required", i)
		}
		if st.MouthOpened == "" {
			return fmt.Errorf("styles[%d].mouthOpened is required", i)
		}
	}
	return nil
}
