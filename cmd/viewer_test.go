package cmd

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/canpok1/vox-actor/internal/app"
	"github.com/canpok1/vox-actor/internal/domain/entity"
	"github.com/canpok1/vox-actor/internal/infra"
	"github.com/spf13/cobra"
)

type stubStreamPlayer struct {
	silentReason string
}

func (p *stubStreamPlayer) Start(_ context.Context) error                          { return nil }
func (p *stubStreamPlayer) Shutdown(_ context.Context) error                       { return nil }
func (p *stubStreamPlayer) Addr() string                                           { return "127.0.0.1:0" }
func (p *stubStreamPlayer) Play(_ context.Context, _ []byte, _ app.PlayMeta) error { return nil }
func (p *stubStreamPlayer) SetSilent(reason string)                                { p.silentReason = reason }
func (p *stubStreamPlayer) PlayText(_ context.Context, _ app.PlayMeta) error       { return nil }

// captureStderr は関数実行中の os.Stderr への出力を文字列として収集する。
// buildLoggerFromFlags が os.Stderr に書くため、WARN ログの検証に使う。
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stderr = w

	done := make(chan string, 1)
	go func() {
		var out bytes.Buffer
		_, _ = out.ReadFrom(r)
		done <- out.String()
	}()

	fn()

	_ = w.Close()
	os.Stderr = orig
	return <-done
}

func findViewerCmd(t *testing.T, rootCmd *cobra.Command) *cobra.Command {
	t.Helper()
	for _, c := range rootCmd.Commands() {
		if c.Name() == "viewer" {
			return c
		}
	}
	t.Fatal("viewer subcommand not found")
	return nil
}

// tempLockResolver はテスト用に一時ディレクトリのロックパスを返す LockPathResolver を作る。
// viewer が実際に起動中でも干渉しない隔離された lock path を使うため各テストで利用する。
func tempLockResolver(t *testing.T) func() (string, error) {
	t.Helper()
	dir := t.TempDir()
	lockPath := filepath.Join(dir, "viewer.lock")
	return func() (string, error) { return lockPath, nil }
}

func TestViewerCmd_RegisteredAsSubcommand(t *testing.T) {
	rootCmd := makeRootCmd()

	found := false
	for _, c := range rootCmd.Commands() {
		if c.Name() == "viewer" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'viewer' subcommand to be registered")
	}
}

func TestViewerCmd_DefaultOptionValues(t *testing.T) {
	rootCmd := makeRootCmd()
	viewerCmd := findViewerCmd(t, rootCmd)

	engineURL, _ := viewerCmd.Flags().GetString("engine-url")
	if engineURL != "http://localhost:50021" {
		t.Errorf("expected default engine-url 'http://localhost:50021', got: %s", engineURL)
	}
	host, _ := viewerCmd.Flags().GetString("host")
	if host != "127.0.0.1" {
		t.Errorf("expected default host '127.0.0.1', got: %s", host)
	}
	port, _ := viewerCmd.Flags().GetInt("port")
	if port != 8080 {
		t.Errorf("expected default port 8080, got: %d", port)
	}
	speaker, _ := viewerCmd.Flags().GetInt("speaker")
	if speaker != 3 {
		t.Errorf("expected default speaker 3, got: %d", speaker)
	}
	speed, _ := viewerCmd.Flags().GetFloat64("speed")
	if speed != 1.0 {
		t.Errorf("expected default speed 1.0, got: %f", speed)
	}
	pitch, _ := viewerCmd.Flags().GetFloat64("pitch")
	if pitch != 0.0 {
		t.Errorf("expected default pitch 0.0, got: %f", pitch)
	}
	intonation, _ := viewerCmd.Flags().GetFloat64("intonation")
	if intonation != 1.0 {
		t.Errorf("expected default intonation 1.0, got: %f", intonation)
	}
	verbose, _ := viewerCmd.Flags().GetBool("verbose")
	if verbose {
		t.Error("expected --verbose default to be false")
	}
}

func TestViewerCmd_NoDryRunFlag(t *testing.T) {
	rootCmd := makeRootCmd()
	viewerCmd := findViewerCmd(t, rootCmd)
	if viewerCmd.Flags().Lookup("dry-run") != nil {
		t.Error("viewer should not have --dry-run flag")
	}
}

func TestViewerCmd_NoWatchFlags(t *testing.T) {
	rootCmd := makeRootCmd()
	viewerCmd := findViewerCmd(t, rootCmd)
	if viewerCmd.Flags().Lookup("watch") != nil {
		t.Error("viewer should not have --watch flag (use 'vox-actor watch' instead)")
	}
	if viewerCmd.Flags().Lookup("watch-queue") != nil {
		t.Error("viewer should not have --watch-queue flag (use 'vox-actor watch' instead)")
	}
	if viewerCmd.Flags().Lookup("delete") != nil {
		t.Error("viewer should not have --delete flag")
	}
}

func TestViewerCmd_HelpContainsFlags(t *testing.T) {
	rootCmd := makeRootCmd()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"viewer", "--help"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error for --help, got: %v", err)
	}

	output := buf.String()
	flags := []string{"--engine-url", "--speaker", "--speed", "--pitch", "--intonation", "--host", "--port", "--verbose"}
	for _, flag := range flags {
		if !strings.Contains(output, flag) {
			t.Errorf("expected help output to contain '%s'\noutput:\n%s", flag, output)
		}
	}
	if strings.Contains(output, "--dry-run") {
		t.Error("viewer help should NOT contain '--dry-run'")
	}
	if strings.Contains(output, "--watch") {
		t.Error("viewer help should NOT contain '--watch'")
	}
	if strings.Contains(output, "--watch-queue") {
		t.Error("viewer help should NOT contain '--watch-queue'")
	}
	if strings.Contains(output, "--delete") {
		t.Error("viewer help should NOT contain '--delete'")
	}
}

func TestViewerCmd_PortOutOfRange_ReturnsUsageError(t *testing.T) {
	cases := []struct {
		name string
		port string
	}{
		{"port 0", "0"},
		{"port 65536", "65536"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			deps := &Deps{
				Viewer: &ViewerDeps{
					ClientFactory: func(_ string) app.VoicevoxClient { return &stubVoicevoxClient{} },
					StreamPlayerFactory: func(_ string, _ *slog.Logger, _ map[int]entity.SpeakerStyleInfo, _ []int, _ app.VoicevoxClient) (app.StreamPlayer, error) {
						return &stubStreamPlayer{}, nil
					},
					LockPathResolver: tempLockResolver(t),
				},
			}
			rootCmd := makeRootCmd(deps)
			buf := new(bytes.Buffer)
			rootCmd.SetOut(buf)
			rootCmd.SetErr(buf)
			rootCmd.SetArgs([]string{"viewer", "--port", tc.port})

			err := rootCmd.Execute()
			if err == nil {
				t.Fatalf("expected error for port %s", tc.port)
			}
			if !errors.Is(err, ErrUsage) {
				t.Errorf("expected ErrUsage for port %s, got: %v", tc.port, err)
			}
		})
	}
}

func runViewerWithDeps(t *testing.T, deps *Deps, args ...string) error {
	t.Helper()
	rootCmd := makeRootCmd(deps)
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	rootCmd.SetContext(ctx)
	rootCmd.SetArgs(args)
	return rootCmd.Execute()
}

func TestViewerCmd_HealthCheckFailure_FallsBackToSilent(t *testing.T) {
	sp := &stubStreamPlayer{}
	deps := &Deps{
		Viewer: &ViewerDeps{
			ClientFactory: func(_ string) app.VoicevoxClient { return &stubVoicevoxClient{healthCheckErr: errors.New("down")} },
			StreamPlayerFactory: func(_ string, _ *slog.Logger, _ map[int]entity.SpeakerStyleInfo, _ []int, _ app.VoicevoxClient) (app.StreamPlayer, error) {
				return sp, nil
			},
			LockPathResolver: tempLockResolver(t),
		},
	}

	logs := captureStderr(t, func() {
		_ = runViewerWithDeps(t, deps, "viewer")
	})

	if sp.silentReason == "" {
		t.Errorf("expected SetSilent called with non-empty reason")
	}
	if !strings.Contains(logs, "VOICEVOX engine unreachable") {
		t.Errorf("expected WARN log 'VOICEVOX engine unreachable', got: %s", logs)
	}
}

func TestViewerCmd_GetSpeakersFailure_FallsBackToSilent(t *testing.T) {
	sp := &stubStreamPlayer{}
	deps := &Deps{
		Viewer: &ViewerDeps{
			ClientFactory: func(_ string) app.VoicevoxClient {
				return &stubVoicevoxClient{getSpeakersErr: errors.New("speakers down")}
			},
			StreamPlayerFactory: func(_ string, _ *slog.Logger, _ map[int]entity.SpeakerStyleInfo, _ []int, _ app.VoicevoxClient) (app.StreamPlayer, error) {
				return sp, nil
			},
			LockPathResolver: tempLockResolver(t),
		},
	}

	logs := captureStderr(t, func() {
		_ = runViewerWithDeps(t, deps, "viewer")
	})

	if sp.silentReason == "" {
		t.Errorf("expected SetSilent called with non-empty reason")
	}
	if !strings.Contains(logs, "VOICEVOX engine unreachable") {
		t.Errorf("expected WARN log 'VOICEVOX engine unreachable', got: %s", logs)
	}
}

func TestViewerCmd_EngineHealthy_DoesNotEnterSilent(t *testing.T) {
	sp := &stubStreamPlayer{}
	deps := &Deps{
		Viewer: &ViewerDeps{
			ClientFactory: func(_ string) app.VoicevoxClient {
				return &stubVoicevoxClient{
					speakers: []entity.Speaker{
						{Name: "ずんだもん", Styles: []entity.SpeakerStyle{{ID: 3, Name: "ノーマル"}}},
					},
				}
			},
			StreamPlayerFactory: func(_ string, _ *slog.Logger, _ map[int]entity.SpeakerStyleInfo, _ []int, _ app.VoicevoxClient) (app.StreamPlayer, error) {
				return sp, nil
			},
			LockPathResolver: tempLockResolver(t),
		},
	}

	logs := captureStderr(t, func() {
		_ = runViewerWithDeps(t, deps, "viewer")
	})

	if sp.silentReason != "" {
		t.Errorf("expected SetSilent not called in normal mode, got %q", sp.silentReason)
	}
	if strings.Contains(logs, "VOICEVOX engine unreachable") {
		t.Errorf("expected no WARN log in normal mode, got: %s", logs)
	}
}

func TestViewerCmd_EnvVarVOXEngineURL(t *testing.T) {
	t.Setenv("VOX_ENGINE_URL", "http://custom:9999")
	rootCmd := makeRootCmd()
	viewerCmd := findViewerCmd(t, rootCmd)
	engineURL, _ := viewerCmd.Flags().GetString("engine-url")
	if engineURL != "http://custom:9999" {
		t.Errorf("expected engine-url to be 'http://custom:9999', got: %s", engineURL)
	}
}

func TestViewerCmd_EnvVarVOXSpeaker(t *testing.T) {
	t.Setenv("VOX_SPEAKER", "42")
	rootCmd := makeRootCmd()
	viewerCmd := findViewerCmd(t, rootCmd)
	speaker, _ := viewerCmd.Flags().GetInt("speaker")
	if speaker != 42 {
		t.Errorf("expected speaker to be 42, got: %d", speaker)
	}
}

func TestResolveViewerLockPath_UsesHomeDir(t *testing.T) {
	path, err := resolveViewerLockPath(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".vox-actor", "viewer", "viewer.lock")
	if path != expected {
		t.Errorf("expected path %q, got %q", expected, path)
	}
}

func TestResolveViewerLockPath_IgnoresVoxActorWorkspaceEnv(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("VOX_ACTOR_WORKSPACE", dir)

	path, err := resolveViewerLockPath(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(path, dir) {
		t.Errorf("resolveViewerLockPath() = %q, should not contain VOX_ACTOR_WORKSPACE dir %q", path, dir)
	}
}

func TestViewerCmd_AlreadyRunning_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, "viewer", "viewer.lock")

	// 先に別インスタンスとしてロックを取得しておく
	first, err := infra.AcquireViewerLock(lockPath)
	if err != nil {
		t.Fatalf("pre-lock failed: %v", err)
	}
	defer first.Release() //nolint:errcheck

	deps := &Deps{
		Viewer: &ViewerDeps{
			ClientFactory: func(_ string) app.VoicevoxClient { return &stubVoicevoxClient{} },
			StreamPlayerFactory: func(_ string, _ *slog.Logger, _ map[int]entity.SpeakerStyleInfo, _ []int, _ app.VoicevoxClient) (app.StreamPlayer, error) {
				return &stubStreamPlayer{}, nil
			},
			LockPathResolver: func() (string, error) { return lockPath, nil },
		},
	}

	err = runViewerWithDeps(t, deps, "viewer")
	if err == nil {
		t.Fatal("expected error when viewer is already running")
	}
	if !strings.Contains(err.Error(), "viewer は既に起動中です") {
		t.Errorf("expected error to contain 'viewer は既に起動中です', got: %v", err)
	}
}
