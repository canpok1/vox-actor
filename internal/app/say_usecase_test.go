package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/canpok1/vox-actor/internal/app"
	"github.com/canpok1/vox-actor/internal/domain/entity"
)

// say_usecase テストリスト
// DONE: 正常系: テキストで音声合成・再生が成功する
// DONE: 正常系: 音声パラメータ（speed, pitch, intonation）がSynthesizeに正しく渡される
// DONE: 異常系: HealthCheckエラー時にエラーを返す
// DONE: 異常系: CreateQueryエラー時にエラーを返す
// DONE: 異常系: Synthesizeエラー時にエラーを返す
// DONE: 異常系: Playエラー時にエラーを返す
// DONE: 正常系: contextキャンセル時にnilを返す（グレースフルシャットダウン）

func TestSayUsecase_Run_Success(t *testing.T) {
	client := &mockVoicevoxClient{
		query:   &entity.AudioQuery{},
		wavData: []byte("fake-wav"),
	}
	player := &mockAudioPlayer{}

	uc := app.NewSayUsecase(client, player)
	params := app.SayParams{
		Text:      "こんにちは",
		SpeakerID: 3,
	}

	err := uc.Run(context.Background(), params)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if client.createQueryCalls != 1 {
		t.Errorf("expected 1 CreateQuery call, got: %d", client.createQueryCalls)
	}
	if client.synthesizeCalls != 1 {
		t.Errorf("expected 1 Synthesize call, got: %d", client.synthesizeCalls)
	}
	if player.playCalls != 1 {
		t.Errorf("expected 1 Play call, got: %d", player.playCalls)
	}
}

func TestSayUsecase_Run_WithParams(t *testing.T) {
	client := &mockVoicevoxClient{
		query:   &entity.AudioQuery{},
		wavData: []byte("fake-wav"),
	}
	player := &mockAudioPlayer{}

	speed := 1.5
	pitch := 0.5
	intonation := 1.2

	uc := app.NewSayUsecase(client, player)
	params := app.SayParams{
		Text:       "こんにちは",
		SpeakerID:  5,
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
	if args.speakerID != 5 {
		t.Errorf("expected speakerID 5, got: %d", args.speakerID)
	}
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

func TestSayUsecase_Run_HealthCheckError(t *testing.T) {
	client := &mockVoicevoxClient{
		healthCheckErr: errors.New("connection refused"),
	}
	player := &mockAudioPlayer{}

	uc := app.NewSayUsecase(client, player)
	params := app.SayParams{Text: "こんにちは", SpeakerID: 3}

	err := uc.Run(context.Background(), params)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestSayUsecase_Run_CreateQueryError(t *testing.T) {
	client := &mockVoicevoxClient{
		createQueryErr: errors.New("query failed"),
	}
	player := &mockAudioPlayer{}

	uc := app.NewSayUsecase(client, player)
	params := app.SayParams{Text: "こんにちは", SpeakerID: 3}

	err := uc.Run(context.Background(), params)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestSayUsecase_Run_SynthesizeError(t *testing.T) {
	client := &mockVoicevoxClient{
		query:         &entity.AudioQuery{},
		synthesizeErr: errors.New("synthesis failed"),
	}
	player := &mockAudioPlayer{}

	uc := app.NewSayUsecase(client, player)
	params := app.SayParams{Text: "こんにちは", SpeakerID: 3}

	err := uc.Run(context.Background(), params)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestSayUsecase_Run_PlayError(t *testing.T) {
	client := &mockVoicevoxClient{
		query:   &entity.AudioQuery{},
		wavData: []byte("fake-wav"),
	}
	player := &mockAudioPlayer{
		err: errors.New("play failed"),
	}

	uc := app.NewSayUsecase(client, player)
	params := app.SayParams{Text: "こんにちは", SpeakerID: 3}

	err := uc.Run(context.Background(), params)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestSayUsecase_Run_CancelledContext_ReturnsNil(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 即座にキャンセル

	client := &mockVoicevoxClient{
		healthCheckErr: context.Canceled,
	}
	player := &mockAudioPlayer{}

	uc := app.NewSayUsecase(client, player)
	params := app.SayParams{Text: "こんにちは", SpeakerID: 3}

	err := uc.Run(ctx, params)
	// キャンセル済みcontextの場合、グレースフルシャットダウンとしてnilを返すべき
	if err != nil {
		t.Fatalf("expected nil for cancelled context, got: %v", err)
	}
}
