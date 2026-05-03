package infra

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const viewerDetectTimeout = 500 * time.Millisecond

// DetectViewer checks if a viewer is running by reading the lock file and probing /api/status.
// Returns (addr, true) if the viewer is reachable, ("", false) otherwise.
func DetectViewer(lockPath string) (string, bool) {
	info, err := ReadViewerLockInfo(lockPath)
	if err != nil || info.Addr == "" {
		return "", false
	}

	client := &http.Client{Timeout: viewerDetectTimeout}
	resp, err := client.Get("http://" + info.Addr + "/api/status")
	if err != nil {
		return "", false
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", false
	}
	return info.Addr, true
}

// ViewerClip is a single clip in a play request.
type ViewerClip struct {
	Text       string   `json:"text"`
	SpeakerID  int      `json:"speaker_id"`
	Speed      *float64 `json:"speed,omitempty"`
	Pitch      *float64 `json:"pitch,omitempty"`
	Intonation *float64 `json:"intonation,omitempty"`
}

// ViewerPlayRequest is the request payload for POST /api/play.
type ViewerPlayRequest struct {
	Clips []ViewerClip `json:"clips"`
}

// ViewerPlayResponse is the response from POST /api/play.
type ViewerPlayResponse struct {
	PlaybackID   string `json:"playback_id"`
	ClipCount    int    `json:"clip_count"`
	Silent       bool   `json:"silent"`
	SilentReason string `json:"silent_reason,omitempty"`
}

// ViewerAPIClient is a thin HTTP client for calling viewer's /api/play.
type ViewerAPIClient struct {
	baseURL string // e.g., "http://host:port"
	client  *http.Client
}

// NewViewerAPIClient creates a client targeting the viewer at addr (host:port).
func NewViewerAPIClient(addr string) *ViewerAPIClient {
	return &ViewerAPIClient{
		baseURL: "http://" + addr,
		client:  &http.Client{},
	}
}

// NewViewerAPIClientFromURL creates a client targeting the viewer at the given base URL.
// Trailing slashes are trimmed.
func NewViewerAPIClientFromURL(rawURL string) *ViewerAPIClient {
	return &ViewerAPIClient{
		baseURL: strings.TrimRight(rawURL, "/"),
		client:  &http.Client{},
	}
}

// ViewerPlaybackResponse is the response from GET /api/playback/{id}.
type ViewerPlaybackResponse struct {
	ID             string `json:"id"`
	Status         string `json:"status"`
	ClipCount      int    `json:"clip_count"`
	CompletedClips int    `json:"completed_clips"`
	StartedAt      *int64 `json:"started_at"`
	FinishedAt     *int64 `json:"finished_at"`
	FailedReason   string `json:"failed_reason"`
}

// GetPlayback sends a GET /api/playback/{id} request and returns the playback state.
func (c *ViewerAPIClient) GetPlayback(ctx context.Context, id string) (*ViewerPlaybackResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/playback/"+id, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET /api/playback/%s: %w", id, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET /api/playback/%s returned %d", id, resp.StatusCode)
	}

	var result ViewerPlaybackResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode playback response: %w", err)
	}
	return &result, nil
}

// Play sends a POST /api/play request to the viewer and returns the response.
func (c *ViewerAPIClient) Play(ctx context.Context, req ViewerPlayRequest) (*ViewerPlayResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal play request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/play", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("POST /api/play: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("POST /api/play returned %d", resp.StatusCode)
	}

	var result ViewerPlayResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode play response: %w", err)
	}
	return &result, nil
}
