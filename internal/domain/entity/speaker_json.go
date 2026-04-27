package entity

import (
	"encoding/json"
	"fmt"
)

// SpeakerJSON は speaker.json のトップレベル構造を表す。
type SpeakerJSON struct {
	Name    string            `json:"name"`
	Profile *CharacterProfile `json:"profile,omitempty"`
	Styles  []CharacterStyle  `json:"styles"`
	// 互換性のため、古いスキーマの speakerName も保持
	SpeakerName string `json:"speakerName"`
}

// CharacterProfile は speaker.json の profile フィールドを表す。
type CharacterProfile struct {
	Pronoun         string         `json:"pronoun"`
	SpeechSuffix    []string       `json:"speechSuffix"`
	Personality     []string       `json:"personality"`
	Speakers        map[string]int `json:"speakers"`
	DescriptionPath string         `json:"descriptionPath"`
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

// GetSpeakerName は speaker 名を返す。新スキーマの Name を優先し、互換性のため古い SpeakerName にフォールバックする。
func (s *SpeakerJSON) GetSpeakerName() string {
	if s.Name != "" {
		return s.Name
	}
	return s.SpeakerName
}

// Validate は SpeakerJSON の必須フィールドを検証する。
func (s *SpeakerJSON) Validate() error {
	if s.GetSpeakerName() == "" {
		return fmt.Errorf("name or speakerName is required")
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
