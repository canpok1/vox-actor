package cmd

// viewer-check コマンド テストリスト
// DONE: viewer-checkサブコマンドがrootに登録されている
// DONE: 正常系 / non-verbose: viewer起動中なら終了コード0、stdout/stderrともに空
// DONE: 正常系 / verbose: stdoutに "viewer addr: <addr>" が出力される
// DONE: 正常系 / verbose short flag (-v)
// DONE: 異常系 / non-verbose: viewer未起動なら非0終了、stdout/stderrともに空
// DONE: 異常系 / verbose: viewer未起動でも stdout/stderr は空
// DONE: 異常系はErrUsageではない（ランタイム失敗のため）
// DONE: 未知のフラグはエラーを返す
// DONE: 引数を受け付けない（位置引数がある場合エラー）
// DONE: --help でコマンド説明が出力される
// DONE: 依存が未設定だとエラーを返す（セーフティネット）
// DONE: LockPathResolverがエラーを返す場合は非0終了

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func newViewerCheckDepsRunning(addr string) *ViewerCheckDeps {
	return &ViewerCheckDeps{
		LockPathResolver: func() (string, error) { return "/tmp/fake.lock", nil },
		DetectFunc:       func(_ string) (string, bool) { return addr, true },
	}
}

func newViewerCheckDepsNotRunning() *ViewerCheckDeps {
	return &ViewerCheckDeps{
		LockPathResolver: func() (string, error) { return "/tmp/fake.lock", nil },
		DetectFunc:       func(_ string) (string, bool) { return "", false },
	}
}

func TestViewerCheckCmd_RegisteredAsSubcommand(t *testing.T) {
	rootCmd := makeRootCmd()

	found := false
	for _, c := range rootCmd.Commands() {
		if c.Name() == "viewer-check" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'viewer-check' subcommand to be registered")
	}
}

func TestViewerCheckCmd_ViewerRunning_NoVerbose_ExitZeroSilentOutput(t *testing.T) {
	deps := &Deps{ViewerCheck: newViewerCheckDepsRunning("127.0.0.1:8080")}
	rootCmd := makeRootCmd(deps)
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rootCmd.SetOut(outBuf)
	rootCmd.SetErr(errBuf)
	rootCmd.SetArgs([]string{"viewer-check"})

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

func TestViewerCheckCmd_ViewerRunning_Verbose_StdoutHasAddr(t *testing.T) {
	addr := "127.0.0.1:8080"
	deps := &Deps{ViewerCheck: newViewerCheckDepsRunning(addr)}
	rootCmd := makeRootCmd(deps)
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rootCmd.SetOut(outBuf)
	rootCmd.SetErr(errBuf)
	rootCmd.SetArgs([]string{"viewer-check", "--verbose"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v (stderr=%q)", err, errBuf.String())
	}
	if got := errBuf.String(); got != "" {
		t.Errorf("expected empty stderr, got: %q", got)
	}
	want := "viewer addr: " + addr
	if !strings.Contains(outBuf.String(), want) {
		t.Errorf("expected stdout to contain %q, got: %q", want, outBuf.String())
	}
}

func TestViewerCheckCmd_ViewerRunning_VerboseShortFlag(t *testing.T) {
	addr := "127.0.0.1:9999"
	deps := &Deps{ViewerCheck: newViewerCheckDepsRunning(addr)}
	rootCmd := makeRootCmd(deps)
	outBuf := new(bytes.Buffer)
	rootCmd.SetOut(outBuf)
	rootCmd.SetErr(new(bytes.Buffer))
	rootCmd.SetArgs([]string{"viewer-check", "-v"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "viewer addr: " + addr
	if !strings.Contains(outBuf.String(), want) {
		t.Errorf("expected stdout to contain %q, got: %q", want, outBuf.String())
	}
}

func TestViewerCheckCmd_ViewerNotRunning_NoVerbose_NonZeroExitSilentOutput(t *testing.T) {
	deps := &Deps{ViewerCheck: newViewerCheckDepsNotRunning()}
	rootCmd := makeRootCmd(deps)
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rootCmd.SetOut(outBuf)
	rootCmd.SetErr(errBuf)
	rootCmd.SetArgs([]string{"viewer-check"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when viewer is not running, got nil")
	}
	if got := outBuf.String(); got != "" {
		t.Errorf("expected empty stdout on failure, got: %q", got)
	}
	if got := errBuf.String(); got != "" {
		t.Errorf("expected empty stderr on failure (non-verbose), got: %q", got)
	}
}

func TestViewerCheckCmd_ViewerNotRunning_Verbose_SilentOutput(t *testing.T) {
	deps := &Deps{ViewerCheck: newViewerCheckDepsNotRunning()}
	rootCmd := makeRootCmd(deps)
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rootCmd.SetOut(outBuf)
	rootCmd.SetErr(errBuf)
	rootCmd.SetArgs([]string{"viewer-check", "--verbose"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when viewer is not running, got nil")
	}
	if got := outBuf.String(); got != "" {
		t.Errorf("expected empty stdout on failure (verbose), got: %q", got)
	}
	if got := errBuf.String(); got != "" {
		t.Errorf("expected empty stderr on failure (verbose), got: %q", got)
	}
}

func TestViewerCheckCmd_Failure_NotErrUsage(t *testing.T) {
	deps := &Deps{ViewerCheck: newViewerCheckDepsNotRunning()}
	rootCmd := makeRootCmd(deps)
	rootCmd.SetOut(new(bytes.Buffer))
	rootCmd.SetErr(new(bytes.Buffer))
	rootCmd.SetArgs([]string{"viewer-check"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errors.Is(err, ErrUsage) {
		t.Errorf("viewer check failure should not be ErrUsage (runtime failure, not argument error)")
	}
}

func TestViewerCheckCmd_UnknownFlag(t *testing.T) {
	deps := &Deps{ViewerCheck: newViewerCheckDepsRunning("127.0.0.1:8080")}
	rootCmd := makeRootCmd(deps)
	rootCmd.SetOut(new(bytes.Buffer))
	rootCmd.SetErr(new(bytes.Buffer))
	rootCmd.SetArgs([]string{"viewer-check", "--bogus"})

	if err := rootCmd.Execute(); err == nil {
		t.Fatal("expected error for unknown flag, got nil")
	}
}

func TestViewerCheckCmd_RejectsPositionalArgs(t *testing.T) {
	deps := &Deps{ViewerCheck: newViewerCheckDepsRunning("127.0.0.1:8080")}
	rootCmd := makeRootCmd(deps)
	rootCmd.SetOut(new(bytes.Buffer))
	rootCmd.SetErr(new(bytes.Buffer))
	rootCmd.SetArgs([]string{"viewer-check", "unexpected"})

	if err := rootCmd.Execute(); err == nil {
		t.Fatal("expected error when positional args provided, got nil")
	}
}

func TestViewerCheckCmd_Help(t *testing.T) {
	deps := &Deps{ViewerCheck: newViewerCheckDepsRunning("127.0.0.1:8080")}
	rootCmd := makeRootCmd(deps)
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"viewer-check", "--help"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error for --help: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"viewer-check", "--verbose"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected help output to contain %q, got: %s", want, out)
		}
	}
}

func TestViewerCheckCmd_MissingDeps_ReturnsError(t *testing.T) {
	rootCmd := makeRootCmd(&Deps{})
	rootCmd.SetOut(new(bytes.Buffer))
	rootCmd.SetErr(new(bytes.Buffer))
	rootCmd.SetArgs([]string{"viewer-check"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when ViewerCheck deps are missing, got nil")
	}
}

func TestViewerCheckCmd_LockPathResolveError_NonZeroExit(t *testing.T) {
	deps := &Deps{ViewerCheck: &ViewerCheckDeps{
		LockPathResolver: func() (string, error) { return "", errors.New("home not found") },
		DetectFunc:       func(_ string) (string, bool) { return "", false },
	}}
	rootCmd := makeRootCmd(deps)
	rootCmd.SetOut(new(bytes.Buffer))
	rootCmd.SetErr(new(bytes.Buffer))
	rootCmd.SetArgs([]string{"viewer-check"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when lock path resolution fails, got nil")
	}
}
