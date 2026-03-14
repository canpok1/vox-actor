package cmd

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// act コマンド テストリスト（すべて実装済み）

// グレースフルシャットダウン テストリスト
// DONE: actコマンドのcontextにシグナルハンドリングが設定されていることを確認

// 環境変数対応 テストリスト #59
// DONE: VOX_ENGINE_URL環境変数が設定されている場合、--engine-urlのデフォルト値として反映される
// DONE: VOX_SPEAKER環境変数が設定されている場合、--speakerのデフォルト値として反映される
// TODO: CLIフラグ --engine-url 指定時はVOX_ENGINE_URLより優先される
// TODO: CLIフラグ --speaker 指定時はVOX_SPEAKERより優先される
// TODO: 環境変数未設定時はデフォルト値が使われる（既存テストでカバー済み）
// TODO: VOX_SPEAKER に不正な値（数値でない）が設定されている場合エラーを返す

func TestActCmd_RegisteredAsSubcommand(t *testing.T) {
	rootCmd := makeRootCmd()

	found := false
	for _, c := range rootCmd.Commands() {
		if c.Name() == "act" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'act' subcommand to be registered")
	}
}

func TestActCmd_DefaultOptionValues(t *testing.T) {
	rootCmd := makeRootCmd()

	var actCmd *cobra.Command
	for _, c := range rootCmd.Commands() {
		if c.Name() == "act" {
			actCmd = c
			break
		}
	}
	if actCmd == nil {
		t.Fatal("act subcommand not found")
	}

	engineURL, _ := actCmd.Flags().GetString("engine-url")
	if engineURL != "http://localhost:50021" {
		t.Errorf("expected default engine-url 'http://localhost:50021', got: %s", engineURL)
	}

	speaker, _ := actCmd.Flags().GetInt("speaker")
	if speaker != 3 {
		t.Errorf("expected default speaker 3, got: %d", speaker)
	}

	speed, _ := actCmd.Flags().GetFloat64("speed")
	if speed != 1.0 {
		t.Errorf("expected default speed 1.0, got: %f", speed)
	}

	pitch, _ := actCmd.Flags().GetFloat64("pitch")
	if pitch != 0.0 {
		t.Errorf("expected default pitch 0.0, got: %f", pitch)
	}

	intonation, _ := actCmd.Flags().GetFloat64("intonation")
	if intonation != 1.0 {
		t.Errorf("expected default intonation 1.0, got: %f", intonation)
	}
}

func TestActCmd_NoArgs_ReturnsUsageError(t *testing.T) {
	rootCmd := makeRootCmd()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"act"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when no args provided, got nil")
	}
	if !errors.Is(err, ErrUsage) {
		t.Errorf("expected ErrUsage, got: %v", err)
	}
}

func TestActCmd_HelpContainsFlags(t *testing.T) {
	rootCmd := makeRootCmd()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"act", "--help"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error for --help, got: %v", err)
	}

	output := buf.String()
	flags := []string{"--engine-url", "--speaker", "--speed", "--pitch", "--intonation", "--watch"}
	for _, flag := range flags {
		if !strings.Contains(output, flag) {
			t.Errorf("expected help output to contain '%s'", flag)
		}
	}
}

func TestActCmd_WatchWithFile_ReturnsError(t *testing.T) {
	// --watchフラグ付きでファイルパスを指定した場合はエラー
	dir := t.TempDir()
	file := dir + "/test.txt"
	if err := os.WriteFile(file, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	rootCmd := makeRootCmd()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"act", "--watch", file})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when --watch is used with a file path")
	}
	if !errors.Is(err, ErrUsage) {
		t.Errorf("expected ErrUsage, got: %v", err)
	}
}

func TestActCmd_WatchFlag_DefaultFalse(t *testing.T) {
	rootCmd := makeRootCmd()

	var actCmd *cobra.Command
	for _, c := range rootCmd.Commands() {
		if c.Name() == "act" {
			actCmd = c
			break
		}
	}
	if actCmd == nil {
		t.Fatal("act subcommand not found")
	}

	watch, err := actCmd.Flags().GetBool("watch")
	if err != nil {
		t.Fatalf("expected --watch flag to exist, got error: %v", err)
	}
	if watch {
		t.Error("expected --watch default to be false")
	}
}

func TestActCmd_EnvVarVOXEngineURL(t *testing.T) {
	t.Setenv("VOX_ENGINE_URL", "http://custom:9999")

	rootCmd := makeRootCmd()

	var actCmd *cobra.Command
	for _, c := range rootCmd.Commands() {
		if c.Name() == "act" {
			actCmd = c
			break
		}
	}
	if actCmd == nil {
		t.Fatal("act subcommand not found")
	}

	engineURL, _ := actCmd.Flags().GetString("engine-url")
	if engineURL != "http://custom:9999" {
		t.Errorf("expected engine-url to be 'http://custom:9999', got: %s", engineURL)
	}
}

func TestActCmd_EnvVarVOXSpeaker(t *testing.T) {
	t.Setenv("VOX_SPEAKER", "42")

	rootCmd := makeRootCmd()

	var actCmd *cobra.Command
	for _, c := range rootCmd.Commands() {
		if c.Name() == "act" {
			actCmd = c
			break
		}
	}
	if actCmd == nil {
		t.Fatal("act subcommand not found")
	}

	speaker, _ := actCmd.Flags().GetInt("speaker")
	if speaker != 42 {
		t.Errorf("expected speaker to be 42, got: %d", speaker)
	}
}
