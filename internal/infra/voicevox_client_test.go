package infra_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/canpok1/vox-actor/internal/domain/entity"
	"github.com/canpok1/vox-actor/internal/infra"
)

func TestHealthCheck_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("unexpected method: %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := infra.NewVoicevoxClient(server.URL)
	err := client.HealthCheck(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestHealthCheck_Non200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := infra.NewVoicevoxClient(server.URL)
	err := client.HealthCheck(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestHealthCheck_ConnectionError(t *testing.T) {
	client := infra.NewVoicevoxClient("http://localhost:1")
	err := client.HealthCheck(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCreateQuery_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/audio_query" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}
		if got := r.URL.Query().Get("text"); got != "こんにちは" {
			t.Errorf("unexpected text: %s", got)
		}
		if got := r.URL.Query().Get("speaker"); got != "1" {
			t.Errorf("unexpected speaker: %s", got)
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"accent_phrases": [
				{
					"moras": [
						{"text": "コ", "vowel": "o", "vowel_length": 0.15, "pitch": 3.5}
					],
					"accent": 1,
					"pause_mora": null,
					"is_interrogative": false
				}
			],
			"speedScale": 1.0,
			"pitchScale": 0.0,
			"intonationScale": 1.0,
			"volumeScale": 1.0,
			"prePhonemeLength": 0.1,
			"postPhonemeLength": 0.1,
			"pauseLength": null,
			"pauseLengthScale": 1.0,
			"outputSamplingRate": 24000,
			"outputStereo": false,
			"kana": "コンニチワ"
		}`)
	}))
	defer server.Close()

	client := infra.NewVoicevoxClient(server.URL)
	query, err := client.CreateQuery(context.Background(), "こんにちは", 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if query == nil {
		t.Fatal("expected query, got nil")
	}

	// フィールド検証
	if query.SpeedScale != 1.0 {
		t.Errorf("expected speedScale 1.0, got %f", query.SpeedScale)
	}
	if query.PitchScale != 0.0 {
		t.Errorf("expected pitchScale 0.0, got %f", query.PitchScale)
	}
	if query.IntonationScale != 1.0 {
		t.Errorf("expected intonationScale 1.0, got %f", query.IntonationScale)
	}
	if query.OutputSamplingRate != 24000 {
		t.Errorf("expected outputSamplingRate 24000, got %d", query.OutputSamplingRate)
	}
	if query.Kana != "コンニチワ" {
		t.Errorf("expected kana コンニチワ, got %s", query.Kana)
	}
	if len(query.AccentPhrases) != 1 {
		t.Fatalf("expected 1 accent phrase, got %d", len(query.AccentPhrases))
	}
	if len(query.AccentPhrases[0].Moras) != 1 {
		t.Fatalf("expected 1 mora, got %d", len(query.AccentPhrases[0].Moras))
	}
	if query.AccentPhrases[0].Moras[0].Text != "コ" {
		t.Errorf("expected mora text コ, got %s", query.AccentPhrases[0].Moras[0].Text)
	}
}

func TestCreateQuery_Non200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	client := infra.NewVoicevoxClient(server.URL)
	query, err := client.CreateQuery(context.Background(), "test", 1)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if query != nil {
		t.Fatal("expected nil query on error")
	}
}

func TestSynthesize_Success(t *testing.T) {
	expectedWAV := []byte("RIFF....WAVEfmt mock wav data")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/synthesis" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}
		if got := r.URL.Query().Get("speaker"); got != "1" {
			t.Errorf("unexpected speaker: %s", got)
		}

		// リクエストボディがAudioQueryのJSONであることを確認
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}
		var q entity.AudioQuery
		if err := json.Unmarshal(body, &q); err != nil {
			t.Fatalf("failed to unmarshal request body: %v", err)
		}
		if q.SpeedScale != 1.0 {
			t.Errorf("expected speedScale 1.0, got %f", q.SpeedScale)
		}

		w.Header().Set("Content-Type", "audio/wav")
		w.Write(expectedWAV)
	}))
	defer server.Close()

	query := &entity.AudioQuery{
		SpeedScale:         1.0,
		PitchScale:         0.0,
		IntonationScale:    1.0,
		VolumeScale:        1.0,
		OutputSamplingRate: 24000,
	}

	client := infra.NewVoicevoxClient(server.URL)
	wav, err := client.Synthesize(context.Background(), query, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !bytes.Equal(wav, expectedWAV) {
		t.Errorf("unexpected wav data: got %v", wav)
	}
}

func TestSynthesize_PassesThroughQueryDirectly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}
		var q entity.AudioQuery
		if err := json.Unmarshal(body, &q); err != nil {
			t.Fatalf("failed to unmarshal request body: %v", err)
		}

		// queryがそのまま渡されることを検証（パラメータ上書きはドメイン層の責務）
		if q.SpeedScale != 1.5 {
			t.Errorf("expected speedScale 1.5, got %f", q.SpeedScale)
		}
		if q.PitchScale != 0.5 {
			t.Errorf("expected pitchScale 0.5, got %f", q.PitchScale)
		}
		if q.IntonationScale != 2.0 {
			t.Errorf("expected intonationScale 2.0, got %f", q.IntonationScale)
		}

		w.Header().Set("Content-Type", "audio/wav")
		w.Write([]byte("wav"))
	}))
	defer server.Close()

	// 既にWithOverridesで上書き済みのqueryを渡す想定
	query := &entity.AudioQuery{
		SpeedScale:         1.5,
		PitchScale:         0.5,
		IntonationScale:    2.0,
		VolumeScale:        1.0,
		OutputSamplingRate: 24000,
	}

	client := infra.NewVoicevoxClient(server.URL)
	_, err := client.Synthesize(context.Background(), query, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestNewVoicevoxClient_DefaultTimeout(t *testing.T) {
	// デフォルトのタイムアウトが30秒に設定されていることを直接検証する
	client := infra.NewVoicevoxClient("http://localhost")
	if got := client.ClientTimeout(); got != 30*time.Second {
		t.Errorf("expected default timeout 30s, got %v", got)
	}
}

func TestNewVoicevoxClient_WithTimeout(t *testing.T) {
	// WithTimeout オプションでカスタムタイムアウトを指定できることを確認
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// 50msのタイムアウトを設定 → 200ms待つサーバーに対してタイムアウトエラーになるはず
	client := infra.NewVoicevoxClient(server.URL, infra.WithTimeout(50*time.Millisecond))
	err := client.HealthCheck(context.Background())
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	// タイムアウトエラーであることを型で確認
	var netErr net.Error
	if (!errors.As(err, &netErr) || !netErr.Timeout()) &&
		!errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected timeout-related error, got: %v", err)
	}
}

func TestNewVoicevoxClient_WithTimeoutZeroIgnored(t *testing.T) {
	// 0以下のタイムアウトは無視され、デフォルト値が維持されることを確認
	client := infra.NewVoicevoxClient("http://localhost", infra.WithTimeout(0))
	if got := client.ClientTimeout(); got != 30*time.Second {
		t.Errorf("expected default timeout 30s when 0 is passed, got %v", got)
	}

	clientNeg := infra.NewVoicevoxClient("http://localhost", infra.WithTimeout(-1*time.Second))
	if got := clientNeg.ClientTimeout(); got != 30*time.Second {
		t.Errorf("expected default timeout 30s when negative is passed, got %v", got)
	}
}

func TestNewVoicevoxClient_WithTimeoutSuccess(t *testing.T) {
	// 十分なタイムアウトを設定した場合は正常に完了することを確認
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := infra.NewVoicevoxClient(server.URL, infra.WithTimeout(5*time.Second))
	err := client.HealthCheck(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestSynthesize_Non200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	query := &entity.AudioQuery{
		SpeedScale:         1.0,
		OutputSamplingRate: 24000,
	}

	client := infra.NewVoicevoxClient(server.URL)
	wav, err := client.Synthesize(context.Background(), query, 1)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if wav != nil {
		t.Fatal("expected nil wav on error")
	}
}

func TestGetSpeakers_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/speakers" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("unexpected method: %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[
			{
				"name": "ずんだもん",
				"speaker_uuid": "388f246b-8c41-4ac1-8e2d-5d79f3ff56d9",
				"styles": [
					{"name": "ノーマル", "id": 3, "type": "talk"},
					{"name": "あまあま", "id": 1, "type": "talk"}
				],
				"version": "0.16.0"
			},
			{
				"name": "四国めたん",
				"speaker_uuid": "7ffcb7ce-00ec-4bdc-82cd-45a8889e43ff",
				"styles": [
					{"name": "ノーマル", "id": 2, "type": "talk"}
				],
				"version": "0.16.0"
			}
		]`)
	}))
	defer server.Close()

	client := infra.NewVoicevoxClient(server.URL)
	speakers, err := client.GetSpeakers(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(speakers) != 2 {
		t.Fatalf("expected 2 speakers, got %d", len(speakers))
	}
	if speakers[0].Name != "ずんだもん" {
		t.Errorf("expected first speaker name 'ずんだもん', got %s", speakers[0].Name)
	}
	if len(speakers[0].Styles) != 2 {
		t.Fatalf("expected ずんだもん to have 2 styles, got %d", len(speakers[0].Styles))
	}
	if speakers[0].Styles[0].ID != 3 || speakers[0].Styles[0].Name != "ノーマル" {
		t.Errorf("unexpected first style: %+v", speakers[0].Styles[0])
	}
	if speakers[1].Name != "四国めたん" || speakers[1].Styles[0].ID != 2 {
		t.Errorf("unexpected second speaker: %+v", speakers[1])
	}
}

func TestGetSpeakers_Non200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := infra.NewVoicevoxClient(server.URL)
	speakers, err := client.GetSpeakers(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if speakers != nil {
		t.Fatal("expected nil speakers on error")
	}
}

func TestGetSpeakers_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("not-json"))
	}))
	defer server.Close()

	client := infra.NewVoicevoxClient(server.URL)
	speakers, err := client.GetSpeakers(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if speakers != nil {
		t.Fatal("expected nil speakers on error")
	}
}
