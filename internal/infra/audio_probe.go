package infra

// otoAudioProbe は oto を直接使う audioProbe 実装。
// beep/speaker は oto.Context 生成後のエラーを外部に公開しないため、
// デバイス不在を正しく検知するには oto を直接扱う必要がある。
// getOtoContext でシングルトンを共有することで、beep/speaker.Init との二重生成を回避する。
type otoAudioProbe struct{}

// NewOtoAudioProbe は oto を使う audioProbe 実装を生成する。
func NewOtoAudioProbe() *otoAudioProbe {
	return &otoAudioProbe{}
}

// Probe は共有 oto.Context の生成を試みてデバイス可用性を確認する。
// Context はプロセス内でシングルトンとして保持され、beep/speaker との衝突を防ぐ。
func (p *otoAudioProbe) Probe() error {
	_, err := getOtoContext()
	return err
}
