package infra

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/canpok1/vox-actor/internal/app"
	"github.com/canpok1/vox-actor/internal/domain/entity"
)

// defaultTimeout はHTTPクライアントのデフォルトタイムアウト時間。
const defaultTimeout = 30 * time.Second

// VoicevoxClientOption はVoicevoxClientの生成時に指定するオプション。
type VoicevoxClientOption func(*VoicevoxClient)

// WithTimeout はHTTPクライアントのタイムアウト時間を指定するオプション。
func WithTimeout(timeout time.Duration) VoicevoxClientOption {
	return func(c *VoicevoxClient) {
		if timeout <= 0 {
			return
		}
		c.httpClient.Timeout = timeout
	}
}

// VoicevoxClientがapp.VoicevoxClientインターフェースを満たすことをコンパイル時に検証する。
var _ app.VoicevoxClient = (*VoicevoxClient)(nil)

// VoicevoxClient はVOICEVOXエンジンとHTTP通信を行うクライアント。
type VoicevoxClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewVoicevoxClient は新しいVoicevoxClientを生成する。
// オプションを指定しない場合、HTTPクライアントのタイムアウトはデフォルト値（30秒）が使用される。
func NewVoicevoxClient(baseURL string, opts ...VoicevoxClientOption) *VoicevoxClient {
	c := &VoicevoxClient{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: defaultTimeout},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
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
func (c *VoicevoxClient) CreateQuery(ctx context.Context, text string, speakerID int) (*entity.AudioQuery, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/audio_query", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	q := req.URL.Query()
	q.Set("text", text)
	q.Set("speaker", strconv.Itoa(speakerID))
	req.URL.RawQuery = q.Encode()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("create query failed: status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var query entity.AudioQuery
	if err := json.Unmarshal(body, &query); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &query, nil
}

// Synthesize は音声合成を実行し、WAV形式のバイト列を返す。
// speed, pitch, intonation が非nilの場合、AudioQueryの対応フィールドを上書きする。
func (c *VoicevoxClient) Synthesize(ctx context.Context, query *entity.AudioQuery, speakerID int, speed, pitch, intonation *float64) ([]byte, error) {
	// AudioQueryのコピーを作成し、指定されたパラメータを上書き
	q := *query
	if speed != nil {
		q.SpeedScale = *speed
	}
	if pitch != nil {
		q.PitchScale = *pitch
	}
	if intonation != nil {
		q.IntonationScale = *intonation
	}

	jsonBody, err := json.Marshal(q)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal audio query: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/synthesis", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	params := req.URL.Query()
	params.Set("speaker", strconv.Itoa(speakerID))
	req.URL.RawQuery = params.Encode()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("synthesis failed: status %d", resp.StatusCode)
	}

	wav, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return wav, nil
}
