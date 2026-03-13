package infra_test

// TODO: Synthesize - 正常系: speed/pitch/intonationを上書きして送信する
// TODO: Synthesize - 異常系: エンジンが非200を返す場合エラーを返す

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

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
	wav, err := client.Synthesize(context.Background(), query, 1, nil, nil, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !bytes.Equal(wav, expectedWAV) {
		t.Errorf("unexpected wav data: got %v", wav)
	}
}
