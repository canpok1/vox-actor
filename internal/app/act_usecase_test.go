package app_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/canpok1/vox-actor/internal/app"
	"github.com/canpok1/vox-actor/internal/domain/entity"
)

// act_usecase テストリスト（すべて実装済み）

// --- モック ---

type mockScriptReader struct {
	scripts []entity.Script
	err     error
}

func (m *mockScriptReader) Read(_ string) ([]entity.Script, error) {
	return m.scripts, m.err
}

type mockVoicevoxClient struct {
	healthCheckErr   error
	query            *entity.AudioQuery
	createQueryErr   error
	wavData          []byte
	synthesizeErr    error
	createQueryCalls int
	synthesizeCalls  int
	synthesizeArgs   []synthesizeCallArgs
}

type synthesizeCallArgs struct {
	speed      *float64
	pitch      *float64
	intonation *float64
}

func (m *mockVoicevoxClient) HealthCheck(_ context.Context) error {
	return m.healthCheckErr
}

func (m *mockVoicevoxClient) CreateQuery(_ context.Context, _ string, _ int) (*entity.AudioQuery, error) {
	m.createQueryCalls++
	return m.query, m.createQueryErr
}

func (m *mockVoicevoxClient) Synthesize(_ context.Context, _ *entity.AudioQuery, _ int, speed, pitch, intonation *float64) ([]byte, error) {
	m.synthesizeCalls++
	m.synthesizeArgs = append(m.synthesizeArgs, synthesizeCallArgs{speed: speed, pitch: pitch, intonation: intonation})
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
