package entity

// Clip はドメイン層の再生1単位を表す値オブジェクト。
// HTTP 境界では infra パッケージの ViewerClip が JSON 変換を担う。
type Clip struct {
	Text       string
	SpeakerID  SpeakerID
	Speed      *float64
	Pitch      *float64
	Intonation *float64
}

// NewClip は Clip を生成する。
func NewClip(text string, speakerID SpeakerID, speed, pitch, intonation *float64) Clip {
	return Clip{
		Text:       text,
		SpeakerID:  speakerID,
		Speed:      speed,
		Pitch:      pitch,
		Intonation: intonation,
	}
}
