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

// findWatchCmd はrootCmdからwatchサブコマンドを検索して返すテストヘルパー。
func findWatchCmd(t *testing.T, rootCmd *cobra.Command) *cobra.Command {
	t.Helper()
	for _, c := range rootCmd.Commands() {
		if c.Name() == "watch" {
			return c
		}
	}
	t.Fatal("watch subcommand not found")
	return nil
}

func TestWatchCmd_RegisteredAsSubcommand(t *testing.T) {
	rootCmd := makeRootCmd()

	found := false
	for _, c := range rootCmd.Commands() {
		if c.Name() == "watch" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'watch' subcommand to be registered")
	}
}

func TestWatchCmd_DefaultOptionValues(t *testing.T) {
	rootCmd := makeRootCmd()
	watchCmd := findWatchCmd(t, rootCmd)

	engineURL, _ := watchCmd.Flags().GetString("engine-url")
	if engineURL != "http://localhost:50021" {
		t.Errorf("expected default engine-url 'http://localhost:50021', got: %s", engineURL)
	}

	speaker, _ := watchCmd.Flags().GetInt("speaker")
	if speaker != 3 {
		t.Errorf("expected default speaker 3, got: %d", speaker)
	}

	speed, _ := watchCmd.Flags().GetFloat64("speed")
	if speed != 1.0 {
		t.Errorf("expected default speed 1.0, got: %f", speed)
	}

	pitch, _ := watchCmd.Flags().GetFloat64("pitch")
	if pitch != 0.0 {
		t.Errorf("expected default pitch 0.0, got: %f", pitch)
	}

	intonation, _ := watchCmd.Flags().GetFloat64("intonation")
	if intonation != 1.0 {
		t.Errorf("expected default intonation 1.0, got: %f", intonation)
	}

	deleteMode, _ := watchCmd.Flags().GetBool("delete")
	if deleteMode {
		t.Error("expected --delete default to be false")
	}

	stream, _ := watchCmd.Flags().GetBool("stream")
	if stream {
		t.Error("expected --stream default to be false")
	}

	streamAddr, _ := watchCmd.Flags().GetString("stream-addr")
	if streamAddr != "127.0.0.1:8080" {
		t.Errorf("expected default stream-addr '127.0.0.1:8080', got: %s", streamAddr)
	}

	verbose, _ := watchCmd.Flags().GetBool("verbose")
	if verbose {
		t.Error("expected --verbose default to be false")
	}
}

func TestWatchCmd_NoArgs_ReturnsUsageError(t *testing.T) {
	rootCmd := makeRootCmd()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"watch"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when no args provided, got nil")
	}
	if !errors.Is(err, ErrUsage) {
		t.Errorf("expected ErrUsage, got: %v", err)
	}
}

func TestWatchCmd_WithFile_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	file := dir + "/test.txt"
	if err := os.WriteFile(file, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	rootCmd := makeRootCmd()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"watch", file})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when file path is given")
	}
	if !errors.Is(err, ErrUsage) {
		t.Errorf("expected ErrUsage, got: %v", err)
	}
}

func TestWatchCmd_WithNonExistentPath_ReturnsError(t *testing.T) {
	rootCmd := makeRootCmd()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"watch", "/path/that/does/not/exist"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when non-existent path is given")
	}
	if !errors.Is(err, ErrUsage) {
		t.Errorf("expected ErrUsage, got: %v", err)
	}
}

func TestWatchCmd_OneOfMultiplePathsMissing_ReturnsError(t *testing.T) {
	dir := t.TempDir()

	rootCmd := makeRootCmd()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"watch", dir, "/path/that/does/not/exist"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when any of the paths is missing")
	}
	if !errors.Is(err, ErrUsage) {
		t.Errorf("expected ErrUsage, got: %v", err)
	}
}

func TestWatchCmd_HelpContainsFlags(t *testing.T) {
	rootCmd := makeRootCmd()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"watch", "--help"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error for --help, got: %v", err)
	}

	output := buf.String()
	flags := []string{"--engine-url", "--speaker", "--speed", "--pitch", "--intonation", "--delete", "--stream", "--stream-addr", "--verbose", "--dry-run"}
	for _, flag := range flags {
		if !strings.Contains(output, flag) {
			t.Errorf("expected help output to contain '%s'", flag)
		}
	}

	// --stream-history-size は #226 で廃止された
	if strings.Contains(output, "--stream-history-size") {
		t.Error("expected help output NOT to contain '--stream-history-size'")
	}
}

func TestWatchCmd_StreamAndDryRun_ReturnsUsageError(t *testing.T) {
	dir := t.TempDir()

	rootCmd := makeRootCmd()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"watch", "--stream", "--dry-run", dir})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when --stream and --dry-run are combined")
	}
	if !errors.Is(err, ErrUsage) {
		t.Errorf("expected ErrUsage, got: %v", err)
	}
}

// --- #228 /speakers 取得 ---

// stubVoicevoxClient は GetSpeakers/HealthCheck の振る舞いをカスタマイズできるテスト用クライアント。
type stubVoicevoxClient struct {
	healthCheckErr  error
	getSpeakersErr  error
	speakers        []entity.Speaker
	getSpeakersCall int
}

func (c *stubVoicevoxClient) HealthCheck(_ context.Context) error { return c.healthCheckErr }
func (c *stubVoicevoxClient) CreateQuery(_ context.Context, _ string, _ int) (*entity.AudioQuery, error) {
	return nil, nil
}
func (c *stubVoicevoxClient) Synthesize(_ context.Context, _ *entity.AudioQuery, _ int) ([]byte, error) {
	return nil, nil
}
func (c *stubVoicevoxClient) GetSpeakers(_ context.Context) ([]entity.Speaker, error) {
	c.getSpeakersCall++
	return c.speakers, c.getSpeakersErr
}

type stubScriptReader struct{}

func (stubScriptReader) Read(_ string) ([]entity.Script, error) { return nil, nil }

type stubFileMover struct{}

func (stubFileMover) MoveToDone(_ string) error { return nil }
func (stubFileMover) Delete(_ string) error     { return nil }

type stubDirWatcher struct{}

func (stubDirWatcher) Watch(ctx context.Context, _ string) (<-chan string, <-chan error) {
	fileCh := make(chan string)
	errCh := make(chan error)
	go func() {
		<-ctx.Done()
		close(fileCh)
		close(errCh)
	}()
	return fileCh, errCh
}

type stubAudioPlayer struct{}

func (stubAudioPlayer) Play(_ context.Context, _ []byte, _ app.PlayMeta) error { return nil }

type stubStreamPlayer struct{}

func (stubStreamPlayer) Start(_ context.Context) error                          { return nil }
func (stubStreamPlayer) Shutdown(_ context.Context) error                       { return nil }
func (stubStreamPlayer) Addr() string                                           { return "127.0.0.1:0" }
func (stubStreamPlayer) Play(_ context.Context, _ []byte, _ app.PlayMeta) error { return nil }

func TestWatchCmd_Stream_GetSpeakersFailure_ReturnsError(t *testing.T) {
	dir := t.TempDir()

	stubClient := &stubVoicevoxClient{
		getSpeakersErr: errors.New("speakers endpoint down"),
	}
	deps := &Deps{
		Watch: &WatchDeps{
			Reader:            stubScriptReader{},
			ClientFactory:     func(_ string) app.VoicevoxClient { return stubClient },
			Player:            stubAudioPlayer{},
			Mover:             stubFileMover{},
			DirWatcherFactory: func(_ *slog.Logger) app.DirWatcher { return stubDirWatcher{} },
			StreamPlayerFactory: func(_ string, _ *slog.Logger, _ map[int]entity.SpeakerStyleInfo, _ app.VoicevoxClient) (app.StreamPlayer, error) {
				t.Error("StreamPlayerFactory should not be called when GetSpeakers fails")
				return stubStreamPlayer{}, nil
			},
		},
	}
	rootCmd := makeRootCmd(deps)
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"watch", "--stream", dir})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when GetSpeakers fails")
	}
	if !strings.Contains(err.Error(), "failed to get speakers") {
		t.Errorf("expected wrapped 'failed to get speakers', got: %v", err)
	}
	if stubClient.getSpeakersCall != 1 {
		t.Errorf("expected GetSpeakers to be called once, got %d", stubClient.getSpeakersCall)
	}
}

func TestWatchCmd_Stream_PassesSpeakerLookupToFactory(t *testing.T) {
	dir := t.TempDir()

	stubClient := &stubVoicevoxClient{
		speakers: []entity.Speaker{
			{Name: "ずんだもん", Styles: []entity.SpeakerStyle{{ID: 3, Name: "ノーマル"}}},
		},
	}

	var captured map[int]entity.SpeakerStyleInfo
	deps := &Deps{
		Watch: &WatchDeps{
			Reader:            stubScriptReader{},
			ClientFactory:     func(_ string) app.VoicevoxClient { return stubClient },
			Player:            stubAudioPlayer{},
			Mover:             stubFileMover{},
			DirWatcherFactory: func(_ *slog.Logger) app.DirWatcher { return stubDirWatcher{} },
			StreamPlayerFactory: func(_ string, _ *slog.Logger, lookup map[int]entity.SpeakerStyleInfo, _ app.VoicevoxClient) (app.StreamPlayer, error) {
				captured = lookup
				// usecase.Run まで進ませず、watch loop から確実に抜けるためにキャンセルを返す。
				return stubStreamPlayer{}, nil
			},
		},
	}
	rootCmd := makeRootCmd(deps)
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	// usecase.Run 内のディレクトリ監視は stubDirWatcher が ctx.Done() を待つだけなので、
	// 短い timeout を持つ context を作って即座に抜ける。
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	rootCmd.SetContext(ctx)
	rootCmd.SetArgs([]string{"watch", "--stream", dir})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if captured == nil {
		t.Fatal("StreamPlayerFactory was not called with a lookup")
	}
	info, ok := captured[3]
	if !ok {
		t.Fatalf("lookup missing key 3, got: %+v", captured)
	}
	if info.SpeakerName != "ずんだもん" || info.StyleName != "ノーマル" {
		t.Errorf("unexpected lookup[3]=%+v", info)
	}
}

func TestWatchCmd_EnvVarVOXEngineURL(t *testing.T) {
	t.Setenv("VOX_ENGINE_URL", "http://custom:9999")

	rootCmd := makeRootCmd()
	watchCmd := findWatchCmd(t, rootCmd)

	engineURL, _ := watchCmd.Flags().GetString("engine-url")
	if engineURL != "http://custom:9999" {
		t.Errorf("expected engine-url to be 'http://custom:9999', got: %s", engineURL)
	}
}

func TestWatchCmd_EnvVarVOXSpeaker(t *testing.T) {
	t.Setenv("VOX_SPEAKER", "42")

	rootCmd := makeRootCmd()
	watchCmd := findWatchCmd(t, rootCmd)

	speaker, _ := watchCmd.Flags().GetInt("speaker")
	if speaker != 42 {
		t.Errorf("expected speaker to be 42, got: %d", speaker)
	}
}

// --- #239 --queue オプション ---

// makeQueueWatchDeps は --queue のテストで共通利用する WatchDeps を組み立てる。
// resolver は呼び出された回数と返却パスを captured を通じて観測できる。
func makeQueueWatchDeps(resolver func() (string, error)) *Deps {
	return &Deps{
		Watch: &WatchDeps{
			Reader:            stubScriptReader{},
			ClientFactory:     func(_ string) app.VoicevoxClient { return &stubVoicevoxClient{} },
			Player:            stubAudioPlayer{},
			Mover:             stubFileMover{},
			DirWatcherFactory: func(_ *slog.Logger) app.DirWatcher { return stubDirWatcher{} },
			QueuePathResolver: resolver,
		},
	}
}

func TestWatchCmd_Queue_ResolvesAndCreatesQueueDir(t *testing.T) {
	baseDir := t.TempDir()
	queueDir := baseDir + "/.vox-actor/queue"

	resolverCalled := 0
	deps := makeQueueWatchDeps(func() (string, error) {
		resolverCalled++
		return queueDir, nil
	})

	rootCmd := makeRootCmd(deps)
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	rootCmd.SetContext(ctx)
	rootCmd.SetArgs([]string{"watch", "--queue"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
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

func TestWatchCmd_Queue_WithPositionalArg_ReturnsUsageError(t *testing.T) {
	dir := t.TempDir()
	deps := makeQueueWatchDeps(func() (string, error) {
		t.Error("resolver should not be called when usage error occurs")
		return "", nil
	})

	rootCmd := makeRootCmd(deps)
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"watch", "--queue", dir})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when --queue is combined with a positional arg")
	}
	if !errors.Is(err, ErrUsage) {
		t.Errorf("expected ErrUsage, got: %v", err)
	}
	if !strings.Contains(err.Error(), "--queue") {
		t.Errorf("expected error to mention --queue, got: %v", err)
	}
}

func TestWatchCmd_NoArgsAndNoQueue_ReturnsUsageError(t *testing.T) {
	rootCmd := makeRootCmd()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"watch"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when neither args nor --queue is given")
	}
	if !errors.Is(err, ErrUsage) {
		t.Errorf("expected ErrUsage, got: %v", err)
	}
}

func TestWatchCmd_Queue_ResolverReturnsNotInRepo_ReturnsError(t *testing.T) {
	deps := makeQueueWatchDeps(func() (string, error) {
		return "", infra.ErrNotInGitRepo
	})

	rootCmd := makeRootCmd(deps)
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"watch", "--queue"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when resolver fails with not-in-git-repo")
	}
	if !strings.Contains(err.Error(), "gitリポジトリではありません") {
		t.Errorf("expected error message to contain 'gitリポジトリではありません', got: %v", err)
	}
}

func TestWatchCmd_Queue_ResolverReturnsGitNotFound_ReturnsError(t *testing.T) {
	deps := makeQueueWatchDeps(func() (string, error) {
		return "", infra.ErrGitNotFound
	})

	rootCmd := makeRootCmd(deps)
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"watch", "--queue"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when resolver fails with git-not-found")
	}
	if !strings.Contains(err.Error(), "gitコマンドが見つかりません") {
		t.Errorf("expected error message to contain 'gitコマンドが見つかりません', got: %v", err)
	}
}

func TestWatchCmd_HelpContainsQueueFlag(t *testing.T) {
	rootCmd := makeRootCmd()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"watch", "--help"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("expected no error for --help, got: %v", err)
	}
	if !strings.Contains(buf.String(), "--queue") {
		t.Errorf("expected help output to contain '--queue'")
	}
}
