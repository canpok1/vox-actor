package entity

// AudioQuery はVOICEVOX音声合成用のクエリを表す。
type AudioQuery struct {
	AccentPhrases      []AccentPhrase `json:"accent_phrases"`
	SpeedScale         float64        `json:"speedScale"`
	PitchScale         float64        `json:"pitchScale"`
	IntonationScale    float64        `json:"intonationScale"`
	VolumeScale        float64        `json:"volumeScale"`
	PrePhonemeLength   float64        `json:"prePhonemeLength"`
	PostPhonemeLength  float64        `json:"postPhonemeLength"`
	PauseLength        *float64       `json:"pauseLength"`
	PauseLengthScale   float64        `json:"pauseLengthScale"`
	OutputSamplingRate int            `json:"outputSamplingRate"`
	OutputStereo       bool           `json:"outputStereo"`
	Kana               string         `json:"kana"`
}

// AccentPhrase はアクセント句を表す。
type AccentPhrase struct {
	Moras           []Mora `json:"moras"`
	Accent          int    `json:"accent"`
	PauseMora       *Mora  `json:"pause_mora"`
	IsInterrogative bool   `json:"is_interrogative"`
}

// WithOverrides は指定されたパラメータでAudioQueryのフィールドを上書きした新しいAudioQueryを返す。
// nilのパラメータは上書きしない。値レシーバのため元のAudioQueryは変更されない。
func (q AudioQuery) WithOverrides(speed, pitch, intonation *float64) AudioQuery {
	if speed != nil {
		q.SpeedScale = *speed
	}
	if pitch != nil {
		q.PitchScale = *pitch
	}
	if intonation != nil {
		q.IntonationScale = *intonation
	}
	return q
}

// Mora はモーラ（音節）を表す。
type Mora struct {
	Text            string   `json:"text"`
	Consonant       *string  `json:"consonant"`
	ConsonantLength *float64 `json:"consonant_length"`
	Vowel           string   `json:"vowel"`
	VowelLength     float64  `json:"vowel_length"`
	Pitch           float64  `json:"pitch"`
}
