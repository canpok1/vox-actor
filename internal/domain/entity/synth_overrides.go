package entity

// SynthOverrides はセリフ単位の合成パラメータを保持する値オブジェクト。
// nilフィールドは「未指定（デフォルト値を使用）」を意味する。
type SynthOverrides struct {
	Speed      *float64
	Pitch      *float64
	Intonation *float64
}

// MergeWith はselfの値を優先し、nilフィールドはdefaultsの値を採用した新しいSynthOverridesを返す。
func (o SynthOverrides) MergeWith(defaults SynthOverrides) SynthOverrides {
	result := defaults
	if o.Speed != nil {
		result.Speed = o.Speed
	}
	if o.Pitch != nil {
		result.Pitch = o.Pitch
	}
	if o.Intonation != nil {
		result.Intonation = o.Intonation
	}
	return result
}
