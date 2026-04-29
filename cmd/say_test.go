package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/canpok1/vox-actor/internal/app"
	"github.com/canpok1/vox-actor/internal/domain/entity"
	"github.com/spf13/cobra"
)

// say コマンド テストリスト
// DONE: sayサブコマンドがrootに登録されている
// #418 viewer 連携:
// DONE: viewer 起動中は /api/play に POST してローカル再生をしない → TestRunSay_ViewerRunning_POSTsToViewer
// DONE: viewer 無音モード応答で WARN ログを出す                   → TestRunSay_ViewerRunning_SilentMode_Warns
// DONE: viewer 未起動はローカル再生にフォールバック               → TestRunSay_ViewerNotRunning_UsesLocalPlayer
// DONE: --dry-run 時は viewer へ POST しない                      → TestRunSay_DryRun_NoViewerPost
// DONE: 引数なしでErrUsageを返す
// DONE: ヘルプ出力に全フラグが含まれる（--engine-url, --speaker, --speed, --pitch, --intonation, --verbose）
// DONE: ヘルプ出力に--output/--watchフラグが含まれない
// DONE: デフォルトオプション値の確認（engine-url, speaker, speed, pitch, intonation）
// DONE: 環境変数VOX_ENGINE_URLがデフォルト値に反映される
// DONE: 環境変数VOX_SPEAKERがデフォルト値に反映される
// DONE: VOX_SPEAKERに不正な値が設定されている場合デフォルト値が使われる
// DONE: --verboseフラグのデフォルト値がfalse

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
	flags := []string{"--engine-url", "--speaker", "--speed", "--pitch", "--intonation", "--verbose", "--dry-run"}
	for _, flag := range flags {
		if !strings.Contains(output, flag) {
			t.Errorf("expected help output to contain '%s'", flag)
		}
	}
}

func TestSayCmd_HelpDoesNotContainRemovedFlags(t *testing.T) {
	rootCmd := makeRootCmd()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"say", "--help"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error for --help, got: %v", err)
	}

	output := buf.String()
	for _, flag := range []string{"--watch", "--output"} {
		if strings.Contains(output, flag) {
			t.Errorf("expected help output NOT to contain '%s'", flag)
		}
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

// noopClient / noopPlayer は say コマンドのテスト用スタブ。
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

// --- viewer 連携テスト (#418) ---

// newCmdWithContext は context.Background() を設定した cobra.Command を返すテストヘルパー。
func newCmdWithContext(t *testing.T) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	return cmd
}

func TestRunSay_ViewerRunning_POSTsToViewer(t *testing.T) {
	var postCount int
	var capturedReq map[string]interface{}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/api/play", func(w http.ResponseWriter, r *http.Request) {
		postCount++
		_ = json.NewDecoder(r.Body).Decode(&capturedReq)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"silent": false})
	})
	lockPath, _ := setupFakeViewer(t, mux)

	cmd := newCmdWithContext(t)
	registerCommonFlags(cmd)
	_ = cmd.ParseFlags([]string{})

	deps := &SayDeps{
		ClientFactory:    func(_ string) app.VoicevoxClient { return &noopClient{} },
		Player:           &noopPlayer{},
		LockPathResolver: func() (string, error) { return lockPath, nil },
	}

	if err := runSay(cmd, []string{"こんにちは"}, deps); err != nil {
		t.Fatalf("runSay: %v", err)
	}
	if postCount != 1 {
		t.Errorf("expected 1 POST to /api/play, got %d", postCount)
	}
	if capturedReq["text"] != "こんにちは" {
		t.Errorf("expected text=%q, got %v", "こんにちは", capturedReq["text"])
	}
}

func TestRunSay_ViewerRunning_SilentMode_Warns(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/api/play", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"silent":        true,
			"silent_reason": "VOICEVOX接続失敗",
		})
	})
	lockPath, _ := setupFakeViewer(t, mux)

	cmd := newCmdWithContext(t)
	registerCommonFlags(cmd)
	_ = cmd.ParseFlags([]string{})

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	deps := &SayDeps{
		ClientFactory:    func(_ string) app.VoicevoxClient { return &noopClient{} },
		Player:           &noopPlayer{},
		LockPathResolver: func() (string, error) { return lockPath, nil },
		Logger:           logger,
	}

	if err := runSay(cmd, []string{"テスト"}, deps); err != nil {
		t.Fatalf("runSay: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "WARN") {
		t.Errorf("expected WARN log for silent mode, got: %s", output)
	}
}

func TestRunSay_ViewerNotRunning_UsesLocalPlayer(t *testing.T) {
	playCalled := false

	player := &trackingPlayer{playFn: func() { playCalled = true }}
	client := &mockSayClient{}

	dir := t.TempDir()
	lockPath := filepath.Join(dir, "viewer.lock")

	cmd := newCmdWithContext(t)
	registerCommonFlags(cmd)
	_ = cmd.ParseFlags([]string{})

	deps := &SayDeps{
		ClientFactory:    func(_ string) app.VoicevoxClient { return client },
		Player:           player,
		LockPathResolver: func() (string, error) { return lockPath, nil },
	}

	if err := runSay(cmd, []string{"ローカル再生テスト"}, deps); err != nil {
		t.Fatalf("runSay: %v", err)
	}
	if !playCalled {
		t.Error("expected local player to be called")
	}
}

func TestRunSay_DryRun_NoViewerPost(t *testing.T) {
	var postCount int
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/api/play", func(w http.ResponseWriter, r *http.Request) {
		postCount++
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"silent": false})
	})
	lockPath, _ := setupFakeViewer(t, mux)

	cmd := newCmdWithContext(t)
	registerCommonFlags(cmd)
	_ = cmd.ParseFlags([]string{"--dry-run"})

	deps := &SayDeps{
		ClientFactory:    func(_ string) app.VoicevoxClient { return &noopClient{} },
		Player:           &noopPlayer{},
		LockPathResolver: func() (string, error) { return lockPath, nil },
	}

	if err := runSay(cmd, []string{"ドライラン"}, deps); err != nil {
		t.Fatalf("runSay: %v", err)
	}
	if postCount != 0 {
		t.Errorf("expected no POST in dry-run, got %d", postCount)
	}
}

// trackingPlayer は Play が呼ばれたかを追跡するスタブ。
type trackingPlayer struct {
	playFn func()
}

func (p *trackingPlayer) Play(_ context.Context, _ []byte, _ app.PlayMeta) error {
	if p.playFn != nil {
		p.playFn()
	}
	return nil
}

// mockSayClient は say テスト用の最小限 VoicevoxClient スタブ。
type mockSayClient struct{}

func (c *mockSayClient) HealthCheck(_ context.Context) error { return nil }
func (c *mockSayClient) CreateQuery(_ context.Context, _ string, _ int) (*entity.AudioQuery, error) {
	return &entity.AudioQuery{}, nil
}
func (c *mockSayClient) Synthesize(_ context.Context, _ *entity.AudioQuery, _ int) ([]byte, error) {
	return []byte("RIFFx"), nil
}
func (c *mockSayClient) GetSpeakers(_ context.Context) ([]entity.Speaker, error) { return nil, nil }
