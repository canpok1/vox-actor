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
	"time"

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
	flags := []string{"--engine-url", "--speaker", "--speed", "--pitch", "--intonation", "--delete", "--verbose", "--dry-run"}
	for _, flag := range flags {
		if !strings.Contains(output, flag) {
			t.Errorf("expected help output to contain '%s'", flag)
		}
	}

	// --stream / --stream-addr は廃止済みのため help に表示されない
	if strings.Contains(output, "--stream") {
		t.Error("expected help output NOT to contain '--stream'")
	}

	// --stream-history-size は #226 で廃止された
	if strings.Contains(output, "--stream-history-size") {
		t.Error("expected help output NOT to contain '--stream-history-size'")
	}
}

func TestWatchCmd_StreamFlagRemoved_ReturnsUsageError(t *testing.T) {
	dir := t.TempDir()

	rootCmd := makeRootCmd()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"watch", "--stream", dir})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when --stream is used")
	}
	if !errors.Is(err, ErrUsage) {
		t.Errorf("expected ErrUsage, got: %v", err)
	}
	if !strings.Contains(err.Error(), "viewer") {
		t.Errorf("expected error to mention 'viewer', got: %v", err)
	}
}

func TestWatchCmd_StreamAddrFlagRemoved_ReturnsUsageError(t *testing.T) {
	dir := t.TempDir()

	rootCmd := makeRootCmd()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"watch", "--stream-addr", "0.0.0.0:9090", dir})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when --stream-addr is used")
	}
	if !errors.Is(err, ErrUsage) {
		t.Errorf("expected ErrUsage, got: %v", err)
	}
	if !strings.Contains(err.Error(), "viewer") {
		t.Errorf("expected error to mention 'viewer', got: %v", err)
	}
}

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

// --- AudioProbe テスト (#468) ---
// DONE: watch で非dryRun時に起動直後 AudioProbe が呼ばれる            → TestRunWatch_AudioProbe_CalledAtStartup
// DONE: watch で dry-run の場合 AudioProbe がスキップされる           → TestRunWatch_AudioProbe_SkippedOnDryRun
// DONE: watch で AudioProbe が失敗した場合起動拒否してエラーを返す    → TestRunWatch_AudioProbe_FailureCausesError

// --- #481 --viewer-url / lockfile auto-detect テスト ---
// DONE: --viewer-url フラグがヘルプに含まれる                                             → TestWatchCmd_HelpContainsViewerURL
// DONE: --viewer-url デフォルトは空文字                                                   → TestWatchCmd_ViewerURL_DefaultEmpty
// DONE: VOX_VIEWER_URL 環境変数でデフォルト値が設定される                                 → TestWatchCmd_EnvVarVOXViewerURL
// DONE: --viewer-url 指定時は AudioProbe をスキップする                                   → TestRunWatch_ExplicitViewerURL_SkipsAudioProbe
// DONE: lockfile auto-detect で viewer 検知時は AudioProbe をスキップする                 → TestRunWatch_LockfileAutoDetect_SkipsAudioProbe
// DONE: --viewer-url 指定時にファイル検知 → viewer に POST する                          → TestRunWatch_ExplicitViewerURL_PostsToViewer
// DONE: lockfile auto-detect で viewer 検知時にファイル検知 → viewer に POST する        → TestRunWatch_LockfileAutoDetect_PostsToViewer

// makeWatchDepsWithProbe は AudioProbe 付きの WatchDeps を組み立てるテストヘルパー。
func makeWatchDepsWithProbe(probe func() error) *WatchDeps {
	return &WatchDeps{
		Reader:            stubScriptReader{},
		ClientFactory:     func(_ string) app.VoicevoxClient { return &stubVoicevoxClient{} },
		Player:            stubAudioPlayer{},
		Mover:             stubFileMover{},
		DirWatcherFactory: func(_ *slog.Logger) app.DirWatcher { return stubDirWatcher{} },
		AudioProbe:        probe,
	}
}

func TestRunWatch_AudioProbe_CalledAtStartup(t *testing.T) {
	probeCalled := false
	watchDir := t.TempDir()

	cmd := newCmdWithContext(t)
	registerCommonFlags(cmd)
	cmd.Flags().Bool("delete", false, "")
	cmd.Flags().Bool("queue", false, "")
	_ = cmd.ParseFlags([]string{})

	deps := makeWatchDepsWithProbe(func() error { probeCalled = true; return nil })

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	cmd.SetContext(ctx)

	_ = runWatch(cmd, []string{watchDir}, deps)
	if !probeCalled {
		t.Error("expected AudioProbe to be called at startup, but it was not")
	}
}

func TestRunWatch_AudioProbe_SkippedOnDryRun(t *testing.T) {
	probeCalled := false
	watchDir := t.TempDir()

	cmd := newCmdWithContext(t)
	registerCommonFlags(cmd)
	cmd.Flags().Bool("delete", false, "")
	cmd.Flags().Bool("queue", false, "")
	_ = cmd.ParseFlags([]string{"--dry-run"})

	deps := makeWatchDepsWithProbe(func() error { probeCalled = true; return nil })

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	cmd.SetContext(ctx)

	_ = runWatch(cmd, []string{watchDir}, deps)
	if probeCalled {
		t.Error("expected AudioProbe to be skipped in dry-run, but it was called")
	}
}

func TestRunWatch_AudioProbe_FailureCausesError(t *testing.T) {
	watchDir := t.TempDir()

	cmd := newCmdWithContext(t)
	registerCommonFlags(cmd)
	cmd.Flags().Bool("delete", false, "")
	cmd.Flags().Bool("queue", false, "")
	_ = cmd.ParseFlags([]string{})

	deps := makeWatchDepsWithProbe(func() error { return errors.New("no audio device") })

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	cmd.SetContext(ctx)

	err := runWatch(cmd, []string{watchDir}, deps)
	if err == nil {
		t.Fatal("expected error when AudioProbe fails, got nil")
	}
	if !strings.Contains(err.Error(), "audio") {
		t.Errorf("expected error message to mention 'audio', got: %v", err)
	}
}

// --- #481 --viewer-url テスト ---

func TestWatchCmd_HelpContainsViewerURL(t *testing.T) {
	rootCmd := makeRootCmd()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"watch", "--help"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("expected no error for --help, got: %v", err)
	}
	if !strings.Contains(buf.String(), "--viewer-url") {
		t.Error("expected help output to contain '--viewer-url'")
	}
}

func TestWatchCmd_ViewerURL_DefaultEmpty(t *testing.T) {
	t.Setenv("VOX_VIEWER_URL", "")

	rootCmd := makeRootCmd()
	watchCmd := findWatchCmd(t, rootCmd)

	viewerURL, err := watchCmd.Flags().GetString("viewer-url")
	if err != nil {
		t.Fatalf("expected viewer-url flag to exist, got error: %v", err)
	}
	if viewerURL != "" {
		t.Errorf("expected default viewer-url to be empty, got: %s", viewerURL)
	}
}

func TestWatchCmd_EnvVarVOXViewerURL(t *testing.T) {
	t.Setenv("VOX_VIEWER_URL", "http://remote:8080")

	rootCmd := makeRootCmd()
	watchCmd := findWatchCmd(t, rootCmd)

	viewerURL, err := watchCmd.Flags().GetString("viewer-url")
	if err != nil {
		t.Fatalf("expected viewer-url flag to exist, got error: %v", err)
	}
	if viewerURL != "http://remote:8080" {
		t.Errorf("expected viewer-url 'http://remote:8080', got: %s", viewerURL)
	}
}

func TestRunWatch_ExplicitViewerURL_SkipsAudioProbe(t *testing.T) {
	probeCalled := false
	watchDir := t.TempDir()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/play", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"silent": false})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	cmd := newCmdWithContext(t)
	registerCommonFlags(cmd)
	cmd.Flags().Bool("delete", false, "")
	cmd.Flags().Bool("queue", false, "")
	_ = cmd.ParseFlags([]string{"--viewer-url", srv.URL})

	deps := &WatchDeps{
		Reader:            stubScriptReader{},
		ClientFactory:     func(_ string) app.VoicevoxClient { return &stubVoicevoxClient{} },
		Player:            stubAudioPlayer{},
		Mover:             stubFileMover{},
		DirWatcherFactory: func(_ *slog.Logger) app.DirWatcher { return stubDirWatcher{} },
		AudioProbe:        func() error { probeCalled = true; return nil },
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	cmd.SetContext(ctx)

	_ = runWatch(cmd, []string{watchDir}, deps)
	if probeCalled {
		t.Error("expected AudioProbe to be skipped when --viewer-url is set, but it was called")
	}
}

func TestRunWatch_LockfileAutoDetect_SkipsAudioProbe(t *testing.T) {
	probeCalled := false
	watchDir := t.TempDir()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("/api/play", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"silent": false})
	})
	lockPath, _ := setupFakeViewer(t, mux)

	cmd := newCmdWithContext(t)
	registerCommonFlags(cmd)
	cmd.Flags().Bool("delete", false, "")
	cmd.Flags().Bool("queue", false, "")
	_ = cmd.ParseFlags([]string{})

	deps := &WatchDeps{
		Reader:            stubScriptReader{},
		ClientFactory:     func(_ string) app.VoicevoxClient { return &stubVoicevoxClient{} },
		Player:            stubAudioPlayer{},
		Mover:             stubFileMover{},
		DirWatcherFactory: func(_ *slog.Logger) app.DirWatcher { return stubDirWatcher{} },
		AudioProbe:        func() error { probeCalled = true; return nil },
		LockPathResolver:  func() (string, error) { return lockPath, nil },
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	cmd.SetContext(ctx)

	_ = runWatch(cmd, []string{watchDir}, deps)
	if probeCalled {
		t.Error("expected AudioProbe to be skipped when viewer is auto-detected, but it was called")
	}
}

// oneFileDirWatcher は指定ファイルを1件送信してから待機するスタブ DirWatcher。
type oneFileDirWatcher struct {
	filePath string
}

func (w *oneFileDirWatcher) Watch(ctx context.Context, _ string) (<-chan string, <-chan error) {
	fileCh := make(chan string, 1)
	errCh := make(chan error)
	fileCh <- w.filePath
	go func() {
		<-ctx.Done()
		close(fileCh)
		close(errCh)
	}()
	return fileCh, errCh
}

func TestRunWatch_ExplicitViewerURL_PostsToViewer(t *testing.T) {
	watchDir := t.TempDir()
	scriptFile := filepath.Join(watchDir, "test.txt")
	if err := os.WriteFile(scriptFile, []byte("ウォッチテスト"), 0o644); err != nil {
		t.Fatal(err)
	}

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
	cmd.Flags().Bool("delete", false, "")
	cmd.Flags().Bool("queue", false, "")
	_ = cmd.ParseFlags([]string{"--viewer-url", srv.URL})

	deps := &WatchDeps{
		Reader:        infra.NewFileReader(),
		ClientFactory: func(_ string) app.VoicevoxClient { return &stubVoicevoxClient{} },
		Player:        stubAudioPlayer{},
		Mover:         stubFileMover{},
		DirWatcherFactory: func(_ *slog.Logger) app.DirWatcher {
			return &oneFileDirWatcher{filePath: scriptFile}
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cmd.SetContext(ctx)

	done := make(chan error, 1)
	go func() { done <- runWatch(cmd, []string{watchDir}, deps) }()

	// Wait for the POST to be received
	for i := 0; i < 100; i++ {
		if postCount > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	<-done

	if postCount == 0 {
		t.Error("expected at least 1 POST to /api/play, got 0")
	}
	if capturedText != "ウォッチテスト" {
		t.Errorf("expected text=%q, got %q", "ウォッチテスト", capturedText)
	}
}

func TestRunWatch_LockfileAutoDetect_PostsToViewer(t *testing.T) {
	watchDir := t.TempDir()
	scriptFile := filepath.Join(watchDir, "test.txt")
	if err := os.WriteFile(scriptFile, []byte("ロックファイルテスト"), 0o644); err != nil {
		t.Fatal(err)
	}

	var postCount int
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("/api/play", func(w http.ResponseWriter, r *http.Request) {
		postCount++
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"silent": false})
	})
	lockPath, _ := setupFakeViewer(t, mux)

	cmd := newCmdWithContext(t)
	registerCommonFlags(cmd)
	cmd.Flags().Bool("delete", false, "")
	cmd.Flags().Bool("queue", false, "")
	_ = cmd.ParseFlags([]string{})

	deps := &WatchDeps{
		Reader:        infra.NewFileReader(),
		ClientFactory: func(_ string) app.VoicevoxClient { return &stubVoicevoxClient{} },
		Player:        stubAudioPlayer{},
		Mover:         stubFileMover{},
		DirWatcherFactory: func(_ *slog.Logger) app.DirWatcher {
			return &oneFileDirWatcher{filePath: scriptFile}
		},
		LockPathResolver: func() (string, error) { return lockPath, nil },
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cmd.SetContext(ctx)

	done := make(chan error, 1)
	go func() { done <- runWatch(cmd, []string{watchDir}, deps) }()

	for i := 0; i < 100; i++ {
		if postCount > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	<-done

	if postCount == 0 {
		t.Error("expected at least 1 POST to /api/play via lockfile auto-detect, got 0")
	}
}
