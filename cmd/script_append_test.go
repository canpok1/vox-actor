package cmd

import (
	"bytes"
	"errors"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"

	"github.com/canpok1/vox-actor/internal/app"
	"github.com/canpok1/vox-actor/internal/domain/entity"
	"github.com/spf13/cobra"
)

// script append コマンド テストリスト
// DONE: scriptサブコマンドがrootに登録されている
// DONE: script appendサブコマンドがscriptに登録されている
// DONE: 引数なし（fileのみ）でErrUsageを返す
// DONE: 引数なし（fileもtextもなし）でErrUsageを返す
// DONE: ヘルプ出力に--speaker/--speed/--pitch/--intonationが含まれる
// DONE: ヘルプ出力に--engine-url/--dry-runが含まれない
// DONE: デフォルトオプション値の確認（speaker, speed, pitch, intonation）
// DONE: 環境変数VOX_SPEAKERがデフォルト値に反映される
// DONE: WriterFactoryがnilのときエラーになる
// DONE: Writer.Writeが呼ばれる
// DONE: --speaker/--speed/--pitch/--intonation が ScriptWriter に伝わる

func findScriptCmd(t *testing.T, rootCmd *cobra.Command) *cobra.Command {
	t.Helper()
	for _, c := range rootCmd.Commands() {
		if c.Name() == "script" {
			return c
		}
	}
	t.Fatal("script subcommand not found")
	return nil
}

func findScriptAppendCmd(t *testing.T, rootCmd *cobra.Command) *cobra.Command {
	t.Helper()
	scriptCmd := findScriptCmd(t, rootCmd)
	for _, c := range scriptCmd.Commands() {
		if c.Name() == "append" {
			return c
		}
	}
	t.Fatal("script append subcommand not found")
	return nil
}

func TestScriptCmd_RegisteredAsSubcommand(t *testing.T) {
	rootCmd := makeRootCmd()
	found := false
	for _, c := range rootCmd.Commands() {
		if c.Name() == "script" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'script' subcommand to be registered")
	}
}

func TestScriptAppendCmd_RegisteredUnderScript(t *testing.T) {
	rootCmd := makeRootCmd()
	scriptCmd := findScriptCmd(t, rootCmd)
	found := false
	for _, c := range scriptCmd.Commands() {
		if c.Name() == "append" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'append' subcommand under 'script'")
	}
}

func TestScriptAppendCmd_NoArgs_ReturnsUsageError(t *testing.T) {
	rootCmd := makeRootCmd()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"script", "append"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when no args provided, got nil")
	}
	if !errors.Is(err, ErrUsage) {
		t.Errorf("expected ErrUsage, got: %v", err)
	}
}

func TestScriptAppendCmd_OneArg_ReturnsUsageError(t *testing.T) {
	rootCmd := makeRootCmd()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"script", "append", "file.jsonl"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when only one arg provided, got nil")
	}
	if !errors.Is(err, ErrUsage) {
		t.Errorf("expected ErrUsage, got: %v", err)
	}
}

func TestScriptAppendCmd_HelpContainsFlags(t *testing.T) {
	rootCmd := makeRootCmd()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"script", "append", "--help"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error for --help, got: %v", err)
	}

	output := buf.String()
	flags := []string{"--speaker", "--speed", "--pitch", "--intonation", "--verbose"}
	for _, flag := range flags {
		if !strings.Contains(output, flag) {
			t.Errorf("expected help output to contain '%s'", flag)
		}
	}
}

func TestScriptAppendCmd_HelpDoesNotContainEngineURLOrDryRun(t *testing.T) {
	rootCmd := makeRootCmd()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"script", "append", "--help"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error for --help, got: %v", err)
	}

	output := buf.String()
	for _, flag := range []string{"--engine-url", "--dry-run"} {
		if strings.Contains(output, flag) {
			t.Errorf("expected help output NOT to contain '%s'", flag)
		}
	}
}

func TestScriptAppendCmd_DefaultOptionValues(t *testing.T) {
	rootCmd := makeRootCmd()
	appendCmd := findScriptAppendCmd(t, rootCmd)

	speaker, _ := appendCmd.Flags().GetInt("speaker")
	if speaker != 3 {
		t.Errorf("expected default speaker 3, got: %d", speaker)
	}
	speed, _ := appendCmd.Flags().GetFloat64("speed")
	if speed != 1.0 {
		t.Errorf("expected default speed 1.0, got: %f", speed)
	}
	pitch, _ := appendCmd.Flags().GetFloat64("pitch")
	if pitch != 0.0 {
		t.Errorf("expected default pitch 0.0, got: %f", pitch)
	}
	intonation, _ := appendCmd.Flags().GetFloat64("intonation")
	if intonation != 1.0 {
		t.Errorf("expected default intonation 1.0, got: %f", intonation)
	}
}

func TestScriptAppendCmd_EnvVarVOXSpeaker(t *testing.T) {
	t.Setenv("VOX_SPEAKER", "42")
	rootCmd := makeRootCmd()
	appendCmd := findScriptAppendCmd(t, rootCmd)

	speaker, _ := appendCmd.Flags().GetInt("speaker")
	if speaker != 42 {
		t.Errorf("expected speaker 42, got: %d", speaker)
	}
}

func TestScriptAppendCmd_NilWriterFactory_ReturnsError(t *testing.T) {
	deps := &Deps{
		ScriptAppend: &ScriptAppendDeps{
			WriterFactory: nil,
		},
	}
	rootCmd := makeRootCmd(deps)
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	tmpDir := t.TempDir()
	rootCmd.SetArgs([]string{"script", "append", filepath.Join(tmpDir, "out.jsonl"), "hello"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when WriterFactory is nil, got nil")
	}
}

type captureScriptWriter struct {
	calls []captureWriterCall
}

type captureWriterCall struct {
	path   string
	script entity.Script
}

func (w *captureScriptWriter) Write(path string, script entity.Script) (string, error) {
	w.calls = append(w.calls, captureWriterCall{path: path, script: script})
	return path, nil
}

func TestScriptAppendCmd_CallsWriter(t *testing.T) {
	writer := &captureScriptWriter{}
	deps := &Deps{
		ScriptAppend: &ScriptAppendDeps{
			WriterFactory: func(_ *slog.Logger) app.ScriptWriter { return writer },
		},
	}
	rootCmd := makeRootCmd(deps)

	tmpDir := t.TempDir()
	dest := filepath.Join(tmpDir, "out.jsonl")
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"script", "append", dest, "hello"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(writer.calls) != 1 {
		t.Fatalf("expected 1 Write call, got: %d", len(writer.calls))
	}
	if writer.calls[0].path != dest {
		t.Errorf("expected path=%q, got %q", dest, writer.calls[0].path)
	}
	if writer.calls[0].script.Text != "hello" {
		t.Errorf("expected text=hello, got %q", writer.calls[0].script.Text)
	}
}

func TestScriptAppendCmd_FlagsPassedToWriter(t *testing.T) {
	writer := &captureScriptWriter{}
	deps := &Deps{
		ScriptAppend: &ScriptAppendDeps{
			WriterFactory: func(_ *slog.Logger) app.ScriptWriter { return writer },
		},
	}
	rootCmd := makeRootCmd(deps)

	tmpDir := t.TempDir()
	dest := filepath.Join(tmpDir, "out.jsonl")
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{
		"script", "append",
		"--speaker", "5",
		"--speed", "1.2",
		"--pitch", "0.05",
		"--intonation", "1.5",
		dest, "hello",
	})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(writer.calls) != 1 {
		t.Fatalf("expected 1 Write call, got: %d", len(writer.calls))
	}
	s := writer.calls[0].script
	if s.SpeakerID == nil || s.SpeakerID.Value() != 5 {
		t.Errorf("expected SpeakerID=5, got: %v", s.SpeakerID)
	}
	if s.Overrides.Speed == nil || *s.Overrides.Speed != 1.2 {
		t.Errorf("expected Overrides.Speed=1.2, got: %v", s.Overrides.Speed)
	}
	if s.Overrides.Pitch == nil || *s.Overrides.Pitch != 0.05 {
		t.Errorf("expected Overrides.Pitch=0.05, got: %v", s.Overrides.Pitch)
	}
	if s.Overrides.Intonation == nil || *s.Overrides.Intonation != 1.5 {
		t.Errorf("expected Overrides.Intonation=1.5, got: %v", s.Overrides.Intonation)
	}
}
