package cmd

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"

	"github.com/canpok1/vox-actor/internal/app"
	"github.com/canpok1/vox-actor/internal/domain/entity"
	"github.com/spf13/cobra"
)

// say コマンド テストリスト
// DONE: sayサブコマンドがrootに登録されている
// DONE: 引数なしでErrUsageを返す
// DONE: ヘルプ出力に全フラグが含まれる（--engine-url, --speaker, --speed, --pitch, --intonation, --verbose, --output）
// DONE: ヘルプ出力に--watchフラグが含まれない
// DONE: デフォルトオプション値の確認（engine-url, speaker, speed, pitch, intonation）
// DONE: 環境変数VOX_ENGINE_URLがデフォルト値に反映される
// DONE: 環境変数VOX_SPEAKERがデフォルト値に反映される
// DONE: VOX_SPEAKERに不正な値が設定されている場合デフォルト値が使われる
// DONE: --verboseフラグのデフォルト値がfalse
// DONE: --output と --dry-run の併用は ErrUsage
// DONE: --output 指定時、Writer.Write が呼ばれる
// DONE: --output 指定時、ClientFactory/Player は呼ばれない

// findSayCmd はrootCmdからsayサブコマンドを検索して返すテストヘルパー。
func findSayCmd(t *testing.T, rootCmd *cobra.Command) *cobra.Command {
	t.Helper()
	for _, c := range rootCmd.Commands() {
		if c.Name() == "say" {
			return c
		}
	}
	t.Fatal("say subcommand not found")
	return nil
}

func TestSayCmd_RegisteredAsSubcommand(t *testing.T) {
	rootCmd := makeRootCmd()

	found := false
	for _, c := range rootCmd.Commands() {
		if c.Name() == "say" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'say' subcommand to be registered")
	}
}

func TestSayCmd_NoArgs_ReturnsUsageError(t *testing.T) {
	rootCmd := makeRootCmd()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"say"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when no args provided, got nil")
	}
	if !errors.Is(err, ErrUsage) {
		t.Errorf("expected ErrUsage, got: %v", err)
	}
}

func TestSayCmd_HelpContainsFlags(t *testing.T) {
	rootCmd := makeRootCmd()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"say", "--help"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error for --help, got: %v", err)
	}

	output := buf.String()
	flags := []string{"--engine-url", "--speaker", "--speed", "--pitch", "--intonation", "--verbose", "--dry-run", "--output"}
	for _, flag := range flags {
		if !strings.Contains(output, flag) {
			t.Errorf("expected help output to contain '%s'", flag)
		}
	}
}

func TestSayCmd_HelpDoesNotContainWatchFlag(t *testing.T) {
	rootCmd := makeRootCmd()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"say", "--help"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error for --help, got: %v", err)
	}

	output := buf.String()
	if strings.Contains(output, "--watch") {
		t.Error("expected help output NOT to contain '--watch'")
	}
}

func TestSayCmd_DefaultOptionValues(t *testing.T) {
	rootCmd := makeRootCmd()
	sayCmd := findSayCmd(t, rootCmd)

	engineURL, _ := sayCmd.Flags().GetString("engine-url")
	if engineURL != "http://localhost:50021" {
		t.Errorf("expected default engine-url 'http://localhost:50021', got: %s", engineURL)
	}

	speaker, _ := sayCmd.Flags().GetInt("speaker")
	if speaker != 3 {
		t.Errorf("expected default speaker 3, got: %d", speaker)
	}

	speed, _ := sayCmd.Flags().GetFloat64("speed")
	if speed != 1.0 {
		t.Errorf("expected default speed 1.0, got: %f", speed)
	}

	pitch, _ := sayCmd.Flags().GetFloat64("pitch")
	if pitch != 0.0 {
		t.Errorf("expected default pitch 0.0, got: %f", pitch)
	}

	intonation, _ := sayCmd.Flags().GetFloat64("intonation")
	if intonation != 1.0 {
		t.Errorf("expected default intonation 1.0, got: %f", intonation)
	}
}

func TestSayCmd_EnvVarVOXEngineURL(t *testing.T) {
	t.Setenv("VOX_ENGINE_URL", "http://custom:9999")

	rootCmd := makeRootCmd()
	sayCmd := findSayCmd(t, rootCmd)

	engineURL, _ := sayCmd.Flags().GetString("engine-url")
	if engineURL != "http://custom:9999" {
		t.Errorf("expected engine-url to be 'http://custom:9999', got: %s", engineURL)
	}
}

func TestSayCmd_EnvVarVOXSpeaker(t *testing.T) {
	t.Setenv("VOX_SPEAKER", "42")

	rootCmd := makeRootCmd()
	sayCmd := findSayCmd(t, rootCmd)

	speaker, _ := sayCmd.Flags().GetInt("speaker")
	if speaker != 42 {
		t.Errorf("expected speaker to be 42, got: %d", speaker)
	}
}

func TestSayCmd_EnvVarVOXSpeaker_InvalidValue(t *testing.T) {
	t.Setenv("VOX_SPEAKER", "not-a-number")

	rootCmd := makeRootCmd()
	sayCmd := findSayCmd(t, rootCmd)

	speaker, _ := sayCmd.Flags().GetInt("speaker")
	if speaker != 3 {
		t.Errorf("expected default speaker 3 when env var is invalid, got: %d", speaker)
	}
}

func TestSayCmd_VerboseFlag_DefaultFalse(t *testing.T) {
	rootCmd := makeRootCmd()
	sayCmd := findSayCmd(t, rootCmd)

	verbose, err := sayCmd.Flags().GetBool("verbose")
	if err != nil {
		t.Fatalf("expected --verbose flag to exist, got error: %v", err)
	}
	if verbose {
		t.Error("expected --verbose default to be false")
	}
}

// --- --output (-o) フラグ テスト ---

type fakeScriptWriter struct {
	calls []fakeWriterCall
}

type fakeWriterCall struct {
	path string
	text string
}

func (w *fakeScriptWriter) Write(path string, script entity.Script) (string, error) {
	w.calls = append(w.calls, fakeWriterCall{path: path, text: script.Text})
	return path, nil
}

func TestSayCmd_OutputFlag_HasShortAndLong(t *testing.T) {
	rootCmd := makeRootCmd()
	sayCmd := findSayCmd(t, rootCmd)

	if f := sayCmd.Flags().Lookup("output"); f == nil {
		t.Fatal("expected --output flag to be registered")
	} else if f.Shorthand != "o" {
		t.Errorf("expected --output shorthand to be 'o', got: %q", f.Shorthand)
	}
}

func TestSayCmd_OutputAndDryRun_ReturnsUsageError(t *testing.T) {
	writer := &fakeScriptWriter{}
	deps := &Deps{
		Say: &SayDeps{
			ClientFactory: func(_ string) app.VoicevoxClient { return nil },
			Player:        nil,
			WriterFactory: func(_ *slog.Logger) app.ScriptWriter { return writer },
		},
	}
	rootCmd := makeRootCmd(deps)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"say", "--output", "/tmp/out.txt", "--dry-run", "hello"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for --output + --dry-run, got nil")
	}
	if !errors.Is(err, ErrUsage) {
		t.Errorf("expected ErrUsage, got: %v", err)
	}
}

func TestSayCmd_OutputFlag_CallsWriter_NotClient(t *testing.T) {
	writer := &fakeScriptWriter{}
	clientFactoryCalled := false

	deps := &Deps{
		Say: &SayDeps{
			ClientFactory: func(_ string) app.VoicevoxClient {
				clientFactoryCalled = true
				return &noopClient{}
			},
			Player:        &noopPlayer{},
			WriterFactory: func(_ *slog.Logger) app.ScriptWriter { return writer },
		},
	}
	rootCmd := makeRootCmd(deps)

	tmpDir := t.TempDir()
	dest := filepath.Join(tmpDir, "out.txt")
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"say", "--output", dest, "hello"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(writer.calls) != 1 {
		t.Fatalf("expected 1 Writer.Write call, got: %d", len(writer.calls))
	}
	if writer.calls[0].path != dest {
		t.Errorf("expected path=%q, got %q", dest, writer.calls[0].path)
	}
	if writer.calls[0].text != "hello" {
		t.Errorf("expected text=hello, got %q", writer.calls[0].text)
	}
	if clientFactoryCalled {
		t.Error("expected ClientFactory NOT to be called when --output is set")
	}
}

// noopClient / noopPlayer は --output 経路では呼ばれないことを保証するためのスタブ。
type noopClient struct{}

func (n *noopClient) HealthCheck(_ context.Context) error { return errors.New("must not be called") }
func (n *noopClient) CreateQuery(_ context.Context, _ string, _ int) (*entity.AudioQuery, error) {
	return nil, errors.New("must not be called")
}
func (n *noopClient) Synthesize(_ context.Context, _ *entity.AudioQuery, _ int) ([]byte, error) {
	return nil, errors.New("must not be called")
}
func (n *noopClient) GetSpeakers(_ context.Context) ([]entity.Speaker, error) {
	return nil, errors.New("must not be called")
}

type noopPlayer struct{}

func (n *noopPlayer) Play(_ context.Context, _ []byte, _ app.PlayMeta) error {
	return errors.New("must not be called")
}
