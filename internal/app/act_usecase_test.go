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
)

// act_usecase テストリスト
// DONE: 進捗表示: 複数スクリプト処理時にInfoログに「[1/3]」のような進捗が含まれる
// TODO: synthesis completedがInfoレベルで出力される

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
}

type synthesizeCallArgs struct {
	speakerID  int
	speed      *float64
	pitch      *float64
	intonation *float64
}

func (m *mockVoicevoxClient) HealthCheck(_ context.Context) error {
	return m.healthCheckErr
}

func (m *mockVoicevoxClient) CreateQuery(_ context.Context, _ string, speakerID int) (*entity.AudioQuery, error) {
	m.createQueryCalls++
	m.createQueryCallArgs = append(m.createQueryCallArgs, createQueryCallArgs{speakerID: speakerID})
	return m.query, m.createQueryErr
}

func (m *mockVoicevoxClient) Synthesize(_ context.Context, _ *entity.AudioQuery, speakerID int, speed, pitch, intonation *float64) ([]byte, error) {
	m.synthesizeCalls++
	m.synthesizeArgs = append(m.synthesizeArgs, synthesizeCallArgs{speakerID: speakerID, speed: speed, pitch: pitch, intonation: intonation})
	return m.wavData, m.synthesizeErr
}

type mockAudioPlayer struct {
	err       error
	playCalls int
}

func (m *mockAudioPlayer) Play(_ context.Context, _ []byte) error {
	m.playCalls++
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
	if args.speed == nil || *args.speed != 1.5 {
		t.Errorf("expected speed 1.5, got: %v", args.speed)
	}
	if args.pitch == nil || *args.pitch != 0.5 {
		t.Errorf("expected pitch 0.5, got: %v", args.pitch)
	}
	if args.intonation == nil || *args.intonation != 1.2 {
		t.Errorf("expected intonation 1.2, got: %v", args.intonation)
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

func (m *cancellingVoicevoxClient) Synthesize(_ context.Context, _ *entity.AudioQuery, _ int, _, _, _ *float64) ([]byte, error) {
	return m.wavData, nil
}

// cancellingAudioPlayer は指定回数再生後にcontextをキャンセルするモック。
type cancellingAudioPlayer struct {
	playCalls   int
	cancelAfter int
	cancelFunc  context.CancelFunc
}

func (m *cancellingAudioPlayer) Play(_ context.Context, _ []byte) error {
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
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

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
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

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
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

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
	if args.speed == nil || *args.speed != 1.5 {
		t.Errorf("expected speed 1.5, got %v", args.speed)
	}
	if args.pitch == nil || *args.pitch != 0.1 {
		t.Errorf("expected pitch 0.1, got %v", args.pitch)
	}
	if args.intonation == nil || *args.intonation != 1.8 {
		t.Errorf("expected intonation 1.8, got %v", args.intonation)
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
	if args.speed == nil || *args.speed != 0.5 {
		t.Errorf("expected speed 0.5 (script override), got %v", args.speed)
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
	if args.speed == nil || *args.speed != 2.0 {
		t.Errorf("expected speed 2.0 (global), got %v", args.speed)
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
	// a.json: speed=0.8
	if client.synthesizeArgs[0].speed == nil || *client.synthesizeArgs[0].speed != 0.8 {
		t.Errorf("expected first speed 0.8, got %v", client.synthesizeArgs[0].speed)
	}
	// b.txt: speed=nil (no script or global)
	if client.synthesizeArgs[1].speed != nil {
		t.Errorf("expected second speed nil, got %v", *client.synthesizeArgs[1].speed)
	}
	// c.json: speed=1.5
	if client.synthesizeArgs[2].speed == nil || *client.synthesizeArgs[2].speed != 1.5 {
		t.Errorf("expected third speed 1.5, got %v", client.synthesizeArgs[2].speed)
	}
}
