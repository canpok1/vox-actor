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

// say_usecase テストリスト
// DONE: 正常系: テキストで音声合成・再生が成功する
// DONE: 正常系: 音声パラメータ（speed, pitch, intonation）がSynthesizeに正しく渡される
// DONE: 異常系: HealthCheckエラー時にエラーを返す
// DONE: 異常系: CreateQueryエラー時にエラーを返す
// DONE: 異常系: Synthesizeエラー時にエラーを返す
// DONE: 異常系: Playエラー時にエラーを返す
// DONE: 正常系: contextキャンセル時にnilを返す（グレースフルシャットダウン）
// DONE: 正常系: WithSayLogger(nil)でpanicしない

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

func TestSayUsecase_Run_PlayReceivesText(t *testing.T) {
	client := &mockVoicevoxClient{
		query:   &entity.AudioQuery{},
		wavData: []byte("fake-wav"),
	}
	player := &mockAudioPlayer{}

	uc := app.NewSayUsecase(client, player)
	params := app.SayParams{Text: "こんにちはなのだ", SpeakerID: 3}

	if err := uc.Run(context.Background(), params); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(player.playMetas) != 1 || player.playMetas[0].Text != "こんにちはなのだ" {
		t.Errorf("expected PlayMeta.Text=%q, got: %+v", "こんにちはなのだ", player.playMetas)
	}
	if player.playMetas[0].SpeakerID != 3 {
		t.Errorf("expected PlayMeta.SpeakerID=3, got: %d", player.playMetas[0].SpeakerID)
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
	// WithOverridesにより、Synthesizeに渡されるqueryのフィールドが上書きされている
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

func TestWithSayLogger_Nil_NoPanic(t *testing.T) {
	client := &mockVoicevoxClient{
		query:   &entity.AudioQuery{},
		wavData: []byte("fake-wav"),
	}
	player := &mockAudioPlayer{}

	// WithSayLogger(nil) でpanicしないことを確認
	uc := app.NewSayUsecase(client, player, app.WithSayLogger(nil))
	params := app.SayParams{Text: "こんにちは", SpeakerID: 3}

	err := uc.Run(context.Background(), params)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestSayUsecase_Run_DryRun_SkipsClientAndPlayerAndLogs(t *testing.T) {
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

	speed := 1.2
	pitch := 0.1
	intonation := 1.5

	uc := app.NewSayUsecase(client, player, app.WithSayLogger(logger))
	params := app.SayParams{
		Text:       "こんにちは",
		SpeakerID:  3,
		Speed:      &speed,
		Pitch:      &pitch,
		Intonation: &intonation,
		DryRun:     true,
	}

	err := uc.Run(context.Background(), params)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// dry-run時: VoicevoxClient/AudioPlayer のメソッドは一切呼ばれない
	if client.createQueryCalls != 0 {
		t.Errorf("expected 0 CreateQuery calls in dry-run, got: %d", client.createQueryCalls)
	}
	if client.synthesizeCalls != 0 {
		t.Errorf("expected 0 Synthesize calls in dry-run, got: %d", client.synthesizeCalls)
	}
	if player.playCalls != 0 {
		t.Errorf("expected 0 Play calls in dry-run, got: %d", player.playCalls)
	}

	// ログに [dry run] プレフィックスと非dry-run同形式のメッセージが含まれる
	output := buf.String()
	if !strings.Contains(output, "[dry run] synthesis completed") {
		t.Errorf("expected '[dry run] synthesis completed' in log, got: %s", output)
	}
	if !strings.Contains(output, "wavSize=0") {
		t.Errorf("expected 'wavSize=0' in log, got: %s", output)
	}
	if !strings.Contains(output, "[dry run] playback completed") {
		t.Errorf("expected '[dry run] playback completed' in log, got: %s", output)
	}
	for _, want := range []string{"text=こんにちは", "speaker=3", "speed=1.2", "pitch=0.1", "intonation=1.5"} {
		if !strings.Contains(output, want) {
			t.Errorf("expected %q in log, got: %s", want, output)
		}
	}
	// say の dry-run では engine health check passed は出力されない
	if strings.Contains(output, "engine health check passed") {
		t.Errorf("expected no 'engine health check passed' in dry-run, got: %s", output)
	}
}

func TestSayUsecase_Run_DryRun_NoHealthCheck(t *testing.T) {
	// HealthCheckがエラーを返してもdry-runでは呼ばれずエラーにならない
	client := &mockVoicevoxClient{
		healthCheckErr: errors.New("connection refused"),
	}
	player := &mockAudioPlayer{}

	uc := app.NewSayUsecase(client, player)
	params := app.SayParams{
		Text:      "こんにちは",
		SpeakerID: 3,
		DryRun:    true,
	}

	err := uc.Run(context.Background(), params)
	if err != nil {
		t.Fatalf("expected no error in dry-run even with HealthCheck failing, got: %v", err)
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

// --- SaveWavPath テスト (#534) ---
// DONE: 正常系: SaveWavPath 指定時に WavSaver.Save が呼ばれる
// DONE: 正常系: SaveWavPath が空の場合 WavSaver.Save は呼ばれない
// DONE: 正常系: DryRun 時は SaveWavPath 指定でも WavSaver.Save は呼ばれない
// DONE: 異常系: WavSaver.Save がエラーを返したらエラーが返る

type mockWavSaver struct {
	saveCalls int
	savedPath string
	savedData []byte
	err       error
}

func (m *mockWavSaver) Save(path string, data []byte) error {
	m.saveCalls++
	m.savedPath = path
	m.savedData = data
	return m.err
}

func TestSayUsecase_Run_SaveWav_SavesFile(t *testing.T) {
	wavData := []byte("RIFF fake-wav")
	client := &mockVoicevoxClient{
		query:   &entity.AudioQuery{},
		wavData: wavData,
	}
	player := &mockAudioPlayer{}
	saver := &mockWavSaver{}

	uc := app.NewSayUsecase(client, player, app.WithWavSaver(saver))
	params := app.SayParams{
		Text:        "こんにちは",
		SpeakerID:   3,
		SaveWavPath: "/tmp/test.wav",
	}

	if err := uc.Run(context.Background(), params); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if saver.saveCalls != 1 {
		t.Errorf("expected 1 Save call, got: %d", saver.saveCalls)
	}
	if saver.savedPath != "/tmp/test.wav" {
		t.Errorf("expected saved path /tmp/test.wav, got: %s", saver.savedPath)
	}
	if string(saver.savedData) != string(wavData) {
		t.Errorf("expected saved data %q, got %q", wavData, saver.savedData)
	}
	// 保存後も再生は行われる
	if player.playCalls != 1 {
		t.Errorf("expected 1 Play call after save, got: %d", player.playCalls)
	}
}

func TestSayUsecase_Run_SaveWav_EmptyPath_NotSaved(t *testing.T) {
	client := &mockVoicevoxClient{
		query:   &entity.AudioQuery{},
		wavData: []byte("RIFF fake-wav"),
	}
	player := &mockAudioPlayer{}
	saver := &mockWavSaver{}

	uc := app.NewSayUsecase(client, player, app.WithWavSaver(saver))
	params := app.SayParams{
		Text:        "こんにちは",
		SpeakerID:   3,
		SaveWavPath: "",
	}

	if err := uc.Run(context.Background(), params); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if saver.saveCalls != 0 {
		t.Errorf("expected 0 Save calls when path is empty, got: %d", saver.saveCalls)
	}
}

func TestSayUsecase_Run_SaveWav_DryRun_NotSaved(t *testing.T) {
	client := &mockVoicevoxClient{
		query:   &entity.AudioQuery{},
		wavData: []byte("RIFF fake-wav"),
	}
	player := &mockAudioPlayer{}
	saver := &mockWavSaver{}

	uc := app.NewSayUsecase(client, player, app.WithWavSaver(saver))
	params := app.SayParams{
		Text:        "こんにちは",
		SpeakerID:   3,
		SaveWavPath: "/tmp/test.wav",
		DryRun:      true,
	}

	if err := uc.Run(context.Background(), params); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if saver.saveCalls != 0 {
		t.Errorf("expected 0 Save calls in dry-run, got: %d", saver.saveCalls)
	}
}

func TestSayUsecase_Run_SaveWav_SaveError_ReturnsError(t *testing.T) {
	client := &mockVoicevoxClient{
		query:   &entity.AudioQuery{},
		wavData: []byte("RIFF fake-wav"),
	}
	player := &mockAudioPlayer{}
	saver := &mockWavSaver{err: errors.New("disk full")}

	uc := app.NewSayUsecase(client, player, app.WithWavSaver(saver))
	params := app.SayParams{
		Text:        "こんにちは",
		SpeakerID:   3,
		SaveWavPath: "/tmp/test.wav",
	}

	if err := uc.Run(context.Background(), params); err == nil {
		t.Fatal("expected error when WavSaver.Save fails, got nil")
	}
}
