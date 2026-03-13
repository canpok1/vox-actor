package app_test

import (
	"context"
	"testing"

	"github.com/canpok1/vox-actor/internal/app"
	"github.com/canpok1/vox-actor/internal/domain/entity"
)

// act_usecase テストリスト
//
// TODO: 正常系: 複数台本で全ての台本が順番に処理される
// TODO: 正常系: 空ファイルの台本はスキップされる（合成・再生が呼ばれない）
// TODO: 正常系: 空ファイルと通常ファイルが混在する場合、空ファイルのみスキップされる
// TODO: 正常系: speed/pitch/intonationオプションがSynthesizeに渡される
// TODO: 異常系: HealthCheckが失敗した場合エラーを返す
// TODO: 異常系: ScriptReader.Readが失敗した場合エラーを返す
// TODO: 異常系: CreateQueryが失敗した場合エラーを返す
// TODO: 異常系: Synthesizeが失敗した場合エラーを返す
// TODO: 異常系: Playが失敗した場合エラーを返す

// --- モック ---

type mockScriptReader struct {
	scripts []entity.Script
	err     error
}

func (m *mockScriptReader) Read(_ string) ([]entity.Script, error) {
	return m.scripts, m.err
}

type mockVoicevoxClient struct {
	healthCheckErr error
	query          *entity.AudioQuery
	createQueryErr error
	wavData        []byte
	synthesizeErr  error
}

func (m *mockVoicevoxClient) HealthCheck(_ context.Context) error {
	return m.healthCheckErr
}

func (m *mockVoicevoxClient) CreateQuery(_ context.Context, _ string, _ int) (*entity.AudioQuery, error) {
	return m.query, m.createQueryErr
}

func (m *mockVoicevoxClient) Synthesize(_ context.Context, _ *entity.AudioQuery, _ int, _, _, _ *float64) ([]byte, error) {
	return m.wavData, m.synthesizeErr
}

type mockAudioPlayer struct {
	err error
}

func (m *mockAudioPlayer) Play(_ context.Context, _ []byte) error {
	return m.err
}

// --- テスト ---

func TestActUsecase_Run_SingleScript_Success(t *testing.T) {
	reader := &mockScriptReader{
		scripts: []entity.Script{
			{Path: "test.txt", Text: "こんにちは", IsEmpty: false},
		},
	}
	client := &mockVoicevoxClient{
		query:   &entity.AudioQuery{},
		wavData: []byte("fake-wav"),
	}
	player := &mockAudioPlayer{}

	uc := app.NewActUsecase(reader, client, player)
	params := app.ActParams{
		Path:      "test.txt",
		SpeakerID: 3,
	}

	err := uc.Run(context.Background(), params)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}
