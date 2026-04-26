package cmd

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os"
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
	deleteMode, _ := viewerCmd.Flags().GetBool("delete")
	if deleteMode {
		t.Error("expected --delete default to be false")
	}
	watchQueue, _ := viewerCmd.Flags().GetBool("watch-queue")
	if watchQueue {
		t.Error("expected --watch-queue default to be false")
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
	flags := []string{"--engine-url", "--speaker", "--speed", "--pitch", "--intonation", "--host", "--port", "--watch", "--watch-queue", "--delete", "--verbose"}
	for _, flag := range flags {
		if !strings.Contains(output, flag) {
			t.Errorf("expected help output to contain '%s'\noutput:\n%s", flag, output)
		}
	}
	if strings.Contains(output, "--dry-run") {
		t.Error("viewer help should NOT contain '--dry-run'")
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
					StreamPlayerFactory: func(_ string, _ *slog.Logger, _ map[int]entity.SpeakerStyleInfo, _ app.VoicevoxClient) (app.StreamPlayer, error) {
						return &stubStreamPlayer{}, nil
					},
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

func TestViewerCmd_WatchWithFile_ReturnsUsageError(t *testing.T) {
	dir := t.TempDir()
	file := dir + "/test.txt"
	if err := os.WriteFile(file, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	deps := &Deps{
		Viewer: &ViewerDeps{
			ClientFactory: func(_ string) app.VoicevoxClient { return &stubVoicevoxClient{} },
			StreamPlayerFactory: func(_ string, _ *slog.Logger, _ map[int]entity.SpeakerStyleInfo, _ app.VoicevoxClient) (app.StreamPlayer, error) {
				return &stubStreamPlayer{}, nil
			},
		},
	}
	rootCmd := makeRootCmd(deps)
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"viewer", "--watch", file})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when file path is given to --watch")
	}
	if !errors.Is(err, ErrUsage) {
		t.Errorf("expected ErrUsage, got: %v", err)
	}
}

func TestViewerCmd_WatchWithNonExistentPath_ReturnsUsageError(t *testing.T) {
	deps := &Deps{
		Viewer: &ViewerDeps{
			ClientFactory: func(_ string) app.VoicevoxClient { return &stubVoicevoxClient{} },
			StreamPlayerFactory: func(_ string, _ *slog.Logger, _ map[int]entity.SpeakerStyleInfo, _ app.VoicevoxClient) (app.StreamPlayer, error) {
				return &stubStreamPlayer{}, nil
			},
		},
	}
	rootCmd := makeRootCmd(deps)
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"viewer", "--watch", "/path/that/does/not/exist"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when non-existent path given to --watch")
	}
	if !errors.Is(err, ErrUsage) {
		t.Errorf("expected ErrUsage, got: %v", err)
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
			Reader:            stubScriptReader{},
			ClientFactory:     func(_ string) app.VoicevoxClient { return &stubVoicevoxClient{healthCheckErr: errors.New("down")} },
			Mover:             stubFileMover{},
			DirWatcherFactory: func(_ *slog.Logger) app.DirWatcher { return stubDirWatcher{} },
			StreamPlayerFactory: func(_ string, _ *slog.Logger, _ map[int]entity.SpeakerStyleInfo, _ app.VoicevoxClient) (app.StreamPlayer, error) {
				return sp, nil
			},
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
			Reader: stubScriptReader{},
			ClientFactory: func(_ string) app.VoicevoxClient {
				return &stubVoicevoxClient{getSpeakersErr: errors.New("speakers down")}
			},
			Mover:             stubFileMover{},
			DirWatcherFactory: func(_ *slog.Logger) app.DirWatcher { return stubDirWatcher{} },
			StreamPlayerFactory: func(_ string, _ *slog.Logger, _ map[int]entity.SpeakerStyleInfo, _ app.VoicevoxClient) (app.StreamPlayer, error) {
				return sp, nil
			},
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
			Reader: stubScriptReader{},
			ClientFactory: func(_ string) app.VoicevoxClient {
				return &stubVoicevoxClient{
					speakers: []entity.Speaker{
						{Name: "ずんだもん", Styles: []entity.SpeakerStyle{{ID: 3, Name: "ノーマル"}}},
					},
				}
			},
			Mover:             stubFileMover{},
			DirWatcherFactory: func(_ *slog.Logger) app.DirWatcher { return stubDirWatcher{} },
			StreamPlayerFactory: func(_ string, _ *slog.Logger, _ map[int]entity.SpeakerStyleInfo, _ app.VoicevoxClient) (app.StreamPlayer, error) {
				return sp, nil
			},
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

func TestViewerCmd_WatchQueue_ResolvesAndCreatesQueueDir(t *testing.T) {
	baseDir := t.TempDir()
	queueDir := baseDir + "/.vox-actor/queue"

	resolverCalled := 0
	sp := &stubStreamPlayer{}
	deps := &Deps{
		Viewer: &ViewerDeps{
			Reader:            stubScriptReader{},
			ClientFactory:     func(_ string) app.VoicevoxClient { return &stubVoicevoxClient{} },
			Mover:             stubFileMover{},
			DirWatcherFactory: func(_ *slog.Logger) app.DirWatcher { return stubDirWatcher{} },
			StreamPlayerFactory: func(_ string, _ *slog.Logger, _ map[int]entity.SpeakerStyleInfo, _ app.VoicevoxClient) (app.StreamPlayer, error) {
				return sp, nil
			},
			QueuePathResolver: func() (string, error) {
				resolverCalled++
				return queueDir, nil
			},
		},
	}

	_ = runViewerWithDeps(t, deps, "viewer", "--watch-queue")

	if resolverCalled != 1 {
		t.Errorf("expected QueuePathResolver to be called once, got %d", resolverCalled)
	}
	info, err := os.Stat(queueDir)
	if err != nil {
		t.Fatalf("expected queue directory to be auto-created at %s: %v", queueDir, err)
	}
	if !info.IsDir() {
		t.Errorf("expected %s to be a directory", queueDir)
	}
}

func TestViewerCmd_WatchQueue_ResolverReturnsNotInRepo_ReturnsError(t *testing.T) {
	deps := &Deps{
		Viewer: &ViewerDeps{
			ClientFactory: func(_ string) app.VoicevoxClient { return &stubVoicevoxClient{} },
			StreamPlayerFactory: func(_ string, _ *slog.Logger, _ map[int]entity.SpeakerStyleInfo, _ app.VoicevoxClient) (app.StreamPlayer, error) {
				return &stubStreamPlayer{}, nil
			},
			QueuePathResolver: func() (string, error) {
				return "", infra.ErrNotInGitRepo
			},
		},
	}

	rootCmd := makeRootCmd(deps)
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"viewer", "--watch-queue"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when resolver fails with not-in-git-repo")
	}
	if !strings.Contains(err.Error(), "gitリポジトリではありません") {
		t.Errorf("expected error to contain 'gitリポジトリではありません', got: %v", err)
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
