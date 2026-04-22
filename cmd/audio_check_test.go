package cmd

// audio-check コマンド テストリスト
// DONE: audio-checkサブコマンドがrootに登録されている
// DONE: 正常系 / non-verbose: 終了コード0、stdout/stderrともに空
// DONE: 正常系 / verbose: stderrに "audio backend: <name>" が出力される
// DONE: 異常系 / non-verbose: エラーを返すがstdout/stderrともに空
// DONE: 異常系 / verbose: stderrに "Error: <reason>" が出力される
// DONE: 未知のフラグはエラーを返す
// DONE: 引数を受け付けない（位置引数がある場合エラー）
// DONE: --help でコマンド説明が出力される
// DONE: 依存が未設定だとエラーを返す（セーフティネット）

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func newAudioCheckDepsSuccess(backend string) *AudioCheckDeps {
	return &AudioCheckDeps{
		CheckFunc: func() (string, error) {
			return backend, nil
		},
	}
}

func newAudioCheckDepsFailure(reason string) *AudioCheckDeps {
	return &AudioCheckDeps{
		CheckFunc: func() (string, error) {
			return "", errors.New(reason)
		},
	}
}

func TestAudioCheckCmd_RegisteredAsSubcommand(t *testing.T) {
	rootCmd := makeRootCmd()

	found := false
	for _, c := range rootCmd.Commands() {
		if c.Name() == "audio-check" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'audio-check' subcommand to be registered")
	}
}

func TestAudioCheckCmd_Success_NoVerbose_SilentOutput(t *testing.T) {
	deps := &Deps{AudioCheck: newAudioCheckDepsSuccess("coreaudio")}
	rootCmd := makeRootCmd(deps)
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rootCmd.SetOut(outBuf)
	rootCmd.SetErr(errBuf)
	rootCmd.SetArgs([]string{"audio-check"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v (stderr=%q)", err, errBuf.String())
	}
	if got := outBuf.String(); got != "" {
		t.Errorf("expected empty stdout, got: %q", got)
	}
	if got := errBuf.String(); got != "" {
		t.Errorf("expected empty stderr, got: %q", got)
	}
}

func TestAudioCheckCmd_Success_Verbose_StderrHasBackend(t *testing.T) {
	deps := &Deps{AudioCheck: newAudioCheckDepsSuccess("coreaudio")}
	rootCmd := makeRootCmd(deps)
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rootCmd.SetOut(outBuf)
	rootCmd.SetErr(errBuf)
	rootCmd.SetArgs([]string{"audio-check", "--verbose"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v (stderr=%q)", err, errBuf.String())
	}
	if got := outBuf.String(); got != "" {
		t.Errorf("expected empty stdout, got: %q", got)
	}
	want := "audio backend: coreaudio"
	if !strings.Contains(errBuf.String(), want) {
		t.Errorf("expected stderr to contain %q, got: %q", want, errBuf.String())
	}
}

func TestAudioCheckCmd_Success_VerboseShortFlag(t *testing.T) {
	deps := &Deps{AudioCheck: newAudioCheckDepsSuccess("pulseaudio")}
	rootCmd := makeRootCmd(deps)
	errBuf := new(bytes.Buffer)
	rootCmd.SetOut(new(bytes.Buffer))
	rootCmd.SetErr(errBuf)
	rootCmd.SetArgs([]string{"audio-check", "-v"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "audio backend: pulseaudio"
	if !strings.Contains(errBuf.String(), want) {
		t.Errorf("expected stderr to contain %q, got: %q", want, errBuf.String())
	}
}

func TestAudioCheckCmd_Failure_NoVerbose_SilentOutput(t *testing.T) {
	deps := &Deps{AudioCheck: newAudioCheckDepsFailure("no output device available")}
	rootCmd := makeRootCmd(deps)
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rootCmd.SetOut(outBuf)
	rootCmd.SetErr(errBuf)
	rootCmd.SetArgs([]string{"audio-check"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when audio check fails, got nil")
	}
	if got := outBuf.String(); got != "" {
		t.Errorf("expected empty stdout on failure, got: %q", got)
	}
	if got := errBuf.String(); got != "" {
		t.Errorf("expected empty stderr on failure (non-verbose), got: %q", got)
	}
}

func TestAudioCheckCmd_Failure_Verbose_StderrHasError(t *testing.T) {
	reason := "no output device available"
	deps := &Deps{AudioCheck: newAudioCheckDepsFailure(reason)}
	rootCmd := makeRootCmd(deps)
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rootCmd.SetOut(outBuf)
	rootCmd.SetErr(errBuf)
	rootCmd.SetArgs([]string{"audio-check", "--verbose"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when audio check fails, got nil")
	}
	if got := outBuf.String(); got != "" {
		t.Errorf("expected empty stdout on failure, got: %q", got)
	}
	if !strings.HasPrefix(strings.TrimSpace(errBuf.String()), "Error:") {
		t.Errorf("expected stderr to start with 'Error:', got: %q", errBuf.String())
	}
	if !strings.Contains(errBuf.String(), reason) {
		t.Errorf("expected stderr to contain reason %q, got: %q", reason, errBuf.String())
	}
}

func TestAudioCheckCmd_Failure_NotErrUsage(t *testing.T) {
	deps := &Deps{AudioCheck: newAudioCheckDepsFailure("device busy")}
	rootCmd := makeRootCmd(deps)
	rootCmd.SetOut(new(bytes.Buffer))
	rootCmd.SetErr(new(bytes.Buffer))
	rootCmd.SetArgs([]string{"audio-check"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errors.Is(err, ErrUsage) {
		t.Errorf("audio check failure should not be ErrUsage (runtime failure, not argument error)")
	}
}

func TestAudioCheckCmd_UnknownFlag(t *testing.T) {
	deps := &Deps{AudioCheck: newAudioCheckDepsSuccess("coreaudio")}
	rootCmd := makeRootCmd(deps)
	rootCmd.SetOut(new(bytes.Buffer))
	rootCmd.SetErr(new(bytes.Buffer))
	rootCmd.SetArgs([]string{"audio-check", "--bogus"})

	if err := rootCmd.Execute(); err == nil {
		t.Fatal("expected error for unknown flag, got nil")
	}
}

func TestAudioCheckCmd_RejectsPositionalArgs(t *testing.T) {
	deps := &Deps{AudioCheck: newAudioCheckDepsSuccess("coreaudio")}
	rootCmd := makeRootCmd(deps)
	rootCmd.SetOut(new(bytes.Buffer))
	rootCmd.SetErr(new(bytes.Buffer))
	rootCmd.SetArgs([]string{"audio-check", "unexpected"})

	if err := rootCmd.Execute(); err == nil {
		t.Fatal("expected error when positional args provided, got nil")
	}
}

func TestAudioCheckCmd_Help(t *testing.T) {
	deps := &Deps{AudioCheck: newAudioCheckDepsSuccess("coreaudio")}
	rootCmd := makeRootCmd(deps)
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"audio-check", "--help"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error for --help: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"audio-check", "--verbose"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected help output to contain %q, got: %s", want, out)
		}
	}
}

func TestAudioCheckCmd_MissingDeps_ReturnsError(t *testing.T) {
	// Deps.AudioCheck が nil のケース
	rootCmd := makeRootCmd(&Deps{})
	rootCmd.SetOut(new(bytes.Buffer))
	rootCmd.SetErr(new(bytes.Buffer))
	rootCmd.SetArgs([]string{"audio-check"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when AudioCheck deps are missing, got nil")
	}
}
