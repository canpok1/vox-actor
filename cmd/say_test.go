package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
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
// #481 --viewer-url:
// DONE: --viewer-url フラグがヘルプに含まれる                                   → TestSayCmd_HelpContainsViewerURL
// DONE: --viewer-url デフォルトは空文字                                          → TestSayCmd_ViewerURL_DefaultEmpty
// DONE: VOX_VIEWER_URL 環境変数でデフォルト値が設定される                       → TestSayCmd_EnvVarVOXViewerURL
// DONE: --viewer-url 指定時は明示 URL の viewer に POST する                     → TestRunSay_ExplicitViewerURL_POSTsToViewer
// DONE: --viewer-url 指定時は lockfile auto-detect をスキップする                → TestRunSay_ExplicitViewerURL_SkipsLockfile
// DONE: --viewer-url 指定時は AudioProbe をスキップする                          → TestRunSay_ExplicitViewerURL_SkipsAudioProbe
// DONE: --viewer-url 指定時に接続失敗 → フォールバックせずエラーを返す          → TestRunSay_ExplicitViewerURL_ConnectionError_ReturnsError

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
	// clips 形式: {"clips":[{"text":"...","speaker_id":...}]}
	if clips, ok := capturedReq["clips"].([]interface{}); !ok || len(clips) == 0 {
		t.Errorf("expected clips array in request, got %v", capturedReq)
	} else if clip, ok := clips[0].(map[string]interface{}); !ok {
		t.Errorf("expected clips[0] to be an object, got %T", clips[0])
	} else if clip["text"] != "こんにちは" {
		t.Errorf("expected text=%q, got %v", "こんにちは", clip["text"])
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

// --- AudioProbe テスト (#468) ---
// DONE: say でビューワー未起動・非dryRunの場合 AudioProbe が呼ばれる     → TestRunSay_AudioProbe_CalledOnLocalPlayerPath
// DONE: say で dry-run の場合 AudioProbe がスキップされる               → TestRunSay_AudioProbe_SkippedOnDryRun
// DONE: say でビューワー起動中は AudioProbe がスキップされる            → TestRunSay_AudioProbe_SkippedWhenViewerRunning
// DONE: say で AudioProbe が失敗した場合エラーを返す                    → TestRunSay_AudioProbe_FailureCausesError

func TestRunSay_AudioProbe_CalledOnLocalPlayerPath(t *testing.T) {
	probeCalled := false

	dir := t.TempDir()
	lockPath := filepath.Join(dir, "viewer.lock")

	cmd := newCmdWithContext(t)
	registerCommonFlags(cmd)
	_ = cmd.ParseFlags([]string{})

	deps := &SayDeps{
		ClientFactory:    func(_ string) app.VoicevoxClient { return &mockSayClient{} },
		Player:           &trackingPlayer{},
		LockPathResolver: func() (string, error) { return lockPath, nil },
		AudioProbe:       func() error { probeCalled = true; return nil },
	}

	if err := runSay(cmd, []string{"テスト"}, deps); err != nil {
		t.Fatalf("runSay: %v", err)
	}
	if !probeCalled {
		t.Error("expected AudioProbe to be called on local player path, but it was not")
	}
}

func TestRunSay_AudioProbe_SkippedOnDryRun(t *testing.T) {
	probeCalled := false

	cmd := newCmdWithContext(t)
	registerCommonFlags(cmd)
	_ = cmd.ParseFlags([]string{"--dry-run"})

	deps := &SayDeps{
		ClientFactory: func(_ string) app.VoicevoxClient { return &mockSayClient{} },
		Player:        &trackingPlayer{},
		AudioProbe:    func() error { probeCalled = true; return nil },
	}

	if err := runSay(cmd, []string{"テスト"}, deps); err != nil {
		t.Fatalf("runSay: %v", err)
	}
	if probeCalled {
		t.Error("expected AudioProbe to be skipped in dry-run, but it was called")
	}
}

func TestRunSay_AudioProbe_SkippedWhenViewerRunning(t *testing.T) {
	probeCalled := false

	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("/api/play", func(w http.ResponseWriter, r *http.Request) {
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
		AudioProbe:       func() error { probeCalled = true; return nil },
	}

	if err := runSay(cmd, []string{"テスト"}, deps); err != nil {
		t.Fatalf("runSay: %v", err)
	}
	if probeCalled {
		t.Error("expected AudioProbe to be skipped when viewer is running, but it was called")
	}
}

func TestRunSay_AudioProbe_FailureCausesError(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, "viewer.lock")

	cmd := newCmdWithContext(t)
	registerCommonFlags(cmd)
	_ = cmd.ParseFlags([]string{})

	deps := &SayDeps{
		ClientFactory:    func(_ string) app.VoicevoxClient { return &mockSayClient{} },
		Player:           &trackingPlayer{},
		LockPathResolver: func() (string, error) { return lockPath, nil },
		AudioProbe:       func() error { return errors.New("no audio device") },
	}

	err := runSay(cmd, []string{"テスト"}, deps)
	if err == nil {
		t.Fatal("expected error when AudioProbe fails, got nil")
	}
	if !strings.Contains(err.Error(), "audio") {
		t.Errorf("expected error message to mention 'audio', got: %v", err)
	}
}

// --- #481 --viewer-url テスト ---

func TestSayCmd_HelpContainsViewerURL(t *testing.T) {
	rootCmd := makeRootCmd()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"say", "--help"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("expected no error for --help, got: %v", err)
	}
	if !strings.Contains(buf.String(), "--viewer-url") {
		t.Error("expected help output to contain '--viewer-url'")
	}
}

func TestSayCmd_ViewerURL_DefaultEmpty(t *testing.T) {
	t.Setenv("VOX_VIEWER_URL", "")

	rootCmd := makeRootCmd()
	sayCmd := findSayCmd(t, rootCmd)

	viewerURL, err := sayCmd.Flags().GetString("viewer-url")
	if err != nil {
		t.Fatalf("expected viewer-url flag to exist, got error: %v", err)
	}
	if viewerURL != "" {
		t.Errorf("expected default viewer-url to be empty, got: %s", viewerURL)
	}
}

func TestSayCmd_EnvVarVOXViewerURL(t *testing.T) {
	t.Setenv("VOX_VIEWER_URL", "http://remote:8080")

	rootCmd := makeRootCmd()
	sayCmd := findSayCmd(t, rootCmd)

	viewerURL, err := sayCmd.Flags().GetString("viewer-url")
	if err != nil {
		t.Fatalf("expected viewer-url flag to exist, got error: %v", err)
	}
	if viewerURL != "http://remote:8080" {
		t.Errorf("expected viewer-url 'http://remote:8080', got: %s", viewerURL)
	}
}

func TestRunSay_ExplicitViewerURL_POSTsToViewer(t *testing.T) {
	var postCount int
	var capturedText string
	mux := http.NewServeMux()
	mux.HandleFunc("/api/play", func(w http.ResponseWriter, r *http.Request) {
		postCount++
		var req map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&req)
		// clips 形式: {"clips":[{"text":"...","speaker_id":...}]}
		if clips, ok := req["clips"].([]interface{}); ok && len(clips) > 0 {
			if clip, ok := clips[0].(map[string]interface{}); ok {
				if v, ok := clip["text"].(string); ok {
					capturedText = v
				}
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"silent": false})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	cmd := newCmdWithContext(t)
	registerCommonFlags(cmd)
	_ = cmd.ParseFlags([]string{"--viewer-url", srv.URL})

	deps := &SayDeps{
		ClientFactory: func(_ string) app.VoicevoxClient { return &noopClient{} },
		Player:        &noopPlayer{},
	}

	if err := runSay(cmd, []string{"リモートテスト"}, deps); err != nil {
		t.Fatalf("runSay: %v", err)
	}
	if postCount != 1 {
		t.Errorf("expected 1 POST to /api/play, got %d", postCount)
	}
	if capturedText != "リモートテスト" {
		t.Errorf("expected text=%q, got %q", "リモートテスト", capturedText)
	}
}

func TestRunSay_ExplicitViewerURL_SkipsLockfile(t *testing.T) {
	// lockfile には起動中の viewer が存在するが、--viewer-url 優先のため lockfile は使われない
	var lockfilePostCount int
	lockfileMux := http.NewServeMux()
	lockfileMux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	lockfileMux.HandleFunc("/api/play", func(w http.ResponseWriter, r *http.Request) {
		lockfilePostCount++
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"silent": false})
	})
	lockPath, _ := setupFakeViewer(t, lockfileMux)

	// explicit URL 側のサーバー
	var explicitPostCount int
	explicitMux := http.NewServeMux()
	explicitMux.HandleFunc("/api/play", func(w http.ResponseWriter, r *http.Request) {
		explicitPostCount++
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"silent": false})
	})
	explicitSrv := httptest.NewServer(explicitMux)
	defer explicitSrv.Close()

	cmd := newCmdWithContext(t)
	registerCommonFlags(cmd)
	_ = cmd.ParseFlags([]string{"--viewer-url", explicitSrv.URL})

	deps := &SayDeps{
		ClientFactory:    func(_ string) app.VoicevoxClient { return &noopClient{} },
		Player:           &noopPlayer{},
		LockPathResolver: func() (string, error) { return lockPath, nil },
	}

	if err := runSay(cmd, []string{"テスト"}, deps); err != nil {
		t.Fatalf("runSay: %v", err)
	}
	if lockfilePostCount != 0 {
		t.Errorf("expected 0 POST to lockfile viewer, got %d", lockfilePostCount)
	}
	if explicitPostCount != 1 {
		t.Errorf("expected 1 POST to explicit viewer, got %d", explicitPostCount)
	}
}

func TestRunSay_ExplicitViewerURL_SkipsAudioProbe(t *testing.T) {
	probeCalled := false

	mux := http.NewServeMux()
	mux.HandleFunc("/api/play", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"silent": false})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	cmd := newCmdWithContext(t)
	registerCommonFlags(cmd)
	_ = cmd.ParseFlags([]string{"--viewer-url", srv.URL})

	deps := &SayDeps{
		ClientFactory: func(_ string) app.VoicevoxClient { return &noopClient{} },
		Player:        &noopPlayer{},
		AudioProbe:    func() error { probeCalled = true; return nil },
	}

	if err := runSay(cmd, []string{"テスト"}, deps); err != nil {
		t.Fatalf("runSay: %v", err)
	}
	if probeCalled {
		t.Error("expected AudioProbe to be skipped when --viewer-url is set, but it was called")
	}
}

func TestRunSay_ExplicitViewerURL_ConnectionError_ReturnsError(t *testing.T) {
	// Not-listening server to simulate connection failure
	playCalled := false

	cmd := newCmdWithContext(t)
	registerCommonFlags(cmd)
	_ = cmd.ParseFlags([]string{"--viewer-url", "http://127.0.0.1:1"})

	deps := &SayDeps{
		ClientFactory: func(_ string) app.VoicevoxClient { return &noopClient{} },
		Player:        &trackingPlayer{playFn: func() { playCalled = true }},
	}

	err := runSay(cmd, []string{"テスト"}, deps)
	if err == nil {
		t.Fatal("expected error when viewer is not reachable, got nil")
	}
	if playCalled {
		t.Error("expected no local playback when --viewer-url fails")
	}
}

// --- #489 say が viewer モードで playback_id を stdout 出力 ---

func TestRunSay_ViewerMode_PrintsPlaybackIDToStdout(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("/api/play", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"playback_id": "22222222-2222-4222-8222-222222222222",
			"clip_count":  1,
			"silent":      false,
		})
	})
	lockPath, _ := setupFakeViewer(t, mux)

	cmd := newCmdWithContext(t)
	registerCommonFlags(cmd)
	_ = cmd.ParseFlags([]string{})

	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	deps := &SayDeps{
		ClientFactory:    func(_ string) app.VoicevoxClient { return &noopClient{} },
		Player:           &noopPlayer{},
		LockPathResolver: func() (string, error) { return lockPath, nil },
	}

	if err := runSay(cmd, []string{"こんにちは"}, deps); err != nil {
		t.Fatalf("runSay: %v", err)
	}
	got := strings.TrimSpace(stdout.String())
	if got != "22222222-2222-4222-8222-222222222222" {
		t.Errorf("expected stdout=playback_id, got %q", got)
	}
}

func TestRunSay_LocalMode_PrintsLocalPlaybackToStdout(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	lockPath := filepath.Join(dir, "viewer.lock")

	cmd := newCmdWithContext(t)
	registerCommonFlags(cmd)
	_ = cmd.ParseFlags([]string{})

	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	deps := &SayDeps{
		ClientFactory:    func(_ string) app.VoicevoxClient { return &mockSayClient{} },
		Player:           &trackingPlayer{playFn: func() {}},
		LockPathResolver: func() (string, error) { return lockPath, nil },
	}

	if err := runSay(cmd, []string{"こんにちは"}, deps); err != nil {
		t.Fatalf("runSay: %v", err)
	}
	got := strings.TrimSpace(stdout.String())
	if got != "local_playback" {
		t.Errorf("expected stdout=local_playback, got %q", got)
	}
}

// --- #534 --save-wav テスト ---
// DONE: --save-wav フラグがヘルプに含まれる             → TestSayCmd_HelpContainsSaveWav
// DONE: --save-wav デフォルトは空文字                   → TestSayCmd_SaveWavFlag_DefaultEmpty
// DONE: VOX_SAVE_WAV 環境変数でデフォルト値が設定される → TestSayCmd_EnvVarVOXSaveWav
// DONE: --save-wav 指定時にファイルが保存される          → TestRunSay_SaveWav_SavesFile

func TestSayCmd_HelpContainsSaveWav(t *testing.T) {
	rootCmd := makeRootCmd()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"say", "--help"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("expected no error for --help, got: %v", err)
	}
	if !strings.Contains(buf.String(), "--save-wav") {
		t.Error("expected help output to contain '--save-wav'")
	}
}

func TestSayCmd_SaveWavFlag_DefaultEmpty(t *testing.T) {
	t.Setenv("VOX_SAVE_WAV", "")

	rootCmd := makeRootCmd()
	sayCmd := findSayCmd(t, rootCmd)

	saveWav, err := sayCmd.Flags().GetString("save-wav")
	if err != nil {
		t.Fatalf("expected save-wav flag to exist, got error: %v", err)
	}
	if saveWav != "" {
		t.Errorf("expected default save-wav to be empty, got: %s", saveWav)
	}
}

func TestSayCmd_EnvVarVOXSaveWav(t *testing.T) {
	t.Setenv("VOX_SAVE_WAV", "/tmp/output.wav")

	rootCmd := makeRootCmd()
	sayCmd := findSayCmd(t, rootCmd)

	saveWav, err := sayCmd.Flags().GetString("save-wav")
	if err != nil {
		t.Fatalf("expected save-wav flag to exist, got error: %v", err)
	}
	if saveWav != "/tmp/output.wav" {
		t.Errorf("expected save-wav '/tmp/output.wav', got: %s", saveWav)
	}
}

func TestRunSay_SaveWav_SavesFile(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, "viewer.lock")
	wavPath := filepath.Join(dir, "output.wav")

	cmd := newCmdWithContext(t)
	registerCommonFlags(cmd)
	registerSaveWavFlag(cmd)
	_ = cmd.ParseFlags([]string{"--save-wav", wavPath})

	deps := &SayDeps{
		ClientFactory:    func(_ string) app.VoicevoxClient { return &mockSayClient{} },
		Player:           &trackingPlayer{playFn: func() {}},
		LockPathResolver: func() (string, error) { return lockPath, nil },
	}

	if err := runSay(cmd, []string{"こんにちは"}, deps); err != nil {
		t.Fatalf("runSay: %v", err)
	}

	if _, err := os.Stat(wavPath); os.IsNotExist(err) {
		t.Errorf("expected wav file to be saved at %s", wavPath)
	}
}
