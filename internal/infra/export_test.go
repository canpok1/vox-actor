package infra

import "time"

// ClientTimeout はテスト用にHTTPクライアントのタイムアウト値を返す。
func (c *VoicevoxClient) ClientTimeout() time.Duration {
	return c.httpClient.Timeout
}

// PlaybackStatus はテスト用に playback state の状態文字列と存在可否を返す。
func (p *HTTPStreamPlayer) PlaybackStatus(id string) (string, bool) {
	p.playbackMu.Lock()
	defer p.playbackMu.Unlock()
	state, ok := p.playbacks[id]
	if !ok {
		return "", false
	}
	return string(state.status), true
}

// PrunePlaybacks はテスト用に TTL 切れの playback state を削除する。
func (p *HTTPStreamPlayer) PrunePlaybacks() {
	p.prunePlaybacks()
}
