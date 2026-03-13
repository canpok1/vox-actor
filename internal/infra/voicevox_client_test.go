package infra_test

// TODO: HealthCheck - 異常系: エンジンが非200を返す場合エラーを返す
// TODO: HealthCheck - 異常系: エンジンに接続できない場合エラーを返す
// TODO: CreateQuery - 正常系: テキストとspeakerIDからAudioQueryを返す
// TODO: CreateQuery - 正常系: レスポンスJSONの各フィールドが正しくパースされる
// TODO: CreateQuery - 異常系: エンジンが非200を返す場合エラーを返す
// TODO: Synthesize - 正常系: AudioQueryとspeakerIDからWAVバイト列を返す
// TODO: Synthesize - 正常系: speed/pitch/intonationを上書きして送信する
// TODO: Synthesize - 異常系: エンジンが非200を返す場合エラーを返す

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

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
