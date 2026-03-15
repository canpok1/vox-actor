package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

// 共通フラグヘルパー テストリスト #117
// DONE: RegisterCommonFlagsでengine-urlフラグが登録される
// DONE: RegisterCommonFlagsでspeakerフラグが登録される
// DONE: RegisterCommonFlagsでspeedフラグが登録される（デフォルト1.0）
// DONE: RegisterCommonFlagsでpitchフラグが登録される（デフォルト0.0）
// DONE: RegisterCommonFlagsでintonationフラグが登録される（デフォルト1.0）
// DONE: RegisterCommonFlagsでverboseフラグが登録される（デフォルトfalse）
// DONE: VOX_ENGINE_URL環境変数がengine-urlのデフォルト値に反映される
// DONE: VOX_SPEAKER環境変数がspeakerのデフォルト値に反映される
// DONE: VOX_SPEAKERに不正な値が設定されている場合デフォルト値3が使われる
// DONE: actコマンドがRegisterCommonFlagsを使用してもフラグ挙動が変わらない
// DONE: sayコマンドがRegisterCommonFlagsを使用してもフラグ挙動が変わらない

func TestRegisterCommonFlags_EnvVarEngineURL(t *testing.T) {
	t.Setenv("VOX_ENGINE_URL", "http://custom:8888")

	cmd := &cobra.Command{Use: "test"}
	registerCommonFlags(cmd)

	engineURL, _ := cmd.Flags().GetString("engine-url")
	if engineURL != "http://custom:8888" {
		t.Errorf("expected engine-url 'http://custom:8888', got: %s", engineURL)
	}
}

func TestRegisterCommonFlags_EnvVarSpeaker(t *testing.T) {
	t.Setenv("VOX_SPEAKER", "10")

	cmd := &cobra.Command{Use: "test"}
	registerCommonFlags(cmd)

	speaker, _ := cmd.Flags().GetInt("speaker")
	if speaker != 10 {
		t.Errorf("expected speaker 10, got: %d", speaker)
	}
}

func TestRegisterCommonFlags_EnvVarSpeaker_InvalidValue(t *testing.T) {
	t.Setenv("VOX_SPEAKER", "abc")

	cmd := &cobra.Command{Use: "test"}
	registerCommonFlags(cmd)

	speaker, _ := cmd.Flags().GetInt("speaker")
	if speaker != 3 {
		t.Errorf("expected default speaker 3 when env var is invalid, got: %d", speaker)
	}
}

func TestRegisterCommonFlags_Speaker(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	registerCommonFlags(cmd)

	speaker, err := cmd.Flags().GetInt("speaker")
	if err != nil {
		t.Fatalf("expected speaker flag to exist, got error: %v", err)
	}
	if speaker != 3 {
		t.Errorf("expected default speaker 3, got: %d", speaker)
	}
}

func TestRegisterCommonFlags_Speed(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	registerCommonFlags(cmd)

	speed, err := cmd.Flags().GetFloat64("speed")
	if err != nil {
		t.Fatalf("expected speed flag to exist, got error: %v", err)
	}
	if speed != 1.0 {
		t.Errorf("expected default speed 1.0, got: %f", speed)
	}
}

func TestRegisterCommonFlags_Pitch(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	registerCommonFlags(cmd)

	pitch, err := cmd.Flags().GetFloat64("pitch")
	if err != nil {
		t.Fatalf("expected pitch flag to exist, got error: %v", err)
	}
	if pitch != 0.0 {
		t.Errorf("expected default pitch 0.0, got: %f", pitch)
	}
}

func TestRegisterCommonFlags_Intonation(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	registerCommonFlags(cmd)

	intonation, err := cmd.Flags().GetFloat64("intonation")
	if err != nil {
		t.Fatalf("expected intonation flag to exist, got error: %v", err)
	}
	if intonation != 1.0 {
		t.Errorf("expected default intonation 1.0, got: %f", intonation)
	}
}

func TestRegisterCommonFlags_Verbose(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	registerCommonFlags(cmd)

	verbose, err := cmd.Flags().GetBool("verbose")
	if err != nil {
		t.Fatalf("expected verbose flag to exist, got error: %v", err)
	}
	if verbose {
		t.Error("expected default verbose to be false")
	}
}

func TestRegisterCommonFlags_EngineURL(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	registerCommonFlags(cmd)

	engineURL, err := cmd.Flags().GetString("engine-url")
	if err != nil {
		t.Fatalf("expected engine-url flag to exist, got error: %v", err)
	}
	if engineURL != "http://localhost:50021" {
		t.Errorf("expected default engine-url 'http://localhost:50021', got: %s", engineURL)
	}
}
