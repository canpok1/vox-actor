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
)

// script write コマンド テストリスト
// DONE: script writeサブコマンドがscriptに登録されている
// DONE: --json なしでErrUsageを返す
// DONE: --json が不正なJSONのときErrUsageを返す
// DONE: WriterFactoryがnilのときエラーになる
// DONE: WriteAllが呼ばれる
// DONE: JSON配列の各フィールドがScriptに正しく変換される
// DONE: text=""の要素はIsEmpty=trueとして扱われる
// DONE: ヘルプ出力に--verboseが含まれ--engine-url/--dry-runが含まれない

type captureScriptBulkWriter struct {
	calls []captureBulkWriterCall
}

type captureBulkWriterCall struct {
	path    string
	scripts []entity.Script
}

func (w *captureScriptBulkWriter) WriteAll(path string, scripts []entity.Script) (string, error) {
	w.calls = append(w.calls, captureBulkWriterCall{path: path, scripts: scripts})
	return path, nil
}

func TestScriptWriteCmd_RegisteredUnderScript(t *testing.T) {
	rootCmd := makeRootCmd()
	scriptCmd := findScriptCmd(t, rootCmd)
	found := false
	for _, c := range scriptCmd.Commands() {
		if c.Name() == "write" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'write' subcommand under 'script'")
	}
}

func TestScriptWriteCmd_NoJsonFlag_ReturnsUsageError(t *testing.T) {
	rootCmd := makeRootCmd()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	tmpDir := t.TempDir()
	rootCmd.SetArgs([]string{"script", "write", filepath.Join(tmpDir, "out.jsonl")})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when --json is missing, got nil")
	}
	if !errors.Is(err, ErrUsage) {
		t.Errorf("expected ErrUsage, got: %v", err)
	}
}

func TestScriptWriteCmd_InvalidJson_ReturnsUsageError(t *testing.T) {
	writer := &captureScriptBulkWriter{}
	deps := &Deps{
		ScriptWrite: &ScriptWriteDeps{
			WriterFactory: func(_ *slog.Logger) app.ScriptBulkWriter { return writer },
		},
	}
	rootCmd := makeRootCmd(deps)
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	tmpDir := t.TempDir()
	rootCmd.SetArgs([]string{"script", "write", "--json", "not-valid-json", filepath.Join(tmpDir, "out.jsonl")})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
	if !errors.Is(err, ErrUsage) {
		t.Errorf("expected ErrUsage, got: %v", err)
	}
}

func TestScriptWriteCmd_NilWriterFactory_ReturnsError(t *testing.T) {
	deps := &Deps{
		ScriptWrite: &ScriptWriteDeps{
			WriterFactory: nil,
		},
	}
	rootCmd := makeRootCmd(deps)
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	tmpDir := t.TempDir()
	rootCmd.SetArgs([]string{
		"script", "write",
		"--json", `[{"text":"hello"}]`,
		filepath.Join(tmpDir, "out.jsonl"),
	})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when WriterFactory is nil, got nil")
	}
}

func TestScriptWriteCmd_CallsWriteAll(t *testing.T) {
	writer := &captureScriptBulkWriter{}
	deps := &Deps{
		ScriptWrite: &ScriptWriteDeps{
			WriterFactory: func(_ *slog.Logger) app.ScriptBulkWriter { return writer },
		},
	}
	rootCmd := makeRootCmd(deps)

	tmpDir := t.TempDir()
	dest := filepath.Join(tmpDir, "out.jsonl")
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{
		"script", "write",
		"--json", `[{"text":"hello"}]`,
		dest,
	})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(writer.calls) != 1 {
		t.Fatalf("expected 1 WriteAll call, got: %d", len(writer.calls))
	}
	if writer.calls[0].path != dest {
		t.Errorf("expected path=%q, got %q", dest, writer.calls[0].path)
	}
	if len(writer.calls[0].scripts) != 1 {
		t.Fatalf("expected 1 script, got: %d", len(writer.calls[0].scripts))
	}
	if writer.calls[0].scripts[0].Text != "hello" {
		t.Errorf("expected text=hello, got %q", writer.calls[0].scripts[0].Text)
	}
}

func TestScriptWriteCmd_JsonFieldsMappedToScript(t *testing.T) {
	writer := &captureScriptBulkWriter{}
	deps := &Deps{
		ScriptWrite: &ScriptWriteDeps{
			WriterFactory: func(_ *slog.Logger) app.ScriptBulkWriter { return writer },
		},
	}
	rootCmd := makeRootCmd(deps)

	tmpDir := t.TempDir()
	dest := filepath.Join(tmpDir, "out.jsonl")
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{
		"script", "write",
		"--json", `[{"text":"セリフ","speaker":3,"speed":1.2,"pitch":0.05,"intonation":1.1}]`,
		dest,
	})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(writer.calls) != 1 || len(writer.calls[0].scripts) != 1 {
		t.Fatalf("unexpected calls: %v", writer.calls)
	}
	s := writer.calls[0].scripts[0]
	if s.Text != "セリフ" {
		t.Errorf("Text: expected 'セリフ', got %q", s.Text)
	}
	if s.SpeakerID == nil || *s.SpeakerID != 3 {
		t.Errorf("SpeakerID: expected 3, got %v", s.SpeakerID)
	}
	if s.SpeedScale == nil || *s.SpeedScale != 1.2 {
		t.Errorf("SpeedScale: expected 1.2, got %v", s.SpeedScale)
	}
	if s.PitchScale == nil || *s.PitchScale != 0.05 {
		t.Errorf("PitchScale: expected 0.05, got %v", s.PitchScale)
	}
	if s.IntonationScale == nil || *s.IntonationScale != 1.1 {
		t.Errorf("IntonationScale: expected 1.1, got %v", s.IntonationScale)
	}
}

func TestScriptWriteCmd_EmptyTextIsEmpty(t *testing.T) {
	writer := &captureScriptBulkWriter{}
	deps := &Deps{
		ScriptWrite: &ScriptWriteDeps{
			WriterFactory: func(_ *slog.Logger) app.ScriptBulkWriter { return writer },
		},
	}
	rootCmd := makeRootCmd(deps)

	tmpDir := t.TempDir()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{
		"script", "write",
		"--json", `[{"text":""}]`,
		filepath.Join(tmpDir, "out.jsonl"),
	})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(writer.calls) != 1 || len(writer.calls[0].scripts) != 1 {
		t.Fatalf("unexpected calls: %v", writer.calls)
	}
	if !writer.calls[0].scripts[0].IsEmpty {
		t.Error("expected IsEmpty=true for empty text")
	}
}

func TestScriptWriteCmd_HelpContainsVerboseNotEngineURL(t *testing.T) {
	rootCmd := makeRootCmd()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"script", "write", "--help"})

	_ = rootCmd.Execute()

	output := buf.String()
	if !strings.Contains(output, "--verbose") {
		t.Errorf("expected help to contain '--verbose'\nstdout:\n%s", output)
	}
	if !strings.Contains(output, "--json") {
		t.Errorf("expected help to contain '--json'\nstdout:\n%s", output)
	}
	for _, flag := range []string{"--engine-url", "--dry-run"} {
		if strings.Contains(output, flag) {
			t.Errorf("expected help NOT to contain %q\nstdout:\n%s", flag, output)
		}
	}
}
