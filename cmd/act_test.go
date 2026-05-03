package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/canpok1/vox-actor/internal/app"
	"github.com/canpok1/vox-actor/internal/domain/entity"
	"github.com/spf13/cobra"
)

// act コマンド テストリスト（すべて実装済み）
// #418 viewer 連携:
// DONE: viewer 起動中は /api/play を行ごとに POST してローカル再生しない    → TestRunAct_ViewerRunning_POSTsPerLine
// DONE: viewer 未起動はローカル再生にフォールバック                         → TestRunAct_ViewerNotRunning_UsesLocalPlayer
// DONE: --dry-run 時は viewer へ POST しない                               → TestRunAct_DryRun_NoViewerPost
// DONE: 途中行で POST 失敗した場合即座にエラー終了する                      → TestRunAct_ViewerRunning_PostFails_StopsEarly

// グレースフルシャットダウン テストリスト
// DONE: actコマンドのcontextにシグナルハンドリングが設定されていることを確認

// 環境変数対応 テストリスト #59
// DONE: VOX_ENGINE_URL環境変数が設定されている場合、--engine-urlのデフォルト値として反映される
// DONE: VOX_SPEAKER環境変数が設定されている場合、--speakerのデフォルト値として反映される
// DONE: CLIフラグ --engine-url 指定時はVOX_ENGINE_URLより優先される
// DONE: CLIフラグ --speaker 指定時はVOX_SPEAKERより優先される
// DONE: 環境変数未設定時はデフォルト値が使われる（既存テストでカバー済み: TestActCmd_DefaultOptionValues）
// DONE: VOX_SPEAKER に不正な値（数値でない）が設定されている場合デフォルト値が使われる

// findActCmd はrootCmdからactサブコマンドを検索して返すテストヘルパー。
// 見つからない場合はテストをFatalで終了する。
func findActCmd(t *testing.T, rootCmd *cobra.Command) *cobra.Command {
	t.Helper()
	for _, c := range rootCmd.Commands() {
		if c.Name() == "act" {
			return c
		}
	}
	t.Fatal("act subcommand not found")
	return nil
}

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
	actCmd := findActCmd(t, rootCmd)

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
	flags := []string{"--engine-url", "--speaker", "--speed", "--pitch", "--intonation", "--verbose", "--dry-run"}
	for _, flag := range flags {
		if !strings.Contains(output, flag) {
			t.Errorf("expected help output to contain '%s'", flag)
		}
	}
}

func TestActCmd_VerboseFlag_DefaultFalse(t *testing.T) {
	rootCmd := makeRootCmd()
	actCmd := findActCmd(t, rootCmd)

	verbose, err := actCmd.Flags().GetBool("verbose")
	if err != nil {
		t.Fatalf("expected --verbose flag to exist, got error: %v", err)
	}
	if verbose {
		t.Error("expected --verbose default to be false")
	}
}

func TestActCmd_EnvVarVOXEngineURL(t *testing.T) {
	t.Setenv("VOX_ENGINE_URL", "http://custom:9999")

	rootCmd := makeRootCmd()
	actCmd := findActCmd(t, rootCmd)

	engineURL, _ := actCmd.Flags().GetString("engine-url")
	if engineURL != "http://custom:9999" {
		t.Errorf("expected engine-url to be 'http://custom:9999', got: %s", engineURL)
	}
}

func TestActCmd_EnvVarVOXSpeaker(t *testing.T) {
	t.Setenv("VOX_SPEAKER", "42")

	rootCmd := makeRootCmd()
	actCmd := findActCmd(t, rootCmd)

	speaker, _ := actCmd.Flags().GetInt("speaker")
	if speaker != 42 {
		t.Errorf("expected speaker to be 42, got: %d", speaker)
	}
}

func TestActCmd_CLIFlagOverridesEnvEngineURL(t *testing.T) {
	t.Setenv("VOX_ENGINE_URL", "http://env:1111")

	rootCmd := makeRootCmd()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"act", "--engine-url", "http://cli:2222", "dummy.txt"})

	// コマンド実行（depsがnilなのでエラーになるが、フラグパースは行われる）
	_ = rootCmd.Execute()

	actCmd := findActCmd(t, rootCmd)

	engineURL, _ := actCmd.Flags().GetString("engine-url")
	if engineURL != "http://cli:2222" {
		t.Errorf("expected CLI flag to override env var, got: %s", engineURL)
	}
}

func TestActCmd_CLIFlagOverridesEnvSpeaker(t *testing.T) {
	t.Setenv("VOX_SPEAKER", "99")

	rootCmd := makeRootCmd()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"act", "--speaker", "7", "dummy.txt"})

	_ = rootCmd.Execute()

	actCmd := findActCmd(t, rootCmd)

	speaker, _ := actCmd.Flags().GetInt("speaker")
	if speaker != 7 {
		t.Errorf("expected CLI flag to override env var, got: %d", speaker)
	}
}

func TestActCmd_EnvVarVOXSpeaker_InvalidValue(t *testing.T) {
	t.Setenv("VOX_SPEAKER", "not-a-number")

	rootCmd := makeRootCmd()
	actCmd := findActCmd(t, rootCmd)

	// 不正な値の場合はデフォルト値3が使われる
	speaker, _ := actCmd.Flags().GetInt("speaker")
	if speaker != 3 {
		t.Errorf("expected default speaker 3 when env var is invalid, got: %d", speaker)
	}
}

// --- viewer 連携テスト (#418) ---

// stubScriptReaderForAct はテキストファイルを渡さずに固定スクリプトを返すスタブ。
type stubScriptReaderForAct struct {
	scripts []entity.Script
}

func (s stubScriptReaderForAct) Read(_ string) ([]entity.Script, error) {
	return s.scripts, nil
}

func TestRunAct_ViewerRunning_POSTsPerLine(t *testing.T) {
	var postCount int
	var capturedTexts []string

	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("/api/play", func(w http.ResponseWriter, r *http.Request) {
		postCount++
		var req map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if text, ok := req["text"].(string); ok {
			capturedTexts = append(capturedTexts, text)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"silent": false})
	})
	lockPath, _ := setupFakeViewer(t, mux)

	scriptFile := filepath.Join(t.TempDir(), "test.txt")
	if err := os.WriteFile(scriptFile, []byte("line1\nline2\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cmd := newCmdWithContext(t)
	registerCommonFlags(cmd)
	_ = cmd.ParseFlags([]string{})

	deps := &ActDeps{
		Reader:           stubScriptReaderForAct{scripts: []entity.Script{{Text: "line1"}, {Text: "line2"}}},
		ClientFactory:    func(_ string) app.VoicevoxClient { return &noopClient{} },
		Player:           &noopPlayer{},
		LockPathResolver: func() (string, error) { return lockPath, nil },
	}

	if err := runAct(cmd, []string{scriptFile}, deps); err != nil {
		t.Fatalf("runAct: %v", err)
	}
	if postCount != 2 {
		t.Errorf("expected 2 POSTs, got %d", postCount)
	}
	if len(capturedTexts) != 2 || capturedTexts[0] != "line1" || capturedTexts[1] != "line2" {
		t.Errorf("expected texts [line1, line2], got %v", capturedTexts)
	}
}

func TestRunAct_ViewerNotRunning_UsesLocalPlayer(t *testing.T) {
	playCalled := 0
	player := &trackingPlayer{playFn: func() { playCalled++ }}
	client := &mockSayClient{}

	dir := t.TempDir()
	lockPath := filepath.Join(dir, "viewer.lock")

	scriptFile := filepath.Join(t.TempDir(), "test.txt")
	if err := os.WriteFile(scriptFile, []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cmd := newCmdWithContext(t)
	registerCommonFlags(cmd)
	_ = cmd.ParseFlags([]string{})

	deps := &ActDeps{
		Reader:           stubScriptReaderForAct{scripts: []entity.Script{{Text: "hello"}}},
		ClientFactory:    func(_ string) app.VoicevoxClient { return client },
		Player:           player,
		LockPathResolver: func() (string, error) { return lockPath, nil },
	}

	if err := runAct(cmd, []string{scriptFile}, deps); err != nil {
		t.Fatalf("runAct: %v", err)
	}
	if playCalled != 1 {
		t.Errorf("expected local player called once, got %d", playCalled)
	}
}

func TestRunAct_DryRun_NoViewerPost(t *testing.T) {
	var postCount int
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("/api/play", func(w http.ResponseWriter, r *http.Request) {
		postCount++
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"silent": false})
	})
	lockPath, _ := setupFakeViewer(t, mux)

	scriptFile := filepath.Join(t.TempDir(), "test.txt")
	if err := os.WriteFile(scriptFile, []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cmd := newCmdWithContext(t)
	registerCommonFlags(cmd)
	_ = cmd.ParseFlags([]string{"--dry-run"})

	deps := &ActDeps{
		Reader:           stubScriptReaderForAct{scripts: []entity.Script{{Text: "hello"}}},
		ClientFactory:    func(_ string) app.VoicevoxClient { return &noopClient{} },
		Player:           &noopPlayer{},
		LockPathResolver: func() (string, error) { return lockPath, nil },
	}

	if err := runAct(cmd, []string{scriptFile}, deps); err != nil {
		t.Fatalf("runAct: %v", err)
	}
	if postCount != 0 {
		t.Errorf("expected no POST in dry-run, got %d", postCount)
	}
}

func TestRunAct_ViewerRunning_PostFails_StopsEarly(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("/api/play", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	})
	lockPath, _ := setupFakeViewer(t, mux)

	scriptFile := filepath.Join(t.TempDir(), "test.txt")
	if err := os.WriteFile(scriptFile, []byte("line1\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cmd := newCmdWithContext(t)
	registerCommonFlags(cmd)
	_ = cmd.ParseFlags([]string{})

	deps := &ActDeps{
		Reader:           stubScriptReaderForAct{scripts: []entity.Script{{Text: "line1"}}},
		ClientFactory:    func(_ string) app.VoicevoxClient { return &noopClient{} },
		Player:           &noopPlayer{},
		LockPathResolver: func() (string, error) { return lockPath, nil },
	}

	err := runAct(cmd, []string{scriptFile}, deps)
	if err == nil {
		t.Fatal("expected error when POST fails, got nil")
	}
}

func TestRunAct_ViewerRunning_SilentMode_Warns(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("/api/play", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"silent":        true,
			"silent_reason": "VOICEVOX接続失敗",
		})
	})
	lockPath, _ := setupFakeViewer(t, mux)

	scriptFile := filepath.Join(t.TempDir(), "test.txt")
	if err := os.WriteFile(scriptFile, []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cmd := newCmdWithContext(t)
	registerCommonFlags(cmd)
	_ = cmd.ParseFlags([]string{})

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	deps := &ActDeps{
		Reader:           stubScriptReaderForAct{scripts: []entity.Script{{Text: "hello"}}},
		ClientFactory:    func(_ string) app.VoicevoxClient { return &noopClient{} },
		Player:           &noopPlayer{},
		LockPathResolver: func() (string, error) { return lockPath, nil },
		Logger:           logger,
	}

	if err := runAct(cmd, []string{scriptFile}, deps); err != nil {
		t.Fatalf("runAct: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "WARN") {
		t.Errorf("expected WARN log for silent mode, got: %s", output)
	}
}

func TestActCmd_WatchFlag_ReturnsMigrationError(t *testing.T) {
	dir := t.TempDir()

	rootCmd := makeRootCmd()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"act", "--watch", dir})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when --watch is used, got nil")
	}
	if !errors.Is(err, ErrUsage) {
		t.Errorf("expected ErrUsage, got: %v", err)
	}
	if !strings.Contains(err.Error(), "watch <dir>") {
		t.Errorf("expected migration guide in error, got: %v", err)
	}
}

func TestActCmd_WatchDeleteFlag_ReturnsMigrationError(t *testing.T) {
	dir := t.TempDir()

	rootCmd := makeRootCmd()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"act", "--watch-delete", dir})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when --watch-delete is used, got nil")
	}
	if !errors.Is(err, ErrUsage) {
		t.Errorf("expected ErrUsage, got: %v", err)
	}
	if !strings.Contains(err.Error(), "watch --delete") {
		t.Errorf("expected migration guide in error, got: %v", err)
	}
}

// mockActClient は act テスト用の VoicevoxClient スタブ（HealthCheck OK、合成成功）。
type mockActClient struct{}

func (c *mockActClient) HealthCheck(_ context.Context) error { return nil }
func (c *mockActClient) CreateQuery(_ context.Context, _ string, _ int) (*entity.AudioQuery, error) {
	return &entity.AudioQuery{}, nil
}
func (c *mockActClient) Synthesize(_ context.Context, _ *entity.AudioQuery, _ int) ([]byte, error) {
	return []byte("RIFFx"), nil
}
func (c *mockActClient) GetSpeakers(_ context.Context) ([]entity.Speaker, error) { return nil, nil }

// --- AudioProbe テスト (#468) ---
// DONE: act でビューワー未起動・非dryRunの場合 AudioProbe が呼ばれる    → TestRunAct_AudioProbe_CalledOnLocalPlayerPath
// DONE: act で dry-run の場合 AudioProbe がスキップされる              → TestRunAct_AudioProbe_SkippedOnDryRun
// DONE: act でビューワー起動中は AudioProbe がスキップされる           → TestRunAct_AudioProbe_SkippedWhenViewerRunning
// DONE: act で AudioProbe が失敗した場合エラーを返す                   → TestRunAct_AudioProbe_FailureCausesError

func TestRunAct_AudioProbe_CalledOnLocalPlayerPath(t *testing.T) {
	probeCalled := false

	dir := t.TempDir()
	lockPath := filepath.Join(dir, "viewer.lock")
	scriptFile := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(scriptFile, []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := newCmdWithContext(t)
	registerCommonFlags(cmd)
	_ = cmd.ParseFlags([]string{})

	deps := &ActDeps{
		Reader:           stubScriptReaderForAct{scripts: []entity.Script{{Text: "hello"}}},
		ClientFactory:    func(_ string) app.VoicevoxClient { return &mockActClient{} },
		Player:           &trackingPlayer{},
		LockPathResolver: func() (string, error) { return lockPath, nil },
		AudioProbe:       func() error { probeCalled = true; return nil },
	}

	if err := runAct(cmd, []string{scriptFile}, deps); err != nil {
		t.Fatalf("runAct: %v", err)
	}
	if !probeCalled {
		t.Error("expected AudioProbe to be called on local player path, but it was not")
	}
}

func TestRunAct_AudioProbe_SkippedOnDryRun(t *testing.T) {
	probeCalled := false

	dir := t.TempDir()
	scriptFile := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(scriptFile, []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := newCmdWithContext(t)
	registerCommonFlags(cmd)
	_ = cmd.ParseFlags([]string{"--dry-run"})

	deps := &ActDeps{
		Reader:        stubScriptReaderForAct{scripts: []entity.Script{{Text: "hello"}}},
		ClientFactory: func(_ string) app.VoicevoxClient { return &mockActClient{} },
		Player:        &trackingPlayer{},
		AudioProbe:    func() error { probeCalled = true; return nil },
	}

	if err := runAct(cmd, []string{scriptFile}, deps); err != nil {
		t.Fatalf("runAct: %v", err)
	}
	if probeCalled {
		t.Error("expected AudioProbe to be skipped in dry-run, but it was called")
	}
}

func TestRunAct_AudioProbe_SkippedWhenViewerRunning(t *testing.T) {
	probeCalled := false

	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("/api/play", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"silent": false})
	})
	lockPath, _ := setupFakeViewer(t, mux)

	dir := t.TempDir()
	scriptFile := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(scriptFile, []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := newCmdWithContext(t)
	registerCommonFlags(cmd)
	_ = cmd.ParseFlags([]string{})

	deps := &ActDeps{
		Reader:           stubScriptReaderForAct{scripts: []entity.Script{{Text: "hello"}}},
		ClientFactory:    func(_ string) app.VoicevoxClient { return &noopClient{} },
		Player:           &noopPlayer{},
		LockPathResolver: func() (string, error) { return lockPath, nil },
		AudioProbe:       func() error { probeCalled = true; return nil },
	}

	if err := runAct(cmd, []string{scriptFile}, deps); err != nil {
		t.Fatalf("runAct: %v", err)
	}
	if probeCalled {
		t.Error("expected AudioProbe to be skipped when viewer is running, but it was called")
	}
}

func TestRunAct_AudioProbe_FailureCausesError(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, "viewer.lock")
	scriptFile := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(scriptFile, []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := newCmdWithContext(t)
	registerCommonFlags(cmd)
	_ = cmd.ParseFlags([]string{})

	deps := &ActDeps{
		Reader:           stubScriptReaderForAct{scripts: []entity.Script{{Text: "hello"}}},
		ClientFactory:    func(_ string) app.VoicevoxClient { return &mockActClient{} },
		Player:           &trackingPlayer{},
		LockPathResolver: func() (string, error) { return lockPath, nil },
		AudioProbe:       func() error { return errors.New("no audio device") },
	}

	err := runAct(cmd, []string{scriptFile}, deps)
	if err == nil {
		t.Fatal("expected error when AudioProbe fails, got nil")
	}
	if !strings.Contains(err.Error(), "audio") {
		t.Errorf("expected error message to mention 'audio', got: %v", err)
	}
}
