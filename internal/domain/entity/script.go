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
	// Overrides はセリフ単位の合成パラメータ（nilフィールドの場合はデフォルト値を使用）。
	Overrides SynthOverrides
}

// ResolveSpeakerID はセリフ単位のSpeakerIDがあればそれを、なければデフォルト値を返す。
func (s Script) ResolveSpeakerID(defaultID int) int {
	if s.SpeakerID != nil {
		return *s.SpeakerID
	}
	return defaultID
}

// ResolveOverrides はセリフ単位のOverridesを優先し、nilフィールドはdefaultsで補完した値を返す。
func (s Script) ResolveOverrides(defaults SynthOverrides) SynthOverrides {
	return s.Overrides.MergeWith(defaults)
}
