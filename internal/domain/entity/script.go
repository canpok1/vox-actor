package entity

// Script は読み込んだ台本テキストを表す。
type Script struct {
	// Path はファイルのパス。
	Path string
	// Text はファイルの内容。
	Text string
	// IsEmpty はファイルが空であるかを示す。
	IsEmpty bool
	// SpeakerID はセリフ単位のキャラクターID（nilの場合はデフォルト値を使用）。
	SpeakerID *int
	// SpeedScale はセリフ単位の話速（nilの場合はデフォルト値を使用）。
	SpeedScale *float64
	// PitchScale はセリフ単位の音高（nilの場合はデフォルト値を使用）。
	PitchScale *float64
	// IntonationScale はセリフ単位の抑揚（nilの場合はデフォルト値を使用）。
	IntonationScale *float64
}

// ResolveSpeakerID はセリフ単位のSpeakerIDがあればそれを、なければデフォルト値を返す。
func (s Script) ResolveSpeakerID(defaultID int) int {
	if s.SpeakerID != nil {
		return *s.SpeakerID
	}
	return defaultID
}

// ResolveSpeed はセリフ単位のSpeedScaleがあればそれを、なければデフォルト値を返す。
func (s Script) ResolveSpeed(defaultSpeed *float64) *float64 {
	if s.SpeedScale != nil {
		return s.SpeedScale
	}
	return defaultSpeed
}

// ResolvePitch はセリフ単位のPitchScaleがあればそれを、なければデフォルト値を返す。
func (s Script) ResolvePitch(defaultPitch *float64) *float64 {
	if s.PitchScale != nil {
		return s.PitchScale
	}
	return defaultPitch
}

// ResolveIntonation はセリフ単位のIntonationScaleがあればそれを、なければデフォルト値を返す。
func (s Script) ResolveIntonation(defaultIntonation *float64) *float64 {
	if s.IntonationScale != nil {
		return s.IntonationScale
	}
	return defaultIntonation
}
