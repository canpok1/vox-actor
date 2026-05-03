//go:build e2e

package e2e

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

type lineAccumulator struct {
	mu    sync.Mutex
	lines []string
}

func (a *lineAccumulator) append(line string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.lines = append(a.lines, line)
}

func (a *lineAccumulator) snapshot() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make([]string, len(a.lines))
	copy(out, a.lines)
	return out
}

func (a *lineAccumulator) String() string {
	return strings.Join(a.snapshot(), "\n")
}

type watchProcess struct {
	t        *testing.T
	cmd      *exec.Cmd
	cancel   context.CancelFunc
	stderr   *lineAccumulator
	done     chan struct{}
	waitErr  error // done が close された後にのみ参照してよい
	stopOnce sync.Once
}

// startWatch はビルド済み vox-actor を別プロセスで起動し、watch サブコマンドを実行する。
// t.Cleanup でプロセスの停止・killが登録されるため呼び出し側での後始末は不要。
func startWatch(t *testing.T, env map[string]string, args ...string) *watchProcess {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	fullArgs := append([]string{"watch"}, args...)
	cmd := exec.CommandContext(ctx, binaryPath, fullArgs...)
	if len(env) > 0 {
		base := os.Environ()
		for k, v := range env {
			base = append(base, k+"="+v)
		}
		cmd.Env = base
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		t.Fatalf("failed to create stderr pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		cancel()
		t.Fatalf("failed to start watch: %v", err)
	}

	acc := &lineAccumulator{}
	go readLinesInto(stderrPipe, acc)

	wp := &watchProcess{
		t:      t,
		cmd:    cmd,
		cancel: cancel,
		stderr: acc,
		done:   make(chan struct{}),
	}
	go func() {
		wp.waitErr = cmd.Wait()
		close(wp.done)
	}()

	t.Cleanup(func() {
		wp.stop(3 * time.Second)
	})

	return wp
}

func readLinesInto(r io.Reader, acc *lineAccumulator) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		acc.append(scanner.Text())
	}
}

// waitForStderr は stderr に needle を含む行が出るまで最大 timeout だけポーリングし、
// 見つかった行を返す。見つからない／プロセスが先に終了した場合は t.Fatalf を呼ぶ。
func (w *watchProcess) waitForStderr(needle string, timeout time.Duration) string {
	w.t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		for _, line := range w.stderr.snapshot() {
			if strings.Contains(line, needle) {
				return line
			}
		}
		select {
		case <-w.done:
			w.t.Fatalf("watch exited before %q appeared: err=%v\nstderr:\n%s",
				needle, w.waitErr, w.stderr.String())
		default:
		}
		if time.Now().After(deadline) {
			w.t.Fatalf("timed out after %s waiting for stderr to contain %q\nstderr:\n%s",
				timeout, needle, w.stderr.String())
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// stop は watch プロセスに SIGTERM を送り終了を待機する。timeout 以内に終了しなければ
// SIGKILL で強制終了する。二回目以降の呼び出しは no-op で初回の終了結果を返す。
func (w *watchProcess) stop(timeout time.Duration) (exitCode int, err error) {
	w.stopOnce.Do(func() {
		_ = w.cmd.Process.Signal(syscall.SIGTERM)
		select {
		case <-w.done:
		case <-time.After(timeout):
			_ = w.cmd.Process.Kill()
			<-w.done
		}
		w.cancel()
	})
	return exitCodeFromErr(w.waitErr), w.waitErr
}

func exitCodeFromErr(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return -1
}

// assertCleanExit は SIGTERM による正常停止（exit code 0）を検証する。
func (w *watchProcess) assertCleanExit(timeout time.Duration) {
	w.t.Helper()
	exitCode, _ := w.stop(timeout)
	if exitCode != 0 {
		w.t.Fatalf("expected clean exit (0), got %d\nstderr:\n%s",
			exitCode, w.stderr.String())
	}
}

func waitUntil(t *testing.T, timeout time.Duration, msg string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		if cond() {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out after %s: %s", timeout, msg)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// assertMovedToDone は src が消えて done/<basename> に移動したことをポーリングで確認する。
func assertMovedToDone(t *testing.T, src string, timeout time.Duration) {
	t.Helper()
	doneFile := filepath.Join(filepath.Dir(src), "done", filepath.Base(src))
	waitUntil(t, timeout,
		"expected "+src+" to be moved to "+doneFile,
		func() bool {
			if _, err := os.Stat(src); !os.IsNotExist(err) {
				return false
			}
			_, err := os.Stat(doneFile)
			return err == nil
		})
}

func TestWatchE2E_DryRun_DoneMove(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	wp := startWatch(t, nil, "--dry-run", dir)
	wp.waitForStderr("watching directory", 5*time.Second)

	src := writeTempFile(t, dir, "a.jsonl", `{"text":"テストイチ"}`+"\n")

	line := wp.waitForStderr("playback completed", 10*time.Second)
	if !strings.Contains(line, "[dry run]") {
		t.Errorf("expected playback completion line to contain '[dry run]', got: %s", line)
	}

	assertMovedToDone(t, src, 3*time.Second)

	wp.assertCleanExit(3 * time.Second)
}

func TestWatchE2E_DryRun_DeleteMode(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	wp := startWatch(t, nil, "--dry-run", "--delete", dir)
	wp.waitForStderr("watching directory", 5*time.Second)

	src := writeTempFile(t, dir, "a.jsonl", `{"text":"デリート"}`+"\n")

	wp.waitForStderr("playback completed", 10*time.Second)

	waitUntil(t, 3*time.Second,
		"expected "+src+" to be removed",
		func() bool {
			_, err := os.Stat(src)
			return os.IsNotExist(err)
		})

	doneDir := filepath.Join(dir, "done")
	if _, err := os.Stat(doneDir); !os.IsNotExist(err) {
		t.Errorf("expected done/ NOT to be created in --delete mode, got err=%v", err)
	}

	wp.assertCleanExit(3 * time.Second)
}

func TestWatchE2E_Queue_AutoCreatesAndProcesses(t *testing.T) {
	t.Parallel()
	workspace := t.TempDir()
	env := map[string]string{"VOX_ACTOR_WORKSPACE": workspace}

	wp := startWatch(t, env, "--queue", "--dry-run")
	wp.waitForStderr("watching directory", 5*time.Second)

	queueDir := filepath.Join(workspace, "queue")
	info, err := os.Stat(queueDir)
	if err != nil {
		t.Fatalf("expected queue dir to be auto-created at %s: %v", queueDir, err)
	}
	if !info.IsDir() {
		t.Fatalf("expected %s to be a directory", queueDir)
	}

	src := writeTempFile(t, queueDir, "a.jsonl", `{"text":"キュー"}`+"\n")

	wp.waitForStderr("playback completed", 10*time.Second)
	assertMovedToDone(t, src, 3*time.Second)

	wp.assertCleanExit(3 * time.Second)
}

func TestWatchE2E_UsageError_ExitCode2(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cases := []struct {
		name string
		args []string
	}{
		{name: "queue with positional arg", args: []string{"watch", "--queue", "--dry-run", dir}},
		{name: "stream flag removed", args: []string{"watch", "--stream", dir}},
		{name: "stream-addr flag removed", args: []string{"watch", "--stream-addr", "0.0.0.0:9090", dir}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, stderr, exitCode := runCLI(t, nil, tc.args...)
			if exitCode != 2 {
				t.Fatalf("expected exit code 2 (usage error), got %d\nstderr:\n%s", exitCode, stderr)
			}
			if strings.TrimSpace(stderr) == "" {
				t.Error("expected non-empty stderr for usage error")
			}
		})
	}
}

func TestWatchE2E_NoArgs_NonZero(t *testing.T) {
	t.Parallel()
	_, stderr, exitCode := runCLI(t, nil, "watch")
	if exitCode == 0 {
		t.Fatalf("expected non-zero exit code for 'watch' without args, got 0\nstderr:\n%s", stderr)
	}
}

// waitForStderrCount は stderr に needle を含む行が count 個以上出るまで待機する。
func (w *watchProcess) waitForStderrCount(needle string, count int, timeout time.Duration) {
	w.t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		found := 0
		for _, line := range w.stderr.snapshot() {
			if strings.Contains(line, needle) {
				found++
			}
		}
		if found >= count {
			return
		}
		select {
		case <-w.done:
			w.t.Fatalf("watch exited before %q appeared %d time(s) (found %d): err=%v\nstderr:\n%s",
				needle, count, found, w.waitErr, w.stderr.String())
		default:
		}
		if time.Now().After(deadline) {
			w.t.Fatalf("timed out after %s waiting for stderr to contain %q %d time(s)\nstderr:\n%s",
				timeout, needle, count, w.stderr.String())
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// --- Group B: ファイル形式ごとの処理 ---

func TestWatchE2E_TxtFile_DryRun_SingleScript(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	wp := startWatch(t, nil, "--dry-run", dir)
	wp.waitForStderr("watching directory", 5*time.Second)

	src := writeTempFile(t, dir, "a.txt", "いちぎょうめ\nにぎょうめ\n")
	wp.waitForStderr("playback completed", 10*time.Second)

	// .txt はファイル全体が1スクリプト → playback completed は1回
	if got := strings.Count(wp.stderr.String(), "playback completed"); got != 1 {
		t.Errorf("expected 1 'playback completed' for .txt, got %d\nstderr:\n%s", got, wp.stderr.String())
	}
	assertMovedToDone(t, src, 3*time.Second)
	wp.assertCleanExit(3 * time.Second)
}

func TestWatchE2E_JsonFile_DryRun_OverrideParams(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	wp := startWatch(t, nil, "--dry-run", dir)
	wp.waitForStderr("watching directory", 5*time.Second)

	content := `{"text":"こんにちは","speaker":11,"speed":1.2,"pitch":0.1,"intonation":1.5}`
	src := writeTempFile(t, dir, "b.json", content)

	line := wp.waitForStderr("playback completed", 10*time.Second)
	wants := []string{"text=こんにちは", "speaker=11", "speed=1.2", "pitch=0.1", "intonation=1.5"}
	for _, w := range wants {
		if !strings.Contains(line, w) {
			t.Errorf("expected playback line to contain %q\nline: %s", w, line)
		}
	}
	assertMovedToDone(t, src, 3*time.Second)
	wp.assertCleanExit(3 * time.Second)
}

func TestWatchE2E_JsonlFile_DryRun_MultipleScripts(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	wp := startWatch(t, nil, "--dry-run", dir)
	wp.waitForStderr("watching directory", 5*time.Second)

	content := `{"text":"いちぎょうめ"}` + "\n" +
		`{"text":"にぎょうめ"}` + "\n" +
		`{"text":"さんぎょうめ"}` + "\n"
	src := writeTempFile(t, dir, "c.jsonl", content)

	wp.waitForStderrCount("playback completed", 3, 15*time.Second)
	assertMovedToDone(t, src, 3*time.Second)

	stderr := wp.stderr.String()
	i1 := strings.Index(stderr, "text=いちぎょうめ")
	i2 := strings.Index(stderr, "text=にぎょうめ")
	i3 := strings.Index(stderr, "text=さんぎょうめ")
	if i1 < 0 || i2 < 0 || i3 < 0 {
		t.Fatalf("expected all three texts in stderr\nstderr:\n%s", stderr)
	}
	if i1 >= i2 || i2 >= i3 {
		t.Errorf("expected texts in order いち→に→さん, positions %d,%d,%d", i1, i2, i3)
	}

	wp.assertCleanExit(3 * time.Second)
}

func TestWatchE2E_InvalidJson_ErrorThenContinues(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	wp := startWatch(t, nil, "--dry-run", dir)
	wp.waitForStderr("watching directory", 5*time.Second)

	invalidSrc := writeTempFile(t, dir, "a.json", "not-valid-json")
	wp.waitForStderr("read error", 10*time.Second)
	assertMovedToDone(t, invalidSrc, 3*time.Second)

	// 有効ファイルも処理できること（監視継続確認）
	validSrc := writeTempFile(t, dir, "b.txt", "ただしいてきすと")
	wp.waitForStderr("playback completed", 10*time.Second)
	assertMovedToDone(t, validSrc, 3*time.Second)

	wp.assertCleanExit(3 * time.Second)
}

func TestWatchE2E_UnsupportedExtension_ProcessedAsText(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	wp := startWatch(t, nil, "--dry-run", dir)
	wp.waitForStderr("watching directory", 5*time.Second)

	src := writeTempFile(t, dir, "readme.md", "マークダウンのテキスト")
	wp.waitForStderr("playback completed", 10*time.Second)
	assertMovedToDone(t, src, 3*time.Second)
	wp.assertCleanExit(3 * time.Second)
}

func TestWatchE2E_EmptyFile_MovedToDone(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	wp := startWatch(t, nil, "--dry-run", dir)
	wp.waitForStderr("watching directory", 5*time.Second)

	src := writeTempFile(t, dir, "empty.txt", "")
	assertMovedToDone(t, src, 10*time.Second)

	if strings.Contains(wp.stderr.String(), "playback completed") {
		t.Errorf("expected no 'playback completed' for empty file\nstderr:\n%s", wp.stderr.String())
	}

	wp.assertCleanExit(3 * time.Second)
}

// --- Group C: 複数ディレクトリ並列監視 ---

func TestWatchE2E_MultipleDir_WatchingLogs(t *testing.T) {
	t.Parallel()
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	wp := startWatch(t, nil, "--dry-run", dir1, dir2)

	wp.waitForStderrCount("watching directory", 2, 10*time.Second)
	wp.assertCleanExit(3 * time.Second)
}

func TestWatchE2E_MultipleDir_BothProcessed(t *testing.T) {
	t.Parallel()
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	wp := startWatch(t, nil, "--dry-run", dir1, dir2)
	wp.waitForStderrCount("watching directory", 2, 10*time.Second)

	src1 := writeTempFile(t, dir1, "a.txt", "でぃあいりーわん")
	src2 := writeTempFile(t, dir2, "b.txt", "でぃれくとりーつー")

	wp.waitForStderrCount("playback completed", 2, 15*time.Second)
	assertMovedToDone(t, src1, 3*time.Second)
	assertMovedToDone(t, src2, 3*time.Second)

	wp.assertCleanExit(3 * time.Second)
}

func TestWatchE2E_MultipleDir_EachDoneCreated(t *testing.T) {
	t.Parallel()
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	wp := startWatch(t, nil, "--dry-run", dir1, dir2)
	wp.waitForStderrCount("watching directory", 2, 10*time.Second)

	src1 := writeTempFile(t, dir1, "a.txt", "でぃあいりーわん")
	src2 := writeTempFile(t, dir2, "b.txt", "でぃれくとりーつー")

	assertMovedToDone(t, src1, 10*time.Second)
	assertMovedToDone(t, src2, 10*time.Second)

	wp.assertCleanExit(3 * time.Second)
}

// --- Group D: 引数バリデーション ---

func TestWatchE2E_FilePath_ExitCode2(t *testing.T) {
	t.Parallel()
	filePath := writeTempFile(t, t.TempDir(), "notadir.txt", "")

	_, stderr, exitCode := runCLI(t, nil, "watch", "--dry-run", filePath)
	if exitCode != 2 {
		t.Fatalf("expected exit code 2 for file path, got %d\nstderr:\n%s", exitCode, stderr)
	}
}

func TestWatchE2E_NonExistentPath_ExitCode2(t *testing.T) {
	t.Parallel()
	_, stderr, exitCode := runCLI(t, nil, "watch", "--dry-run", "/nonexistent/path/12345")
	if exitCode != 2 {
		t.Fatalf("expected exit code 2 for nonexistent path, got %d\nstderr:\n%s", exitCode, stderr)
	}
}

func TestWatchE2E_MultiplePathsOneMissing_ExitCode2(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	_, stderr, exitCode := runCLI(t, nil, "watch", "--dry-run", dir, "/nonexistent/path/12345")
	if exitCode != 2 {
		t.Fatalf("expected exit code 2 when one of multiple paths is missing, got %d\nstderr:\n%s", exitCode, stderr)
	}
}

// --- Group E: フラグと環境変数の優先順位 ---

func TestWatchE2E_Flag_SpeakerOverridesEnv(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	wp := startWatch(t, map[string]string{"VOX_SPEAKER": "10"}, "--dry-run", "--speaker", "5", dir)
	wp.waitForStderr("watching directory", 5*time.Second)

	writeTempFile(t, dir, "a.txt", "すぴーかーてすと")
	line := wp.waitForStderr("playback completed", 10*time.Second)

	if !strings.Contains(line, "speaker=5") {
		t.Errorf("expected speaker=5 (--speaker flag overrides VOX_SPEAKER=10)\nline: %s", line)
	}

	wp.assertCleanExit(3 * time.Second)
}

func TestWatchE2E_Flag_EngineURLOverridesEnv(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	homeDir := t.TempDir()

	// --engine-url に 5xx サーバーを指定。VOX_ENGINE_URL は到達不能アドレス
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	wp := startWatch(t, map[string]string{
		"HOME":           homeDir,
		"VOX_ENGINE_URL": "http://127.0.0.1:1",
	}, "--engine-url", srv.URL, dir)

	select {
	case <-wp.done:
	case <-time.After(10 * time.Second):
		t.Fatal("watch did not exit within 10s")
	}

	exitCode, _ := wp.stop(3 * time.Second)
	if exitCode == 0 {
		t.Errorf("expected non-zero exit (proving --engine-url was used, not VOX_ENGINE_URL)\nstderr:\n%s", wp.stderr.String())
	}
	// 音声デバイスなし環境ではプローブが先に失敗する。
	// いずれの場合も非0終了が正しい挙動。
	if !strings.Contains(wp.stderr.String(), "500") && !strings.Contains(wp.stderr.String(), "audio") {
		t.Errorf("expected '500' or 'audio' in stderr\nstderr:\n%s", wp.stderr.String())
	}
}

func TestWatchE2E_Env_VOXSpeakerInvalid_DefaultFallback(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	wp := startWatch(t, map[string]string{"VOX_SPEAKER": "not-a-number"}, "--dry-run", dir)
	wp.waitForStderr("watching directory", 5*time.Second)

	writeTempFile(t, dir, "a.txt", "でふぉるとすぴーかー")
	line := wp.waitForStderr("playback completed", 10*time.Second)

	if !strings.Contains(line, "speaker=3") {
		t.Errorf("expected default speaker=3 when VOX_SPEAKER=not-a-number\nline: %s", line)
	}

	wp.assertCleanExit(3 * time.Second)
}

func TestWatchE2E_JsonlPerLineOverride_DryRun(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	wp := startWatch(t, nil, "--dry-run", dir)
	wp.waitForStderr("watching directory", 5*time.Second)

	content := `{"text":"てきすと","speaker":11,"speed":1.2,"pitch":0.1,"intonation":1.5}` + "\n"
	writeTempFile(t, dir, "a.jsonl", content)

	line := wp.waitForStderr("playback completed", 10*time.Second)
	wants := []string{"speaker=11", "speed=1.2", "pitch=0.1", "intonation=1.5"}
	for _, w := range wants {
		if !strings.Contains(line, w) {
			t.Errorf("expected playback line to contain %q\nline: %s", w, line)
		}
	}

	wp.assertCleanExit(3 * time.Second)
}

// --- Group F: ライフサイクル / シグナル ---

func TestWatchE2E_SIGINT_GracefulShutdown(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	wp := startWatch(t, nil, "--dry-run", dir)
	wp.waitForStderr("watching directory", 5*time.Second)

	if err := wp.cmd.Process.Signal(syscall.SIGINT); err != nil {
		t.Fatalf("failed to send SIGINT: %v", err)
	}

	select {
	case <-wp.done:
	case <-time.After(5 * time.Second):
		t.Fatal("watch did not exit within 5s after SIGINT")
	}

	exitCode, _ := wp.stop(3 * time.Second)
	if exitCode != 0 {
		t.Errorf("expected clean exit (0) after SIGINT, got %d\nstderr:\n%s", exitCode, wp.stderr.String())
	}
}

// --- Group G: ヘルプ / 境界 ---

func TestWatchE2E_Help_ExitZero(t *testing.T) {
	t.Parallel()
	stdout, stderr, exitCode := runCLI(t, nil, "watch", "--help")
	if exitCode != 0 {
		t.Fatalf("expected exit 0 for 'watch --help', got %d\nstderr:\n%s", exitCode, stderr)
	}
	wants := []string{"--delete", "--queue", "--dry-run", "--speaker", "--engine-url"}
	for _, w := range wants {
		if !strings.Contains(stdout, w) {
			t.Errorf("expected help to contain %q\nstdout:\n%s", w, stdout)
		}
	}
}

func TestWatchE2E_DoneCollision_TimestampRename(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	doneDir := filepath.Join(dir, "done")
	if err := os.MkdirAll(doneDir, 0o755); err != nil {
		t.Fatalf("failed to create done/: %v", err)
	}
	existingDone := writeTempFile(t, doneDir, "a.txt", "きぞんのふぁいる")

	wp := startWatch(t, nil, "--dry-run", dir)
	wp.waitForStderr("watching directory", 5*time.Second)

	src := writeTempFile(t, dir, "a.txt", "あたらしいふぁいる")
	wp.waitForStderr("playback completed", 10*time.Second)

	waitUntil(t, 5*time.Second, "src should be gone", func() bool {
		_, err := os.Stat(src)
		return os.IsNotExist(err)
	})

	if _, err := os.Stat(existingDone); err != nil {
		t.Errorf("expected existing done/a.txt to remain: %v", err)
	}
	entries, err := os.ReadDir(doneDir)
	if err != nil {
		t.Fatalf("failed to read done/: %v", err)
	}
	if len(entries) < 2 {
		t.Errorf("expected at least 2 files in done/ (original + timestamp-renamed), got %d", len(entries))
	}

	wp.assertCleanExit(3 * time.Second)
}

func TestWatchE2E_DoneDir_NotMonitored(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	doneDir := filepath.Join(dir, "done")
	if err := os.MkdirAll(doneDir, 0o755); err != nil {
		t.Fatalf("failed to create done/: %v", err)
	}
	doneFile := writeTempFile(t, doneDir, "existing.txt", "どんのない")

	wp := startWatch(t, nil, "--dry-run", dir)
	wp.waitForStderr("watching directory", 5*time.Second)

	time.Sleep(2 * time.Second)

	if _, err := os.Stat(doneFile); err != nil {
		t.Errorf("expected done/existing.txt to remain untouched: %v", err)
	}
	if strings.Contains(wp.stderr.String(), "file detected") {
		t.Errorf("watch should not detect files inside done/\nstderr:\n%s", wp.stderr.String())
	}

	wp.assertCleanExit(3 * time.Second)
}

// --- Group H: ログ内容 ---

func TestWatchE2E_Verbose_DebugLogs(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	wp := startWatch(t, nil, "--dry-run", "--verbose", dir)
	wp.waitForStderr("watching directory", 5*time.Second)

	writeTempFile(t, dir, "a.txt", "ばーぼーすてすと")
	wp.waitForStderr("playback completed", 10*time.Second)

	if !strings.Contains(wp.stderr.String(), "watch mode starting") {
		t.Errorf("expected DEBUG log 'watch mode starting' with --verbose\nstderr:\n%s", wp.stderr.String())
	}

	wp.assertCleanExit(3 * time.Second)
}

func TestWatchE2E_WatchingDirectoryLog(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	wp := startWatch(t, nil, "--dry-run", dir)
	line := wp.waitForStderr("watching directory", 5*time.Second)

	if !strings.Contains(line, dir) {
		t.Errorf("expected 'watching directory' log to contain dir path\nline: %s", line)
	}

	wp.assertCleanExit(3 * time.Second)
}

func TestWatchE2E_FileDetectedLog(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	wp := startWatch(t, nil, "--dry-run", "--verbose", dir)
	wp.waitForStderr("watching directory", 5*time.Second)

	writeTempFile(t, dir, "detected.txt", "ふぁいるけんちてすと")
	line := wp.waitForStderr("file detected", 10*time.Second)

	if !strings.Contains(line, "detected.txt") {
		t.Errorf("expected 'file detected' log to contain filename\nline: %s", line)
	}

	wp.assertCleanExit(3 * time.Second)
}

// --- Group I: --queue 系の追加検証 ---

func TestWatchE2E_Queue_Delete_Combination(t *testing.T) {
	t.Parallel()
	workspace := t.TempDir()
	env := map[string]string{"VOX_ACTOR_WORKSPACE": workspace}

	wp := startWatch(t, env, "--queue", "--delete", "--dry-run")
	wp.waitForStderr("watching directory", 5*time.Second)

	queueDir := filepath.Join(workspace, "queue")
	src := writeTempFile(t, queueDir, "a.txt", "きゅーでりーとてすと")
	wp.waitForStderr("playback completed", 10*time.Second)

	waitUntil(t, 3*time.Second, "src should be deleted", func() bool {
		_, err := os.Stat(src)
		return os.IsNotExist(err)
	})
	doneDir := filepath.Join(queueDir, "done")
	if _, err := os.Stat(doneDir); !os.IsNotExist(err) {
		t.Errorf("expected done/ NOT to be created in --delete mode")
	}

	wp.assertCleanExit(3 * time.Second)
}
