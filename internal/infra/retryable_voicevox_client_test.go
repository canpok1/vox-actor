package infra_test

import (
	"testing"
	"time"

	"github.com/canpok1/vox-actor/internal/infra"
)

// TODO: 正常系: CreateQueryが1回で成功する場合リトライなしで結果を返す
// TODO: 正常系: Synthesizeが1回で成功する場合リトライなしで結果を返す
// TODO: リトライ成功: CreateQueryが1回失敗後に成功する場合リトライして結果を返す
// TODO: リトライ成功: Synthesizeが1回失敗後に成功する場合リトライして結果を返す
// TODO: リトライ上限超過: CreateQueryが最大3回リトライしても失敗する場合エラーを返す
// TODO: リトライ上限超過: Synthesizeが最大3回リトライしても失敗する場合エラーを返す
// TODO: HealthCheckはリトライ対象外: HealthCheckが失敗してもリトライせずエラーを返す
// TODO: 指数バックオフ: リトライ間隔が1秒→2秒→4秒で増加する
// TODO: バックオフ上限: 指数バックオフの上限が30秒である
// TODO: コンテキストキャンセル: リトライ中にコンテキストがキャンセルされた場合エラーを返す

func TestRetryableVoicevoxClient_DefaultConfig(t *testing.T) {
	inner := infra.NewVoicevoxClient("http://localhost")
	client := infra.NewRetryableVoicevoxClient(inner)

	config := client.RetryConfig()
	if config.MaxRetries != 3 {
		t.Errorf("expected MaxRetries 3, got %d", config.MaxRetries)
	}
	if config.InitialInterval != 1*time.Second {
		t.Errorf("expected InitialInterval 1s, got %v", config.InitialInterval)
	}
	if config.MaxInterval != 30*time.Second {
		t.Errorf("expected MaxInterval 30s, got %v", config.MaxInterval)
	}
}
