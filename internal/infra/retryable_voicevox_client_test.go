package infra_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/canpok1/vox-actor/internal/domain/entity"
	"github.com/canpok1/vox-actor/internal/infra"
)

// mockNetError はnet.Errorインターフェースを実装するテスト用エラー（リトライ対象）。
type mockNetError struct {
	msg string
}

func (e *mockNetError) Error() string   { return e.msg }
func (e *mockNetError) Timeout() bool   { return false }
func (e *mockNetError) Temporary() bool { return true }

// mockVoicevoxClient はテスト用のVoicevoxClientモック。
type mockVoicevoxClient struct {
	healthCheckFunc func(ctx context.Context) error
	createQueryFunc func(ctx context.Context, text string, speakerID int) (*entity.AudioQuery, error)
	synthesizeFunc  func(ctx context.Context, query *entity.AudioQuery, speakerID int) ([]byte, error)
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

func (m *mockVoicevoxClient) Synthesize(ctx context.Context, query *entity.AudioQuery, speakerID int) ([]byte, error) {
	m.callCounts["Synthesize"]++
	if m.synthesizeFunc != nil {
		return m.synthesizeFunc(ctx, query, speakerID)
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
			return nil, &mockNetError{msg: "connection refused"}
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

// 正常系: CreateQueryが1回で成功する場合リトライなしで結果を返す
func TestRetryableVoicevoxClient_CreateQuery_Success(t *testing.T) {
	mock := newMockVoicevoxClient()
	expectedQuery := &entity.AudioQuery{SpeedScale: 1.0}
	mock.createQueryFunc = func(_ context.Context, _ string, _ int) (*entity.AudioQuery, error) {
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
	if mock.callCounts["CreateQuery"] != 1 {
		t.Errorf("expected CreateQuery to be called 1 time, got %d", mock.callCounts["CreateQuery"])
	}
}

// 正常系: Synthesizeが1回で成功する場合リトライなしで結果を返す
func TestRetryableVoicevoxClient_Synthesize_Success(t *testing.T) {
	mock := newMockVoicevoxClient()
	expectedWAV := []byte("wav data")
	mock.synthesizeFunc = func(_ context.Context, _ *entity.AudioQuery, _ int) ([]byte, error) {
		return expectedWAV, nil
	}

	noopSleep := func(_ context.Context, _ time.Duration) error { return nil }
	client := infra.NewRetryableVoicevoxClient(mock, infra.WithSleepFunc(noopSleep))

	query := &entity.AudioQuery{SpeedScale: 1.0}
	wav, err := client.Synthesize(context.Background(), query, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if string(wav) != string(expectedWAV) {
		t.Errorf("expected wav %v, got %v", expectedWAV, wav)
	}
	if mock.callCounts["Synthesize"] != 1 {
		t.Errorf("expected Synthesize to be called 1 time, got %d", mock.callCounts["Synthesize"])
	}
}

// リトライ成功: Synthesizeが1回失敗後に成功する場合リトライして結果を返す
func TestRetryableVoicevoxClient_Synthesize_RetrySuccess(t *testing.T) {
	mock := newMockVoicevoxClient()
	callCount := 0
	expectedWAV := []byte("wav data")
	mock.synthesizeFunc = func(_ context.Context, _ *entity.AudioQuery, _ int) ([]byte, error) {
		callCount++
		if callCount == 1 {
			return nil, &mockNetError{msg: "connection refused"}
		}
		return expectedWAV, nil
	}

	noopSleep := func(_ context.Context, _ time.Duration) error { return nil }
	client := infra.NewRetryableVoicevoxClient(mock, infra.WithSleepFunc(noopSleep))

	query := &entity.AudioQuery{SpeedScale: 1.0}
	wav, err := client.Synthesize(context.Background(), query, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if string(wav) != string(expectedWAV) {
		t.Errorf("expected wav %v, got %v", expectedWAV, wav)
	}
	if mock.callCounts["Synthesize"] != 2 {
		t.Errorf("expected Synthesize to be called 2 times, got %d", mock.callCounts["Synthesize"])
	}
}

// リトライ上限超過: CreateQueryが最大3回リトライしても失敗する場合エラーを返す
func TestRetryableVoicevoxClient_CreateQuery_MaxRetriesExceeded(t *testing.T) {
	mock := newMockVoicevoxClient()
	mock.createQueryFunc = func(_ context.Context, _ string, _ int) (*entity.AudioQuery, error) {
		return nil, &mockNetError{msg: "connection refused"}
	}

	noopSleep := func(_ context.Context, _ time.Duration) error { return nil }
	client := infra.NewRetryableVoicevoxClient(mock, infra.WithSleepFunc(noopSleep))

	query, err := client.CreateQuery(context.Background(), "test", 1)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if query != nil {
		t.Errorf("expected nil query, got %v", query)
	}
	// 初回 + 3回リトライ = 4回呼び出し
	if mock.callCounts["CreateQuery"] != 4 {
		t.Errorf("expected CreateQuery to be called 4 times (1 initial + 3 retries), got %d", mock.callCounts["CreateQuery"])
	}
}

// リトライ上限超過: Synthesizeが最大3回リトライしても失敗する場合エラーを返す
func TestRetryableVoicevoxClient_Synthesize_MaxRetriesExceeded(t *testing.T) {
	mock := newMockVoicevoxClient()
	mock.synthesizeFunc = func(_ context.Context, _ *entity.AudioQuery, _ int) ([]byte, error) {
		return nil, &mockNetError{msg: "connection refused"}
	}

	noopSleep := func(_ context.Context, _ time.Duration) error { return nil }
	client := infra.NewRetryableVoicevoxClient(mock, infra.WithSleepFunc(noopSleep))

	query := &entity.AudioQuery{SpeedScale: 1.0}
	wav, err := client.Synthesize(context.Background(), query, 1)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if wav != nil {
		t.Errorf("expected nil wav, got %v", wav)
	}
	// 初回 + 3回リトライ = 4回呼び出し
	if mock.callCounts["Synthesize"] != 4 {
		t.Errorf("expected Synthesize to be called 4 times (1 initial + 3 retries), got %d", mock.callCounts["Synthesize"])
	}
}

// 指数バックオフ: リトライ間隔が1秒→2秒→4秒で増加する
func TestRetryableVoicevoxClient_ExponentialBackoff(t *testing.T) {
	mock := newMockVoicevoxClient()
	mock.createQueryFunc = func(_ context.Context, _ string, _ int) (*entity.AudioQuery, error) {
		return nil, &mockNetError{msg: "connection refused"}
	}

	var sleepDurations []time.Duration
	recordingSleep := func(_ context.Context, d time.Duration) error {
		sleepDurations = append(sleepDurations, d)
		return nil
	}
	client := infra.NewRetryableVoicevoxClient(mock, infra.WithSleepFunc(recordingSleep))

	_, _ = client.CreateQuery(context.Background(), "test", 1)

	// 3回のリトライ → 3回のスリープ
	expectedDurations := []time.Duration{1 * time.Second, 2 * time.Second, 4 * time.Second}
	if len(sleepDurations) != len(expectedDurations) {
		t.Fatalf("expected %d sleep calls, got %d", len(expectedDurations), len(sleepDurations))
	}
	for i, expected := range expectedDurations {
		if sleepDurations[i] != expected {
			t.Errorf("sleep[%d]: expected %v, got %v", i, expected, sleepDurations[i])
		}
	}
}

// バックオフ上限: 指数バックオフの上限が30秒である
func TestRetryableVoicevoxClient_BackoffMaxInterval(t *testing.T) {
	mock := newMockVoicevoxClient()
	mock.createQueryFunc = func(_ context.Context, _ string, _ int) (*entity.AudioQuery, error) {
		return nil, &mockNetError{msg: "connection refused"}
	}

	var sleepDurations []time.Duration
	recordingSleep := func(_ context.Context, d time.Duration) error {
		sleepDurations = append(sleepDurations, d)
		return nil
	}

	// 初期間隔16秒、上限30秒、最大3回リトライで上限到達を確認
	client := infra.NewRetryableVoicevoxClient(mock,
		infra.WithSleepFunc(recordingSleep),
		infra.WithRetryConfig(infra.RetryConfig{
			MaxRetries:      3,
			InitialInterval: 16 * time.Second,
			MaxInterval:     30 * time.Second,
		}),
	)

	_, _ = client.CreateQuery(context.Background(), "test", 1)

	// 16秒 → 30秒（32秒が上限30秒でクリップ） → 30秒
	expectedDurations := []time.Duration{16 * time.Second, 30 * time.Second, 30 * time.Second}
	if len(sleepDurations) != len(expectedDurations) {
		t.Fatalf("expected %d sleep calls, got %d", len(expectedDurations), len(sleepDurations))
	}
	for i, expected := range expectedDurations {
		if sleepDurations[i] != expected {
			t.Errorf("sleep[%d]: expected %v, got %v", i, expected, sleepDurations[i])
		}
	}
}

// 非リトライエラー: HTTPステータスエラー（非net.Error）はリトライせず即座にエラーを返す
func TestRetryableVoicevoxClient_CreateQuery_NonRetryableError(t *testing.T) {
	mock := newMockVoicevoxClient()
	mock.createQueryFunc = func(_ context.Context, _ string, _ int) (*entity.AudioQuery, error) {
		return nil, errors.New("create query failed: status 400")
	}

	noopSleep := func(_ context.Context, _ time.Duration) error { return nil }
	client := infra.NewRetryableVoicevoxClient(mock, infra.WithSleepFunc(noopSleep))

	_, err := client.CreateQuery(context.Background(), "test", 1)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// 非net.Errorなのでリトライせず1回のみ呼び出し
	if mock.callCounts["CreateQuery"] != 1 {
		t.Errorf("expected CreateQuery to be called 1 time (no retry for non-retryable error), got %d", mock.callCounts["CreateQuery"])
	}
}

// 非リトライエラー: Synthesizeで非net.Errorはリトライせず即座にエラーを返す
func TestRetryableVoicevoxClient_Synthesize_NonRetryableError(t *testing.T) {
	mock := newMockVoicevoxClient()
	mock.synthesizeFunc = func(_ context.Context, _ *entity.AudioQuery, _ int) ([]byte, error) {
		return nil, errors.New("synthesis failed: status 500")
	}

	noopSleep := func(_ context.Context, _ time.Duration) error { return nil }
	client := infra.NewRetryableVoicevoxClient(mock, infra.WithSleepFunc(noopSleep))

	query := &entity.AudioQuery{SpeedScale: 1.0}
	_, err := client.Synthesize(context.Background(), query, 1)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if mock.callCounts["Synthesize"] != 1 {
		t.Errorf("expected Synthesize to be called 1 time (no retry for non-retryable error), got %d", mock.callCounts["Synthesize"])
	}
}

// 設定正規化: 不正なRetryConfigが正規化される
func TestRetryableVoicevoxClient_ConfigNormalization(t *testing.T) {
	inner := newMockVoicevoxClient()

	client := infra.NewRetryableVoicevoxClient(inner, infra.WithRetryConfig(infra.RetryConfig{
		MaxRetries:      -1,
		InitialInterval: -1 * time.Second,
		MaxInterval:     0,
	}))

	config := client.RetryConfig()
	if config.MaxRetries != 0 {
		t.Errorf("expected MaxRetries to be normalized to 0, got %d", config.MaxRetries)
	}
	if config.InitialInterval != 1*time.Second {
		t.Errorf("expected InitialInterval to be normalized to 1s, got %v", config.InitialInterval)
	}
	if config.MaxInterval != 30*time.Second {
		t.Errorf("expected MaxInterval to be normalized to 30s, got %v", config.MaxInterval)
	}
}

// コンテキストキャンセル: リトライ中にコンテキストがキャンセルされた場合エラーを返す
func TestRetryableVoicevoxClient_ContextCancelled(t *testing.T) {
	mock := newMockVoicevoxClient()
	mock.createQueryFunc = func(_ context.Context, _ string, _ int) (*entity.AudioQuery, error) {
		return nil, &mockNetError{msg: "connection refused"}
	}

	cancelledSleep := func(_ context.Context, _ time.Duration) error {
		return context.Canceled
	}
	client := infra.NewRetryableVoicevoxClient(mock, infra.WithSleepFunc(cancelledSleep))

	_, err := client.CreateQuery(context.Background(), "test", 1)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled error, got: %v", err)
	}
	// 初回の1回だけ呼び出されるべき（スリープでキャンセルされるため）
	if mock.callCounts["CreateQuery"] != 1 {
		t.Errorf("expected CreateQuery to be called 1 time, got %d", mock.callCounts["CreateQuery"])
	}
}
