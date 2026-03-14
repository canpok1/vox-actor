package infra

import "time"

// ClientTimeout はテスト用にHTTPクライアントのタイムアウト値を返す。
func (c *VoicevoxClient) ClientTimeout() time.Duration {
	return c.httpClient.Timeout
}
