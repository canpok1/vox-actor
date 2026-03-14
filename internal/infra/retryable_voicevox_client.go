package infra

import (
	"context"
	"fmt"
	"time"

	"github.com/canpok1/vox-actor/internal/app"
	"github.com/canpok1/vox-actor/internal/domain/entity"
)

// RetryConfig はリトライの設定を保持する。
type RetryConfig struct {
	// MaxRetries は最大リトライ回数。
	MaxRetries int
	// InitialInterval はリトライの初期間隔。
	InitialInterval time.Duration
	// MaxInterval はリトライ間隔の上限。
	MaxInterval time.Duration
}

// RetryableVoicevoxClientOption はRetryableVoicevoxClientの生成時に指定するオプション。
type RetryableVoicevoxClientOption func(*RetryableVoicevoxClient)

// WithSleepFunc はリトライ時のスリープ関数を差し替えるオプション（テスト用）。
func WithSleepFunc(f func(ctx context.Context, d time.Duration) error) RetryableVoicevoxClientOption {
	return func(c *RetryableVoicevoxClient) {
		c.sleepFunc = f
	}
}

// RetryableVoicevoxClientがapp.VoicevoxClientインターフェースを満たすことをコンパイル時に検証する。
var _ app.VoicevoxClient = (*RetryableVoicevoxClient)(nil)

// RetryableVoicevoxClient はリトライ機能付きのVOICEVOXクライアント。
// HealthCheckはリトライ対象外とし、CreateQueryとSynthesizeに対して指数バックオフによるリトライを行う。
type RetryableVoicevoxClient struct {
	inner     app.VoicevoxClient
	config    RetryConfig
	sleepFunc func(ctx context.Context, d time.Duration) error
}

// defaultSleep はデフォルトのスリープ関数。コンテキストのキャンセルに対応する。
func defaultSleep(ctx context.Context, d time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(d):
		return nil
	}
}

// NewRetryableVoicevoxClient は新しいRetryableVoicevoxClientを生成する。
// デフォルト設定: 最大3回リトライ、初期間隔1秒、上限30秒。
func NewRetryableVoicevoxClient(inner app.VoicevoxClient, opts ...RetryableVoicevoxClientOption) *RetryableVoicevoxClient {
	c := &RetryableVoicevoxClient{
		inner: inner,
		config: RetryConfig{
			MaxRetries:      3,
			InitialInterval: 1 * time.Second,
			MaxInterval:     30 * time.Second,
		},
		sleepFunc: defaultSleep,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(c)
		}
	}
	return c
}

// RetryConfig はリトライ設定を返す（テスト用）。
func (c *RetryableVoicevoxClient) RetryConfig() RetryConfig {
	return c.config
}

// retryWithBackoff は指数バックオフ付きリトライを実行する汎用関数。
func (c *RetryableVoicevoxClient) retryWithBackoff(ctx context.Context, operation func() error) error {
	var lastErr error
	interval := c.config.InitialInterval

	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		lastErr = operation()
		if lastErr == nil {
			return nil
		}

		if attempt < c.config.MaxRetries {
			if err := c.sleepFunc(ctx, interval); err != nil {
				return fmt.Errorf("retry interrupted: %w", err)
			}
			interval *= 2
			if interval > c.config.MaxInterval {
				interval = c.config.MaxInterval
			}
		}
	}

	return lastErr
}

// HealthCheck はエンジンの疎通確認を行う。リトライは行わない。
func (c *RetryableVoicevoxClient) HealthCheck(ctx context.Context) error {
	return c.inner.HealthCheck(ctx)
}

// CreateQuery はテキストから音声合成用クエリを生成する。失敗時は指数バックオフでリトライを行う。
func (c *RetryableVoicevoxClient) CreateQuery(ctx context.Context, text string, speakerID int) (*entity.AudioQuery, error) {
	var result *entity.AudioQuery
	err := c.retryWithBackoff(ctx, func() error {
		var err error
		result, err = c.inner.CreateQuery(ctx, text, speakerID)
		return err
	})
	return result, err
}

// Synthesize は音声合成を実行し、WAV形式のバイト列を返す。失敗時は指数バックオフでリトライを行う。
func (c *RetryableVoicevoxClient) Synthesize(ctx context.Context, query *entity.AudioQuery, speakerID int, speed, pitch, intonation *float64) ([]byte, error) {
	var result []byte
	err := c.retryWithBackoff(ctx, func() error {
		var err error
		result, err = c.inner.Synthesize(ctx, query, speakerID, speed, pitch, intonation)
		return err
	})
	return result, err
}
