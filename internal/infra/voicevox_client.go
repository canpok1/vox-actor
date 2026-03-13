package infra

import (
	"context"
	"fmt"
	"net/http"

	"github.com/canpok1/vox-actor/internal/domain/entity"
)

// VoicevoxClient はVOICEVOXエンジンとHTTP通信を行うクライアント。
type VoicevoxClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewVoicevoxClient は新しいVoicevoxClientを生成する。
func NewVoicevoxClient(baseURL string) *VoicevoxClient {
	return &VoicevoxClient{
		baseURL:    baseURL,
		httpClient: &http.Client{},
	}
}

// HealthCheck はエンジンの疎通確認を行う。
func (c *VoicevoxClient) HealthCheck(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed: status %d", resp.StatusCode)
	}

	return nil
}

// CreateQuery はテキストから音声合成用クエリを生成する。
func (c *VoicevoxClient) CreateQuery(_ context.Context, _ string, _ int) (*entity.AudioQuery, error) {
	return nil, nil
}

// Synthesize は音声合成を実行し、WAV形式のバイト列を返す。
func (c *VoicevoxClient) Synthesize(_ context.Context, _ *entity.AudioQuery, _ int, _, _, _ *float64) ([]byte, error) {
	return nil, nil
}
