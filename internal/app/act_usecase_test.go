package app_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/canpok1/vox-actor/internal/app"
	"github.com/canpok1/vox-actor/internal/domain/entity"
	"github.com/canpok1/vox-actor/internal/infra/logging"
)

// act_usecase テストリスト
// DONE: 進捗表示: 複数スクリプト処理時にInfoログに「[1/3]」のような進捗が含まれる
// DONE: synthesis completedがInfoレベルで出力される

// グレースフルシャットダウン テストリスト
// DONE: contextがキャンセルされた場合、次のスクリプトの合成・再生をスキップしてnilを返す
// DONE: 複数スクリプトの途中でcontextがキャンセルされた場合、再生中の音声完了後に残りをスキップする
// TODO: contextがキャンセルされていない場合、全スクリプトを通常通り処理する（既存テストで担保済み）

// --- モック ---

type mockScriptReader struct {
	scripts []entity.Script
	err     error
}

func (m *mockScriptReader) Read(_ string) ([]entity.Script, error) {
	return m.scripts, m.err
}

type createQueryCallArgs struct {
	speakerID int
}

type mockVoicevoxClient struct {
	healthCheckErr      error
	query               *entity.AudioQuery
	createQueryErr      error
	wavData             []byte
	synthesizeErr       error
	createQueryCalls    int
	createQueryCallArgs []createQueryCallArgs
	synthesizeCalls     int
	synthesizeArgs      []synthesizeCallArgs
	speakers            []entity.Speaker
	getSpeakersErr      error
	getSpeakersCalls    int
}

type synthesizeCallArgs struct {
	query     *entity.AudioQuery
	speakerID int
}

func (m *mockVoicevoxClient) HealthCheck(_ context.Context) error {
	return m.healthCheckErr
}

func (m *mockVoicevoxClient) CreateQuery(_ context.Context, _ string, speakerID int) (*entity.AudioQuery, error) {
	m.createQueryCalls++
	m.createQueryCallArgs = append(m.createQueryCallArgs, createQueryCallArgs{speakerID: speakerID})
	return m.query, m.createQueryErr
}

func (m *mockVoicevoxClient) Synthesize(_ context.Context, query *entity.AudioQuery, speakerID int) ([]byte, error) {
	m.synthesizeCalls++
	m.synthesizeArgs = append(m.synthesizeArgs, synthesizeCallArgs{query: query, speakerID: speakerID})
	return m.wavData, m.synthesizeErr
}

func (m *mockVoicevoxClient) GetSpeakers(_ context.Context) ([]entity.Speaker, error) {
	m.getSpeakersCalls++
	return m.speakers, m.getSpeakersErr
}

type mockAudioPlayer struct {
	err       error
	playCalls int
	playMetas []app.PlayMeta
}

func (m *mockAudioPlayer) Play(_ context.Context, _ []byte, meta app.PlayMeta) error {
	m.playCalls++
	m.playMetas = append(m.playMetas, meta)
	return m.err
}

// --- テスト ---

func TestActUsecase_Run_PlayReceivesScriptText(t *testing.T) {
	reader := &mockScriptReader{
		scripts: []entity.Script{
			{Path: "a.txt", Text: "おはようなのだ", IsEmpty: false},
			{Path: "b.txt", Text: "さようならなのだ", IsEmpty: false},
		},
	}
	client := &mockVoicevoxClient{
		query:   &entity.AudioQuery{},
		wavData: []byte("fake-wav"),
	}
	player := &mockAudioPlayer{}

	uc := app.NewActUsecase(reader, client, player)
	params := app.ActParams{Path: "scripts/", SpeakerID: 3}

	if err := uc.Run(context.Background(), params); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(player.playMetas) != 2 {
		t.Fatalf("expected 2 Play calls, got: %d", len(player.playMetas))
	}
	if player.playMetas[0].Text != "おはようなのだ" || player.playMetas[1].Text != "さようならなのだ" {
		t.Errorf("expected PlayMeta.Text in script order, got: %+v", player.playMetas)
	}
	if player.playMetas[0].SpeakerID != 3 || player.playMetas[1].SpeakerID != 3 {
		t.Errorf("expected PlayMeta.SpeakerID=3 for both calls (default), got: %+v", player.playMetas)
	}
}

func TestActUsecase_Run_PlayReceivesScriptResolvedSpeakerID(t *testing.T) {
	overrideID := 7
	reader := &mockScriptReader{
		scripts: []entity.Script{
			{Path: "a.txt", Text: "default", IsEmpty: false},
			{Path: "b.txt", Text: "override", IsEmpty: false, SpeakerID: &overrideID},
		},
	}
	client := &mockVoicevoxClient{
		query:   &entity.AudioQuery{},
		wavData: []byte("fake-wav"),
	}
	player := &mockAudioPlayer{}

	uc := app.NewActUsecase(reader, client, player)
	params := app.ActParams{Path: "scripts/", SpeakerID: 3}

	if err := uc.Run(context.Background(), params); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(player.playMetas) != 2 {
		t.Fatalf("expected 2 Play calls, got: %d", len(player.playMetas))
	}
	if player.playMetas[0].SpeakerID != 3 {
		t.Errorf("expected first PlayMeta.SpeakerID=3 (default), got: %d", player.playMetas[0].SpeakerID)
	}
	if player.playMetas[1].SpeakerID != 7 {
		t.Errorf("expected second PlayMeta.SpeakerID=7 (script override), got: %d", player.playMetas[1].SpeakerID)
	}
}

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

func TestActUsecase_Run_MultipleScripts_Success(t *testing.T) {
	reader := &mockScriptReader{
		scripts: []entity.Script{
			{Path: "a.txt", Text: "おはよう", IsEmpty: false},
			{Path: "b.txt", Text: "こんにちは", IsEmpty: false},
			{Path: "c.txt", Text: "こんばんは", IsEmpty: false},
		},
	}
	client := &mockVoicevoxClient{
		query:   &entity.AudioQuery{},
		wavData: []byte("fake-wav"),
	}
	player := &mockAudioPlayer{}

	uc := app.NewActUsecase(reader, client, player)
	params := app.ActParams{
		Path:      "scripts/",
		SpeakerID: 3,
	}

	err := uc.Run(context.Background(), params)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if client.createQueryCalls != 3 {
		t.Errorf("expected 3 CreateQuery calls, got: %d", client.createQueryCalls)
	}
	if client.synthesizeCalls != 3 {
		t.Errorf("expected 3 Synthesize calls, got: %d", client.synthesizeCalls)
	}
	if player.playCalls != 3 {
		t.Errorf("expected 3 Play calls, got: %d", player.playCalls)
	}
}

func TestActUsecase_Run_EmptyScript_Skipped(t *testing.T) {
	reader := &mockScriptReader{
		scripts: []entity.Script{
			{Path: "empty.txt", Text: "", IsEmpty: true},
		},
	}
	client := &mockVoicevoxClient{
		query:   &entity.AudioQuery{},
		wavData: []byte("fake-wav"),
	}
	player := &mockAudioPlayer{}

	uc := app.NewActUsecase(reader, client, player)
	params := app.ActParams{Path: "empty.txt", SpeakerID: 3}

	err := uc.Run(context.Background(), params)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if client.createQueryCalls != 0 {
		t.Errorf("expected 0 CreateQuery calls for empty script, got: %d", client.createQueryCalls)
	}
	if player.playCalls != 0 {
		t.Errorf("expected 0 Play calls for empty script, got: %d", player.playCalls)
	}
}

func TestActUsecase_Run_MixedEmptyAndNonEmpty(t *testing.T) {
	reader := &mockScriptReader{
		scripts: []entity.Script{
			{Path: "a.txt", Text: "おはよう", IsEmpty: false},
			{Path: "b.txt", Text: "", IsEmpty: true},
			{Path: "c.txt", Text: "こんにちは", IsEmpty: false},
		},
	}
	client := &mockVoicevoxClient{
		query:   &entity.AudioQuery{},
		wavData: []byte("fake-wav"),
	}
	player := &mockAudioPlayer{}

	uc := app.NewActUsecase(reader, client, player)
	params := app.ActParams{Path: "scripts/", SpeakerID: 3}

	err := uc.Run(context.Background(), params)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if client.createQueryCalls != 2 {
		t.Errorf("expected 2 CreateQuery calls, got: %d", client.createQueryCalls)
	}
	if player.playCalls != 2 {
		t.Errorf("expected 2 Play calls, got: %d", player.playCalls)
	}
}

func TestActUsecase_Run_WithOptions(t *testing.T) {
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

	speed := 1.5
	pitch := 0.5
	intonation := 1.2

	uc := app.NewActUsecase(reader, client, player)
	params := app.ActParams{
		Path:       "test.txt",
		SpeakerID:  3,
		Speed:      &speed,
		Pitch:      &pitch,
		Intonation: &intonation,
	}

	err := uc.Run(context.Background(), params)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(client.synthesizeArgs) != 1 {
		t.Fatalf("expected 1 Synthesize call, got: %d", len(client.synthesizeArgs))
	}
	args := client.synthesizeArgs[0]
	// WithOverridesにより、queryのフィールドが上書きされている
	if args.query.SpeedScale != 1.5 {
		t.Errorf("expected SpeedScale 1.5, got: %f", args.query.SpeedScale)
	}
	if args.query.PitchScale != 0.5 {
		t.Errorf("expected PitchScale 0.5, got: %f", args.query.PitchScale)
	}
	if args.query.IntonationScale != 1.2 {
		t.Errorf("expected IntonationScale 1.2, got: %f", args.query.IntonationScale)
	}
}

func TestActUsecase_Run_HealthCheckError(t *testing.T) {
	reader := &mockScriptReader{}
	client := &mockVoicevoxClient{
		healthCheckErr: errors.New("connection refused"),
	}
	player := &mockAudioPlayer{}

	uc := app.NewActUsecase(reader, client, player)
	params := app.ActParams{Path: "test.txt", SpeakerID: 3}

	err := uc.Run(context.Background(), params)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestActUsecase_Run_ReadError(t *testing.T) {
	reader := &mockScriptReader{
		err: errors.New("file not found"),
	}
	client := &mockVoicevoxClient{}
	player := &mockAudioPlayer{}

	uc := app.NewActUsecase(reader, client, player)
	params := app.ActParams{Path: "notexist.txt", SpeakerID: 3}

	err := uc.Run(context.Background(), params)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestActUsecase_Run_CreateQueryError(t *testing.T) {
	reader := &mockScriptReader{
		scripts: []entity.Script{
			{Path: "test.txt", Text: "こんにちは", IsEmpty: false},
		},
	}
	client := &mockVoicevoxClient{
		createQueryErr: errors.New("query failed"),
	}
	player := &mockAudioPlayer{}

	uc := app.NewActUsecase(reader, client, player)
	params := app.ActParams{Path: "test.txt", SpeakerID: 3}

	err := uc.Run(context.Background(), params)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "test.txt") {
		t.Errorf("expected error to include script path, got: %v", err)
	}
}

func TestActUsecase_Run_SynthesizeError(t *testing.T) {
	reader := &mockScriptReader{
		scripts: []entity.Script{
			{Path: "test.txt", Text: "こんにちは", IsEmpty: false},
		},
	}
	client := &mockVoicevoxClient{
		query:         &entity.AudioQuery{},
		synthesizeErr: errors.New("synthesis failed"),
	}
	player := &mockAudioPlayer{}

	uc := app.NewActUsecase(reader, client, player)
	params := app.ActParams{Path: "test.txt", SpeakerID: 3}

	err := uc.Run(context.Background(), params)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "test.txt") {
		t.Errorf("expected error to include script path, got: %v", err)
	}
}

func TestActUsecase_Run_PlayError(t *testing.T) {
	reader := &mockScriptReader{
		scripts: []entity.Script{
			{Path: "test.txt", Text: "こんにちは", IsEmpty: false},
		},
	}
	client := &mockVoicevoxClient{
		query:   &entity.AudioQuery{},
		wavData: []byte("fake-wav"),
	}
	player := &mockAudioPlayer{
		err: errors.New("play failed"),
	}

	uc := app.NewActUsecase(reader, client, player)
	params := app.ActParams{Path: "test.txt", SpeakerID: 3}

	err := uc.Run(context.Background(), params)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "test.txt") {
		t.Errorf("expected error to include script path, got: %v", err)
	}
}

func TestActUsecase_Run_CancelledContext_SkipsRemainingScripts(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	reader := &mockScriptReader{
		scripts: []entity.Script{
			{Path: "a.txt", Text: "おはよう", IsEmpty: false},
			{Path: "b.txt", Text: "こんにちは", IsEmpty: false},
			{Path: "c.txt", Text: "こんばんは", IsEmpty: false},
		},
	}
	client := &mockVoicevoxClient{
		query:   &entity.AudioQuery{},
		wavData: []byte("fake-wav"),
	}
	// Playerが最初のスクリプト再生完了後にcontextをキャンセルする
	player := &cancellingAudioPlayer{
		cancelAfter: 1,
		cancelFunc:  cancel,
	}

	uc := app.NewActUsecase(reader, client, player)
	params := app.ActParams{Path: "scripts/", SpeakerID: 3}

	err := uc.Run(ctx, params)

	// グレースフルシャットダウン: エラーなしで終了する
	if err != nil {
		t.Fatalf("expected nil error for graceful shutdown, got: %v", err)
	}

	// 最初のスクリプトのみ再生され、残りはスキップされる
	if player.playCalls != 1 {
		t.Errorf("expected 1 Play call, got: %d", player.playCalls)
	}
}

func TestActUsecase_Run_CancelDuringCreateQuery_ReturnsNil(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	reader := &mockScriptReader{
		scripts: []entity.Script{
			{Path: "a.txt", Text: "おはよう", IsEmpty: false},
			{Path: "b.txt", Text: "こんにちは", IsEmpty: false},
		},
	}
	// CreateQueryが呼ばれた時にcontextをキャンセルしてcontext.Canceledを返すモック
	client := &cancellingVoicevoxClient{
		cancelAfterCreateQuery: 1,
		cancelFunc:             cancel,
		query:                  &entity.AudioQuery{},
		wavData:                []byte("fake-wav"),
	}
	player := &mockAudioPlayer{}

	uc := app.NewActUsecase(reader, client, player)
	params := app.ActParams{Path: "scripts/", SpeakerID: 3}

	err := uc.Run(ctx, params)

	// グレースフルシャットダウン: context.Canceledはエラーではなくnilを返す
	if err != nil {
		t.Fatalf("expected nil error for graceful shutdown, got: %v", err)
	}

	// 最初のスクリプトは再生完了、2番目のCreateQueryでキャンセル
	if player.playCalls != 1 {
		t.Errorf("expected 1 Play call, got: %d", player.playCalls)
	}
}

// cancellingVoicevoxClient は指定回数のCreateQuery後にcontextをキャンセルするモック。
type cancellingVoicevoxClient struct {
	cancelAfterCreateQuery int
	cancelFunc             context.CancelFunc
	query                  *entity.AudioQuery
	wavData                []byte
	createQueryCalls       int
}

func (m *cancellingVoicevoxClient) HealthCheck(_ context.Context) error {
	return nil
}

func (m *cancellingVoicevoxClient) CreateQuery(_ context.Context, _ string, _ int) (*entity.AudioQuery, error) {
	m.createQueryCalls++
	if m.createQueryCalls > m.cancelAfterCreateQuery {
		m.cancelFunc()
		return nil, context.Canceled
	}
	return m.query, nil
}

func (m *cancellingVoicevoxClient) Synthesize(_ context.Context, _ *entity.AudioQuery, _ int) ([]byte, error) {
	return m.wavData, nil
}

func (m *cancellingVoicevoxClient) GetSpeakers(_ context.Context) ([]entity.Speaker, error) {
	return nil, nil
}

// cancellingAudioPlayer は指定回数再生後にcontextをキャンセルするモック。
type cancellingAudioPlayer struct {
	playCalls   int
	cancelAfter int
	cancelFunc  context.CancelFunc
}

func (m *cancellingAudioPlayer) Play(_ context.Context, _ []byte, _ app.PlayMeta) error {
	m.playCalls++
	if m.playCalls >= m.cancelAfter {
		m.cancelFunc()
	}
	return nil
}

func TestActUsecase_Run_LogsProcessingStatus(t *testing.T) {
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

	var buf bytes.Buffer
	logger := slog.New(logging.NewHumanHandler(&buf, &logging.HumanHandlerOptions{Level: slog.LevelInfo, NoColor: true}))

	uc := app.NewActUsecase(reader, client, player, app.WithLogger(logger))
	params := app.ActParams{
		Path:      "test.txt",
		SpeakerID: 3,
	}

	err := uc.Run(context.Background(), params)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	output := buf.String()
	// 処理状況のInfoログが出力されること
	if !strings.Contains(output, "test.txt") {
		t.Errorf("expected log output to contain file path 'test.txt', got: %s", output)
	}
}

func TestActUsecase_Run_VerboseLogsDebugInfo(t *testing.T) {
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

	var buf bytes.Buffer
	logger := slog.New(logging.NewHumanHandler(&buf, &logging.HumanHandlerOptions{Level: slog.LevelDebug, NoColor: true}))

	speed := 1.5
	uc := app.NewActUsecase(reader, client, player, app.WithLogger(logger))
	params := app.ActParams{
		Path:      "test.txt",
		SpeakerID: 3,
		Speed:     &speed,
	}

	err := uc.Run(context.Background(), params)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	output := buf.String()
	// Debug レベルでパラメータ情報が出力されること
	if !strings.Contains(output, "speakerID") {
		t.Errorf("expected verbose log to contain 'speakerID', got: %s", output)
	}
}

func TestActUsecase_Run_NilLogger_NoPanic(t *testing.T) {
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

	// Loggerなしで生成してもパニックしないこと
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

func TestActUsecase_Run_SynthesisCompletedLoggedAtInfoLevel(t *testing.T) {
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

	var buf bytes.Buffer
	logger := slog.New(logging.NewHumanHandler(&buf, &logging.HumanHandlerOptions{Level: slog.LevelInfo, NoColor: true}))

	uc := app.NewActUsecase(reader, client, player, app.WithLogger(logger))
	params := app.ActParams{Path: "test.txt", SpeakerID: 3}

	err := uc.Run(context.Background(), params)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	output := buf.String()
	// synthesis completedがInfoレベルで出力されること
	if !strings.Contains(output, "synthesis completed") {
		t.Errorf("expected log to contain 'synthesis completed' at Info level, got: %s", output)
	}
}

func TestActUsecase_Run_LogsProgressCounter(t *testing.T) {
	reader := &mockScriptReader{
		scripts: []entity.Script{
			{Path: "a.txt", Text: "おはよう", IsEmpty: false},
			{Path: "b.txt", Text: "", IsEmpty: true},
			{Path: "c.txt", Text: "こんにちは", IsEmpty: false},
			{Path: "d.txt", Text: "こんばんは", IsEmpty: false},
		},
	}
	client := &mockVoicevoxClient{
		query:   &entity.AudioQuery{},
		wavData: []byte("fake-wav"),
	}
	player := &mockAudioPlayer{}

	var buf bytes.Buffer
	logger := slog.New(logging.NewHumanHandler(&buf, &logging.HumanHandlerOptions{Level: slog.LevelInfo, NoColor: true}))

	uc := app.NewActUsecase(reader, client, player, app.WithLogger(logger))
	params := app.ActParams{Path: "scripts/", SpeakerID: 3}

	err := uc.Run(context.Background(), params)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	output := buf.String()
	// 非空スクリプトのみカウントされ、[1/3], [2/3], [3/3] が出力される
	if !strings.Contains(output, "[1/3]") {
		t.Errorf("expected log to contain '[1/3]', got: %s", output)
	}
	if !strings.Contains(output, "[2/3]") {
		t.Errorf("expected log to contain '[2/3]', got: %s", output)
	}
	if !strings.Contains(output, "[3/3]") {
		t.Errorf("expected log to contain '[3/3]', got: %s", output)
	}
}

func TestActUsecase_Run_WithNilLogger_NoPanic(t *testing.T) {
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

	// WithLogger(nil)を明示的に渡してもpanicしないこと
	uc := app.NewActUsecase(reader, client, player, app.WithLogger(nil))
	params := app.ActParams{
		Path:      "test.txt",
		SpeakerID: 3,
	}

	err := uc.Run(context.Background(), params)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

// --- dry-run テスト ---

func TestActUsecase_Run_DryRun_SkipsClientAndPlayerAndLogs(t *testing.T) {
	reader := &mockScriptReader{
		scripts: []entity.Script{
			{Path: "a.txt", Text: "おはよう", IsEmpty: false},
			{Path: "b.txt", Text: "", IsEmpty: true},
			{Path: "c.txt", Text: "こんにちは", IsEmpty: false},
		},
	}
	client := &mockVoicevoxClient{
		query:   &entity.AudioQuery{},
		wavData: []byte("fake-wav"),
	}
	player := &mockAudioPlayer{}

	var buf bytes.Buffer
	logger := slog.New(logging.NewHumanHandler(&buf, &logging.HumanHandlerOptions{
		Level:   slog.LevelInfo,
		NoColor: true,
		DryRun:  true,
	}))

	uc := app.NewActUsecase(reader, client, player, app.WithLogger(logger))
	params := app.ActParams{
		Path:      "scripts/",
		SpeakerID: 3,
		DryRun:    true,
	}

	err := uc.Run(context.Background(), params)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if client.createQueryCalls != 0 {
		t.Errorf("expected 0 CreateQuery calls in dry-run, got: %d", client.createQueryCalls)
	}
	if client.synthesizeCalls != 0 {
		t.Errorf("expected 0 Synthesize calls in dry-run, got: %d", client.synthesizeCalls)
	}
	if player.playCalls != 0 {
		t.Errorf("expected 0 Play calls in dry-run, got: %d", player.playCalls)
	}

	output := buf.String()
	// dry-run時のログ: 非dry-run と同じメッセージ名で出力される
	if !strings.Contains(output, "[dry run] [1/2] processing script") {
		t.Errorf("expected '[dry run] [1/2] processing script' in log, got: %s", output)
	}
	if !strings.Contains(output, "[dry run] [2/2] processing script") {
		t.Errorf("expected '[dry run] [2/2] processing script' in log, got: %s", output)
	}
	if !strings.Contains(output, "[dry run] synthesis completed") {
		t.Errorf("expected '[dry run] synthesis completed' in log, got: %s", output)
	}
	if !strings.Contains(output, "wavSize=0") {
		t.Errorf("expected 'wavSize=0' in log, got: %s", output)
	}
	if !strings.Contains(output, "[dry run] playback completed") {
		t.Errorf("expected '[dry run] playback completed' in log, got: %s", output)
	}
	if !strings.Contains(output, "[dry run] all scripts processed") {
		t.Errorf("expected '[dry run] all scripts processed' in log, got: %s", output)
	}
	if !strings.Contains(output, "path=a.txt") {
		t.Errorf("expected 'path=a.txt' in log, got: %s", output)
	}
	if !strings.Contains(output, "text=おはよう") {
		t.Errorf("expected 'text=おはよう' in log, got: %s", output)
	}
	for _, want := range []string{"speaker=3", "speed=default", "pitch=default", "intonation=default"} {
		if !strings.Contains(output, want) {
			t.Errorf("expected %q in log, got: %s", want, output)
		}
	}
}

func TestActUsecase_Run_DryRun_NoHealthCheck(t *testing.T) {
	reader := &mockScriptReader{
		scripts: []entity.Script{
			{Path: "a.txt", Text: "おはよう", IsEmpty: false},
		},
	}
	// HealthCheckがエラーを返してもdry-runでは呼ばれずエラーにならない
	client := &mockVoicevoxClient{
		healthCheckErr: errors.New("connection refused"),
	}
	player := &mockAudioPlayer{}

	uc := app.NewActUsecase(reader, client, player)
	params := app.ActParams{
		Path:      "scripts/",
		SpeakerID: 3,
		DryRun:    true,
	}

	err := uc.Run(context.Background(), params)
	if err != nil {
		t.Fatalf("expected no error in dry-run even with HealthCheck failing, got: %v", err)
	}
}

// --- セリフ単位パラメータ テスト ---

// テストリスト: セリフ単位パラメータ
// DONE: スクリプトにSpeakerIDが設定されている場合、そのSpeakerIDでCreateQuery/Synthesizeが呼ばれる
// DONE: スクリプトにSpeedScale/PitchScale/IntonationScaleが設定されている場合、Synthesizeにそれが渡される
// DONE: スクリプトにパラメータが設定されていない場合、ActParamsのデフォルト値が使われる
// DONE: 複数スクリプトでパラメータが異なる場合、それぞれのスクリプトに対応するパラメータが使われる

func TestActUsecase_Run_ScriptSpeakerID(t *testing.T) {
	scriptSpeaker := 7
	reader := &mockScriptReader{
		scripts: []entity.Script{
			{Path: "script.json", Text: "こんにちは", IsEmpty: false, SpeakerID: &scriptSpeaker},
		},
	}
	client := &mockVoicevoxClient{
		query:   &entity.AudioQuery{},
		wavData: []byte("fake-wav"),
	}
	player := &mockAudioPlayer{}

	uc := app.NewActUsecase(reader, client, player)
	params := app.ActParams{
		Path:      "script.json",
		SpeakerID: 3, // デフォルトは3だがスクリプトで7が指定されている
	}

	err := uc.Run(context.Background(), params)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// CreateQueryにスクリプトのSpeakerID(7)が渡されること
	if len(client.createQueryCallArgs) != 1 {
		t.Fatalf("expected 1 CreateQuery call, got %d", len(client.createQueryCallArgs))
	}
	if client.createQueryCallArgs[0].speakerID != 7 {
		t.Errorf("expected CreateQuery speakerID 7, got %d", client.createQueryCallArgs[0].speakerID)
	}

	// SynthesizeにもスクリプトのSpeakerID(7)が渡されること
	if len(client.synthesizeArgs) != 1 {
		t.Fatalf("expected 1 Synthesize call, got %d", len(client.synthesizeArgs))
	}
	if client.synthesizeArgs[0].speakerID != 7 {
		t.Errorf("expected Synthesize speakerID 7, got %d", client.synthesizeArgs[0].speakerID)
	}
}

func TestActUsecase_Run_ScriptEmotionParams(t *testing.T) {
	speed := 1.5
	pitch := 0.1
	intonation := 1.8
	reader := &mockScriptReader{
		scripts: []entity.Script{
			{
				Path:            "script.json",
				Text:            "感情込めて",
				IsEmpty:         false,
				SpeedScale:      &speed,
				PitchScale:      &pitch,
				IntonationScale: &intonation,
			},
		},
	}
	client := &mockVoicevoxClient{
		query:   &entity.AudioQuery{},
		wavData: []byte("fake-wav"),
	}
	player := &mockAudioPlayer{}

	// グローバルパラメータは未設定
	uc := app.NewActUsecase(reader, client, player)
	params := app.ActParams{
		Path:      "script.json",
		SpeakerID: 3,
	}

	err := uc.Run(context.Background(), params)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(client.synthesizeArgs) != 1 {
		t.Fatalf("expected 1 Synthesize call, got %d", len(client.synthesizeArgs))
	}
	args := client.synthesizeArgs[0]
	// WithOverridesにより、queryのフィールドが上書きされている
	if args.query.SpeedScale != 1.5 {
		t.Errorf("expected SpeedScale 1.5, got %f", args.query.SpeedScale)
	}
	if args.query.PitchScale != 0.1 {
		t.Errorf("expected PitchScale 0.1, got %f", args.query.PitchScale)
	}
	if args.query.IntonationScale != 1.8 {
		t.Errorf("expected IntonationScale 1.8, got %f", args.query.IntonationScale)
	}
}

func TestActUsecase_Run_ScriptParamsOverrideGlobal(t *testing.T) {
	globalSpeed := 2.0
	scriptSpeed := 0.5
	reader := &mockScriptReader{
		scripts: []entity.Script{
			{
				Path:       "script.json",
				Text:       "ゆっくり",
				IsEmpty:    false,
				SpeedScale: &scriptSpeed,
			},
		},
	}
	client := &mockVoicevoxClient{
		query:   &entity.AudioQuery{},
		wavData: []byte("fake-wav"),
	}
	player := &mockAudioPlayer{}

	uc := app.NewActUsecase(reader, client, player)
	params := app.ActParams{
		Path:      "script.json",
		SpeakerID: 3,
		Speed:     &globalSpeed,
	}

	err := uc.Run(context.Background(), params)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(client.synthesizeArgs) != 1 {
		t.Fatalf("expected 1 Synthesize call, got %d", len(client.synthesizeArgs))
	}
	args := client.synthesizeArgs[0]
	// スクリプト単位のSpeedScale(0.5)がグローバル(2.0)より優先される
	if args.query.SpeedScale != 0.5 {
		t.Errorf("expected SpeedScale 0.5 (script override), got %f", args.query.SpeedScale)
	}
}

func TestActUsecase_Run_ScriptNoParams_UsesGlobal(t *testing.T) {
	globalSpeed := 2.0
	reader := &mockScriptReader{
		scripts: []entity.Script{
			{Path: "script.txt", Text: "普通", IsEmpty: false},
		},
	}
	client := &mockVoicevoxClient{
		query:   &entity.AudioQuery{},
		wavData: []byte("fake-wav"),
	}
	player := &mockAudioPlayer{}

	uc := app.NewActUsecase(reader, client, player)
	params := app.ActParams{
		Path:      "script.txt",
		SpeakerID: 3,
		Speed:     &globalSpeed,
	}

	err := uc.Run(context.Background(), params)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(client.synthesizeArgs) != 1 {
		t.Fatalf("expected 1 Synthesize call, got %d", len(client.synthesizeArgs))
	}
	args := client.synthesizeArgs[0]
	// スクリプトにSpeedScaleがないのでグローバル(2.0)が使われる
	if args.query.SpeedScale != 2.0 {
		t.Errorf("expected SpeedScale 2.0 (global), got %f", args.query.SpeedScale)
	}
}

func TestActUsecase_Run_MultipleScriptsWithDifferentParams(t *testing.T) {
	speaker5 := 5
	speed1 := 0.8
	speed2 := 1.5
	reader := &mockScriptReader{
		scripts: []entity.Script{
			{Path: "a.json", Text: "ゆっくり", IsEmpty: false, SpeakerID: &speaker5, SpeedScale: &speed1},
			{Path: "b.txt", Text: "普通", IsEmpty: false},
			{Path: "c.json", Text: "はやく", IsEmpty: false, SpeedScale: &speed2},
		},
	}
	client := &mockVoicevoxClient{
		query:   &entity.AudioQuery{},
		wavData: []byte("fake-wav"),
	}
	player := &mockAudioPlayer{}

	uc := app.NewActUsecase(reader, client, player)
	params := app.ActParams{
		Path:      "scripts/",
		SpeakerID: 3,
	}

	err := uc.Run(context.Background(), params)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(client.createQueryCallArgs) != 3 {
		t.Fatalf("expected 3 CreateQuery calls, got %d", len(client.createQueryCallArgs))
	}
	// a.json: speaker=5
	if client.createQueryCallArgs[0].speakerID != 5 {
		t.Errorf("expected first CreateQuery speakerID 5, got %d", client.createQueryCallArgs[0].speakerID)
	}
	// b.txt: speaker=3 (default)
	if client.createQueryCallArgs[1].speakerID != 3 {
		t.Errorf("expected second CreateQuery speakerID 3, got %d", client.createQueryCallArgs[1].speakerID)
	}
	// c.json: speaker=3 (default, not set in script)
	if client.createQueryCallArgs[2].speakerID != 3 {
		t.Errorf("expected third CreateQuery speakerID 3, got %d", client.createQueryCallArgs[2].speakerID)
	}

	if len(client.synthesizeArgs) != 3 {
		t.Fatalf("expected 3 Synthesize calls, got %d", len(client.synthesizeArgs))
	}
	// a.json: speed=0.8（WithOverridesでqueryに適用済み）
	if client.synthesizeArgs[0].query.SpeedScale != 0.8 {
		t.Errorf("expected first SpeedScale 0.8, got %f", client.synthesizeArgs[0].query.SpeedScale)
	}
	// b.txt: speed未指定（デフォルトのquery値=0.0のまま）
	if client.synthesizeArgs[1].query.SpeedScale != 0.0 {
		t.Errorf("expected second SpeedScale 0.0 (default), got %f", client.synthesizeArgs[1].query.SpeedScale)
	}
	// c.json: speed=1.5
	if client.synthesizeArgs[2].query.SpeedScale != 1.5 {
		t.Errorf("expected third SpeedScale 1.5, got %f", client.synthesizeArgs[2].query.SpeedScale)
	}
}
