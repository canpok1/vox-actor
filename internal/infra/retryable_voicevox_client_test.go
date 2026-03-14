package infra_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/canpok1/vox-actor/internal/domain/entity"
	"github.com/canpok1/vox-actor/internal/infra"
)

// TODO: 正常系: CreateQueryが1回で成功する場合リトライなしで結果を返す
// TODO: 正常系: Synthesizeが1回で成功する場合リトライなしで結果を返す
// TODO: リトライ成功: CreateQueryが1回失敗後に成功する場合リトライして結果を返す
// TODO: リトライ成功: Synthesizeが1回失敗後に成功する場合リトライして結果を返す
// TODO: リトライ上限超過: CreateQueryが最大3回リトライしても失敗する場合エラーを返す
// TODO: リトライ上限超過: Synthesizeが最大3回リトライしても失敗する場合エラーを返す
// TODO: 指数バックオフ: リトライ間隔が1秒→2秒→4秒で増加する
// TODO: バックオフ上限: 指数バックオフの上限が30秒である
// TODO: コンテキストキャンセル: リトライ中にコンテキストがキャンセルされた場合エラーを返す

// mockVoicevoxClient はテスト用のVoicevoxClientモック。
type mockVoicevoxClient struct {
	healthCheckFunc func(ctx context.Context) error
	createQueryFunc func(ctx context.Context, text string, speakerID int) (*entity.AudioQuery, error)
	synthesizeFunc  func(ctx context.Context, query *entity.AudioQuery, speakerID int, speed, pitch, intonation *float64) ([]byte, error)
	callCounts      map[string]int
}

func newMockVoicevoxClient() *mockVoicevoxClient {
	return &mockVoicevoxClient{
		callCounts: make(map[string]int),
	}
}

func (m *mockVoicevoxClient) HealthCheck(ctx context.Context) error {
	m.callCounts["HealthCheck"]++
	if m.healthCheckFunc != nil {
		return m.healthCheckFunc(ctx)
	}
	return nil
}

func (m *mockVoicevoxClient) CreateQuery(ctx context.Context, text string, speakerID int) (*entity.AudioQuery, error) {
	m.callCounts["CreateQuery"]++
	if m.createQueryFunc != nil {
		return m.createQueryFunc(ctx, text, speakerID)
	}
	return &entity.AudioQuery{}, nil
}

func (m *mockVoicevoxClient) Synthesize(ctx context.Context, query *entity.AudioQuery, speakerID int, speed, pitch, intonation *float64) ([]byte, error) {
	m.callCounts["Synthesize"]++
	if m.synthesizeFunc != nil {
		return m.synthesizeFunc(ctx, query, speakerID, speed, pitch, intonation)
	}
	return []byte("wav"), nil
}

// DONE: デフォルト設定: リトライ設定のデフォルト値が仕様通りである（最大3回、初期1秒、上限30秒）
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

// DONE: HealthCheckはリトライ対象外: HealthCheckが失敗してもリトライせずエラーを返す
func TestRetryableVoicevoxClient_HealthCheck_NoRetry(t *testing.T) {
	mock := newMockVoicevoxClient()
	mock.healthCheckFunc = func(_ context.Context) error {
		return errors.New("connection refused")
	}
	client := infra.NewRetryableVoicevoxClient(mock)

	err := client.HealthCheck(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if mock.callCounts["HealthCheck"] != 1 {
		t.Errorf("expected HealthCheck to be called 1 time, got %d", mock.callCounts["HealthCheck"])
	}
}

// リトライ成功: CreateQueryが1回失敗後に成功する場合リトライして結果を返す
func TestRetryableVoicevoxClient_CreateQuery_RetrySuccess(t *testing.T) {
	mock := newMockVoicevoxClient()
	callCount := 0
	expectedQuery := &entity.AudioQuery{SpeedScale: 1.0}
	mock.createQueryFunc = func(_ context.Context, _ string, _ int) (*entity.AudioQuery, error) {
		callCount++
		if callCount == 1 {
			return nil, errors.New("connection refused")
		}
		return expectedQuery, nil
	}

	noopSleep := func(_ context.Context, _ time.Duration) error { return nil }
	client := infra.NewRetryableVoicevoxClient(mock, infra.WithSleepFunc(noopSleep))

	query, err := client.CreateQuery(context.Background(), "test", 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if query != expectedQuery {
		t.Errorf("expected query %v, got %v", expectedQuery, query)
	}
	if mock.callCounts["CreateQuery"] != 2 {
		t.Errorf("expected CreateQuery to be called 2 times, got %d", mock.callCounts["CreateQuery"])
	}
}
