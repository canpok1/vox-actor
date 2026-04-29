//go:build e2e

package e2e

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestActE2E_TxtFile_DryRun(t *testing.T) {
	dir := t.TempDir()
	path := writeTempFile(t, dir, "script.txt", "一行目\n二行目\n三行目\n")

	_, stderr, exitCode := runCLI(t, nil, "act", "--dry-run", path)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d\nstderr:\n%s", exitCode, stderr)
	}

	// .txt は改行を含めてもファイル全体で1 Scriptとして扱う仕様のため playback completed は1回
	if got := strings.Count(stderr, "playback completed"); got != 1 {
		t.Errorf("expected 1 'playback completed', got %d\nstderr:\n%s", got, stderr)
	}
	if !strings.Contains(stderr, "[dry run] playback completed") {
		t.Errorf("expected stderr to contain '[dry run] playback completed'\nstderr:\n%s", stderr)
	}
}

func TestActE2E_JsonFile_DryRun_OverrideParams(t *testing.T) {
	dir := t.TempDir()
	content := `{"text":"こんにちは","speaker":11,"speedScale":1.2,"pitchScale":0.1,"intonationScale":1.5}`
	path := writeTempFile(t, dir, "script.json", content)

	_, stderr, exitCode := runCLI(t, nil, "act", "--dry-run", path)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d\nstderr:\n%s", exitCode, stderr)
	}

	playbackLine := extractLineContaining(t, stderr, "[dry run] playback completed")
	wants := []string{
		"text=こんにちは",
		"speaker=11",
		"speed=1.2",
		"pitch=0.1",
		"intonation=1.5",
	}
	for _, w := range wants {
		if !strings.Contains(playbackLine, w) {
			t.Errorf("expected playback line to contain %q\nline: %s", w, playbackLine)
		}
	}
}

func TestActE2E_JsonlFile_DryRun_MultipleScripts(t *testing.T) {
	dir := t.TempDir()
	content := `{"text":"いちぎょうめ"}
{"text":"にぎょうめ"}
{"text":"さんぎょうめ"}
`
	path := writeTempFile(t, dir, "script.jsonl", content)

	_, stderr, exitCode := runCLI(t, nil, "act", "--dry-run", path)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d\nstderr:\n%s", exitCode, stderr)
	}

	if got := strings.Count(stderr, "playback completed"); got != 3 {
		t.Errorf("expected 3 'playback completed', got %d\nstderr:\n%s", got, stderr)
	}

	i1 := strings.Index(stderr, "text=いちぎょうめ")
	i2 := strings.Index(stderr, "text=にぎょうめ")
	i3 := strings.Index(stderr, "text=さんぎょうめ")
	if i1 < 0 || i2 < 0 || i3 < 0 {
		t.Fatalf("expected all three texts in stderr\nstderr:\n%s", stderr)
	}
	if i1 >= i2 || i2 >= i3 {
		t.Errorf("expected texts in order いち→に→さん, got positions %d,%d,%d", i1, i2, i3)
	}
}

func TestActE2E_Directory_DryRun_LexOrder(t *testing.T) {
	dir := t.TempDir()
	writeTempFile(t, dir, "a.txt", "エーのテキスト")
	writeTempFile(t, dir, "b.json", `{"text":"ビーのテキスト"}`)
	writeTempFile(t, dir, "c.jsonl", `{"text":"シー1"}`+"\n"+`{"text":"シー2"}`+"\n")

	_, stderr, exitCode := runCLI(t, nil, "act", "--dry-run", dir)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d\nstderr:\n%s", exitCode, stderr)
	}

	// a.txt=1, b.json=1, c.jsonl=2
	if got := strings.Count(stderr, "playback completed"); got != 4 {
		t.Errorf("expected 4 'playback completed', got %d\nstderr:\n%s", got, stderr)
	}

	// 辞書順（a.txt → b.json → c.jsonl の1行目 → 2行目）で出現することを確認
	positions := []int{
		strings.Index(stderr, "text=エーのテキスト"),
		strings.Index(stderr, "text=ビーのテキスト"),
		strings.Index(stderr, "text=シー1"),
		strings.Index(stderr, "text=シー2"),
	}
	for i, p := range positions {
		if p < 0 {
			t.Fatalf("expected position %d to be found\nstderr:\n%s", i, stderr)
		}
	}
	for i := 1; i < len(positions); i++ {
		if positions[i-1] >= positions[i] {
			t.Errorf("expected positions[%d]<positions[%d], got %d>=%d", i-1, i, positions[i-1], positions[i])
		}
	}
}

func TestActE2E_NonexistentPath_NonZeroWithStderr(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "does-not-exist.txt")

	_, stderr, exitCode := runCLI(t, nil, "act", "--dry-run", missing)
	if exitCode == 0 {
		t.Fatalf("expected non-zero exit code for nonexistent path, got 0\nstderr:\n%s", stderr)
	}
	if strings.TrimSpace(stderr) == "" {
		t.Error("expected non-empty stderr for nonexistent path")
	}
}

func TestActE2E_BrokenJSON_NonZero(t *testing.T) {
	dir := t.TempDir()
	path := writeTempFile(t, dir, "broken.json", "{")

	_, stderr, exitCode := runCLI(t, nil, "act", "--dry-run", path)
	if exitCode == 0 {
		t.Fatalf("expected non-zero exit code for broken JSON, got 0\nstderr:\n%s", stderr)
	}
	if strings.TrimSpace(stderr) == "" {
		t.Error("expected non-empty stderr for broken JSON")
	}
}

func TestActE2E_NoArgs_ExitCode2(t *testing.T) {
	_, stderr, exitCode := runCLI(t, nil, "act")
	if exitCode != 2 {
		t.Fatalf("expected exit code 2, got %d\nstderr:\n%s", exitCode, stderr)
	}
}

func TestActE2E_TxtFile_FlagsReflectedInLog(t *testing.T) {
	dir := t.TempDir()
	path := writeTempFile(t, dir, "flag.txt", "フラグテスト")

	_, stderr, exitCode := runCLI(t, nil,
		"act", "--dry-run",
		"--speaker", "8",
		"--speed", "1.2",
		path,
	)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d\nstderr:\n%s", exitCode, stderr)
	}

	playbackLine := extractLineContaining(t, stderr, "[dry run] playback completed")
	wants := []string{
		"text=フラグテスト",
		"speaker=8",
		"speed=1.2",
	}
	for _, w := range wants {
		if !strings.Contains(playbackLine, w) {
			t.Errorf("expected playback line to contain %q\nline: %s", w, playbackLine)
		}
	}
}

func TestActE2E_JsonFile_OmittedParams_UsesCLIFlagDefaults(t *testing.T) {
	dir := t.TempDir()
	path := writeTempFile(t, dir, "only_text.json", `{"text":"きほん"}`)

	_, stderr, exitCode := runCLI(t, nil,
		"act", "--dry-run",
		"--speaker", "5",
		"--speed", "1.3",
		path,
	)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d\nstderr:\n%s", exitCode, stderr)
	}

	playbackLine := extractLineContaining(t, stderr, "[dry run] playback completed")
	wants := []string{
		"text=きほん",
		"speaker=5",
		"speed=1.3",
	}
	for _, w := range wants {
		if !strings.Contains(playbackLine, w) {
			t.Errorf("expected playback line to contain %q\nline: %s", w, playbackLine)
		}
	}
}

func TestActE2E_WatchFlag_ExitCode2(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	_, stderr, exitCode := runCLI(t, nil, "act", "--watch", dir)
	if exitCode != 2 {
		t.Fatalf("expected exit code 2 for removed --watch flag, got %d\nstderr:\n%s", exitCode, stderr)
	}
}

func TestActE2E_WatchDeleteFlag_ExitCode2(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	_, stderr, exitCode := runCLI(t, nil, "act", "--watch-delete", dir)
	if exitCode != 2 {
		t.Fatalf("expected exit code 2 for removed --watch-delete flag, got %d\nstderr:\n%s", exitCode, stderr)
	}
}

func TestActE2E_Help_ExitZeroWithAllFlags(t *testing.T) {
	t.Parallel()

	stdout, _, exitCode := runCLI(t, nil, "act", "--help")

	if exitCode != 0 {
		t.Fatalf("expected exit 0 for act --help, got %d", exitCode)
	}
	for _, flag := range []string{"--engine-url", "--speaker", "--speed", "--pitch", "--intonation", "--verbose", "--dry-run"} {
		if !strings.Contains(stdout, flag) {
			t.Errorf("expected act --help to contain %q\nstdout:\n%s", flag, stdout)
		}
	}
}

func TestActE2E_EmptyFile_Txt_ExitsZero(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := writeTempFile(t, dir, "empty.txt", "")

	_, stderr, exitCode := runCLI(t, nil, "act", "--dry-run", path)
	if exitCode != 0 {
		t.Fatalf("expected exit 0 for empty .txt, got %d\nstderr:\n%s", exitCode, stderr)
	}
}

func TestActE2E_EmptyFile_Jsonl_ExitsZero(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := writeTempFile(t, dir, "empty.jsonl", "")

	_, stderr, exitCode := runCLI(t, nil, "act", "--dry-run", path)
	if exitCode != 0 {
		t.Fatalf("expected exit 0 for empty .jsonl, got %d\nstderr:\n%s", exitCode, stderr)
	}
}

func TestActE2E_EmptyJson_NoTextField_NonZero(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := writeTempFile(t, dir, "empty.json", "{}")

	_, stderr, exitCode := runCLI(t, nil, "act", "--dry-run", path)
	if exitCode == 0 {
		t.Fatalf("expected non-zero exit for JSON missing 'text' field, got 0\nstderr:\n%s", stderr)
	}
	if strings.TrimSpace(stderr) == "" {
		t.Error("expected non-empty stderr for missing 'text' field")
	}
}

func TestActE2E_UnsupportedExtension_ExitsZero(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := writeTempFile(t, dir, "note.md", "# 見出し\n本文テキスト")

	_, stderr, exitCode := runCLI(t, nil, "act", "--dry-run", path)
	if exitCode != 0 {
		t.Fatalf("expected exit 0 for .md file (treated as plain text), got %d\nstderr:\n%s", exitCode, stderr)
	}
}

func TestActE2E_JsonlFile_PerLineParams_Reflected(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	content := `{"text":"いちぎょうめ","speaker":11,"speedScale":1.2,"pitchScale":0.05,"intonationScale":1.3}
{"text":"にぎょうめ","speaker":7}
`
	path := writeTempFile(t, dir, "params.jsonl", content)

	_, stderr, exitCode := runCLI(t, nil, "act", "--dry-run", path)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d\nstderr:\n%s", exitCode, stderr)
	}

	lines := strings.Split(stderr, "\n")
	var playbackLines []string
	for _, l := range lines {
		if strings.Contains(l, "playback completed") {
			playbackLines = append(playbackLines, l)
		}
	}
	if len(playbackLines) != 2 {
		t.Fatalf("expected 2 playback lines, got %d\nstderr:\n%s", len(playbackLines), stderr)
	}
	for _, want := range []string{"speaker=11", "speed=1.2", "pitch=0.05", "intonation=1.3"} {
		if !strings.Contains(playbackLines[0], want) {
			t.Errorf("expected first line to contain %q\nline: %s", want, playbackLines[0])
		}
	}
	if !strings.Contains(playbackLines[1], "speaker=7") {
		t.Errorf("expected second line to contain speaker=7\nline: %s", playbackLines[1])
	}
}

func TestActE2E_JsonlFile_EmptyLines_Skipped(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	content := `{"text":"いちぎょうめ"}

{"text":"さんぎょうめ"}
`
	path := writeTempFile(t, dir, "with_empty.jsonl", content)

	_, stderr, exitCode := runCLI(t, nil, "act", "--dry-run", path)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d\nstderr:\n%s", exitCode, stderr)
	}
	if got := strings.Count(stderr, "playback completed"); got != 2 {
		t.Errorf("expected 2 'playback completed' (empty line skipped), got %d\nstderr:\n%s", got, stderr)
	}
}

func TestActE2E_Verbose_ShowsDebugLogs(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := writeTempFile(t, dir, "script.txt", "デバッグテスト")

	_, stderr, exitCode := runCLI(t, nil, "act", "--dry-run", "--verbose", path)
	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d\nstderr:\n%s", exitCode, stderr)
	}
	debugLine := extractLineContaining(t, stderr, "act starting")
	if !strings.Contains(debugLine, "path=") {
		t.Errorf("expected debug line to contain 'path='\nline: %s", debugLine)
	}
}

func TestActE2E_Dir_ScriptsLoadedLog(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeTempFile(t, dir, "a.txt", "エー")
	writeTempFile(t, dir, "b.txt", "ビー")

	_, stderr, exitCode := runCLI(t, nil, "act", "--dry-run", dir)
	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d\nstderr:\n%s", exitCode, stderr)
	}
	if !strings.Contains(stderr, "scripts loaded") {
		t.Errorf("expected 'scripts loaded' log in stderr\nstderr:\n%s", stderr)
	}
}
