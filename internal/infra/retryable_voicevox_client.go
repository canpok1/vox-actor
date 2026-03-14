package infra

import (
	"context"
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

// RetryableVoicevoxClientがapp.VoicevoxClientインターフェースを満たすことをコンパイル時に検証する。
var _ app.VoicevoxClient = (*RetryableVoicevoxClient)(nil)

// RetryableVoicevoxClient はリトライ機能付きのVOICEVOXクライアント。
// HealthCheckはリトライ対象外とし、CreateQueryとSynthesizeに対して指数バックオフによるリトライを行う。
type RetryableVoicevoxClient struct {
	inner  app.VoicevoxClient
	config RetryConfig
}

// NewRetryableVoicevoxClient は新しいRetryableVoicevoxClientを生成する。
// デフォルト設定: 最大3回リトライ、初期間隔1秒、上限30秒。
func NewRetryableVoicevoxClient(inner app.VoicevoxClient) *RetryableVoicevoxClient {
	return &RetryableVoicevoxClient{
		inner: inner,
		config: RetryConfig{
			MaxRetries:      3,
			InitialInterval: 1 * time.Second,
			MaxInterval:     30 * time.Second,
		},
	}
}

// RetryConfig はリトライ設定を返す（テスト用）。
func (c *RetryableVoicevoxClient) RetryConfig() RetryConfig {
	return c.config
}

// HealthCheck はエンジンの疎通確認を行う。リトライは行わない。
func (c *RetryableVoicevoxClient) HealthCheck(ctx context.Context) error {
	return c.inner.HealthCheck(ctx)
}

// CreateQuery はテキストから音声合成用クエリを生成する。失敗時はリトライを行う。
func (c *RetryableVoicevoxClient) CreateQuery(ctx context.Context, text string, speakerID int) (*entity.AudioQuery, error) {
	return c.inner.CreateQuery(ctx, text, speakerID)
}

// Synthesize は音声合成を実行し、WAV形式のバイト列を返す。失敗時はリトライを行う。
func (c *RetryableVoicevoxClient) Synthesize(ctx context.Context, query *entity.AudioQuery, speakerID int, speed, pitch, intonation *float64) ([]byte, error) {
	return c.inner.Synthesize(ctx, query, speakerID, speed, pitch, intonation)
}
