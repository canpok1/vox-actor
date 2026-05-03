//go:build e2e

package e2e

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// scriptJSON は script append の .json / .jsonl 出力をパースするための構造体。
type scriptJSON struct {
	Text            string   `json:"text"`
	Speaker         *int     `json:"speaker"`
	SpeedScale      *float64 `json:"speed"`
	PitchScale      *float64 `json:"pitch"`
	IntonationScale *float64 `json:"intonation"`
}

// readScriptJSON は path のファイルの最初の行を JSON としてパースして返す。
func readScriptJSON(t *testing.T, path string) scriptJSON {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file %s: %v", path, err)
	}
	line := strings.SplitN(strings.TrimRight(string(data), "\n"), "\n", 2)[0]
	var s scriptJSON
	if err := json.Unmarshal([]byte(line), &s); err != nil {
		t.Fatalf("failed to parse JSON from %s: %v\nline: %s", path, err, line)
	}
	return s
}

// collectSayJSONFiles は dir 配下の *_say.json ファイル名一覧を返す。
func collectSayJSONFiles(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("failed to read dir %s: %v", dir, err)
	}
	var files []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), "_say.json") {
			files = append(files, e.Name())
		}
	}
	return files
}

// ─────────────────────────────────────────
// A. 基本的な書き出し挙動
// ─────────────────────────────────────────

func TestScriptAppendE2E_Basic_JsonlCreate(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dest := filepath.Join(dir, "out.jsonl")

	_, stderr, exitCode := runCLI(t, nil, "script", "append", dest, "こんにちは")
	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d\nstderr:\n%s", exitCode, stderr)
	}

	s := readScriptJSON(t, dest)
	if s.Text != "こんにちは" {
		t.Errorf("expected text=%q, got %q", "こんにちは", s.Text)
	}
}

func TestScriptAppendE2E_Basic_JsonCreate(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dest := filepath.Join(dir, "out.json")

	_, stderr, exitCode := runCLI(t, nil, "script", "append", dest, "テスト")
	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d\nstderr:\n%s", exitCode, stderr)
	}

	s := readScriptJSON(t, dest)
	if s.Text != "テスト" {
		t.Errorf("expected text=%q, got %q", "テスト", s.Text)
	}
}

func TestScriptAppendE2E_Basic_TxtCreate(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dest := filepath.Join(dir, "out.txt")

	_, stderr, exitCode := runCLI(t, nil, "script", "append", dest, "テキスト本文")
	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d\nstderr:\n%s", exitCode, stderr)
	}

	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if string(data) != "テキスト本文" {
		t.Errorf("expected file content %q, got %q", "テキスト本文", string(data))
	}
}

func TestScriptAppendE2E_Basic_Append(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dest := filepath.Join(dir, "out.jsonl")

	for _, text := range []string{"1行目", "2行目"} {
		_, stderr, exitCode := runCLI(t, nil, "script", "append", dest, text)
		if exitCode != 0 {
			t.Fatalf("expected exit 0 for %q, got %d\nstderr:\n%s", text, exitCode, stderr)
		}
	}

	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d\ncontent:\n%s", len(lines), string(data))
	}

	var s1, s2 scriptJSON
	if err := json.Unmarshal([]byte(lines[0]), &s1); err != nil {
		t.Fatalf("parse line 1: %v", err)
	}
	if err := json.Unmarshal([]byte(lines[1]), &s2); err != nil {
		t.Fatalf("parse line 2: %v", err)
	}
	if s1.Text != "1行目" {
		t.Errorf("line1: expected text=%q, got %q", "1行目", s1.Text)
	}
	if s2.Text != "2行目" {
		t.Errorf("line2: expected text=%q, got %q", "2行目", s2.Text)
	}
}

func TestScriptAppendE2E_Basic_NoExtension(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dest := filepath.Join(dir, "noext")

	_, stderr, exitCode := runCLI(t, nil, "script", "append", dest, "テスト")
	if exitCode == 0 {
		t.Fatalf("expected non-zero exit, got 0")
	}
	if !strings.Contains(stderr, "no extension") {
		t.Errorf("expected stderr to contain 'no extension'\nstderr:\n%s", stderr)
	}
}

func TestScriptAppendE2E_Basic_UnsupportedExtension(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dest := filepath.Join(dir, "out.md")

	_, stderr, exitCode := runCLI(t, nil, "script", "append", dest, "テスト")
	if exitCode == 0 {
		t.Fatalf("expected non-zero exit, got 0")
	}
	if !strings.Contains(stderr, "unsupported") {
		t.Errorf("expected stderr to contain 'unsupported'\nstderr:\n%s", stderr)
	}
}

func TestScriptAppendE2E_Basic_NoParentDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dest := filepath.Join(dir, "nonexistent", "out.jsonl")

	_, _, exitCode := runCLI(t, nil, "script", "append", dest, "テスト")
	if exitCode == 0 {
		t.Fatalf("expected non-zero exit when parent dir missing, got 0")
	}
}

// ─────────────────────────────────────────
// B. ボイスパラメータの反映
// ─────────────────────────────────────────

func TestScriptAppendE2E_VoiceParams_Reflected(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dest := filepath.Join(dir, "out.jsonl")

	_, stderr, exitCode := runCLI(t, nil,
		"script", "append",
		"--speaker", "5",
		"--speed", "1.2",
		"--pitch", "0.05",
		"--intonation", "1.5",
		dest, "パラメータテスト",
	)
	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d\nstderr:\n%s", exitCode, stderr)
	}

	s := readScriptJSON(t, dest)
	if s.Speaker == nil || *s.Speaker != 5 {
		t.Errorf("expected speaker=5, got %v", s.Speaker)
	}
	if s.SpeedScale == nil || *s.SpeedScale != 1.2 {
		t.Errorf("expected speed=1.2, got %v", s.SpeedScale)
	}
	if s.PitchScale == nil || *s.PitchScale != 0.05 {
		t.Errorf("expected pitch=0.05, got %v", s.PitchScale)
	}
	if s.IntonationScale == nil || *s.IntonationScale != 1.5 {
		t.Errorf("expected intonation=1.5, got %v", s.IntonationScale)
	}
}

func TestScriptAppendE2E_VoiceParams_Omitempty(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dest := filepath.Join(dir, "out.jsonl")

	_, stderr, exitCode := runCLI(t, nil, "script", "append", dest, "省略テスト")
	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d\nstderr:\n%s", exitCode, stderr)
	}

	s := readScriptJSON(t, dest)
	if s.Speaker != nil {
		t.Errorf("expected speaker to be omitted, got %v", *s.Speaker)
	}
	if s.SpeedScale != nil {
		t.Errorf("expected speed to be omitted, got %v", *s.SpeedScale)
	}
	if s.PitchScale != nil {
		t.Errorf("expected pitch to be omitted, got %v", *s.PitchScale)
	}
	if s.IntonationScale != nil {
		t.Errorf("expected intonation to be omitted, got %v", *s.IntonationScale)
	}
}

func TestScriptAppendE2E_VoiceParams_TxtWarn(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dest := filepath.Join(dir, "out.txt")

	_, stderr, exitCode := runCLI(t, nil,
		"script", "append",
		"--speaker", "3",
		dest, "テキスト警告",
	)
	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d\nstderr:\n%s", exitCode, stderr)
	}
	if !strings.Contains(stderr, "text-format output cannot store voice parameters") {
		t.Errorf("expected WARN log in stderr\nstderr:\n%s", stderr)
	}
}

// ─────────────────────────────────────────
// C. ディレクトリ書き出し
// ─────────────────────────────────────────

func TestScriptAppendE2E_Dir_TrailingSlash(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	destDir := dir + string(os.PathSeparator)

	_, stderr, exitCode := runCLI(t, nil, "script", "append", destDir, "ディレクトリ指定")
	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d\nstderr:\n%s", exitCode, stderr)
	}

	jsonFiles := collectSayJSONFiles(t, dir)
	if len(jsonFiles) != 1 {
		t.Fatalf("expected 1 *_say.json file, got %d: %v", len(jsonFiles), jsonFiles)
	}

	s := readScriptJSON(t, filepath.Join(dir, jsonFiles[0]))
	if s.Text != "ディレクトリ指定" {
		t.Errorf("expected text=%q, got %q", "ディレクトリ指定", s.Text)
	}
}

func TestScriptAppendE2E_Dir_ExistingDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	_, stderr, exitCode := runCLI(t, nil, "script", "append", dir, "既存ディレクトリ")
	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d\nstderr:\n%s", exitCode, stderr)
	}

	jsonFiles := collectSayJSONFiles(t, dir)
	if len(jsonFiles) != 1 {
		t.Fatalf("expected 1 *_say.json file, got %d: %v", len(jsonFiles), jsonFiles)
	}
}

func TestScriptAppendE2E_Dir_CreateNew(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	newDir := filepath.Join(dir, "newsubdir")

	_, stderr, exitCode := runCLI(t, nil, "script", "append", newDir+string(os.PathSeparator), "新規ディレクトリ")
	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d\nstderr:\n%s", exitCode, stderr)
	}

	info, err := os.Stat(newDir)
	if err != nil {
		t.Fatalf("expected dir to be created: %v", err)
	}
	if !info.IsDir() {
		t.Errorf("expected %s to be a directory", newDir)
	}

	jsonFiles := collectSayJSONFiles(t, newDir)
	if len(jsonFiles) != 1 {
		t.Fatalf("expected 1 *_say.json file, got %d", len(jsonFiles))
	}
}

// ─────────────────────────────────────────
// D. 引数バリデーション
// ─────────────────────────────────────────

func TestScriptAppendE2E_ArgValidation_ExitCode2(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dest := filepath.Join(dir, "out.jsonl")

	cases := []struct {
		name string
		args []string
	}{
		{name: "no args", args: []string{"script", "append"}},
		{name: "file only", args: []string{"script", "append", dest}},
		{name: "too many args", args: []string{"script", "append", dest, "a", "b"}},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, stderr, exitCode := runCLI(t, nil, tc.args...)
			if exitCode != 2 {
				t.Fatalf("expected exit 2, got %d\nstderr:\n%s", exitCode, stderr)
			}
		})
	}
}

// ─────────────────────────────────────────
// E. ヘルプ / 親コマンド / 環境変数
// ─────────────────────────────────────────

func TestScriptE2E_Help_ScriptAlone(t *testing.T) {
	t.Parallel()

	stdout, _, exitCode := runCLI(t, nil, "script")
	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d", exitCode)
	}
	if !strings.Contains(stdout, "append") {
		t.Errorf("expected help output to contain 'append'\nstdout:\n%s", stdout)
	}
}

func TestScriptAppendE2E_Help_Flags(t *testing.T) {
	t.Parallel()

	stdout, stderr, exitCode := runCLI(t, nil, "script", "append", "--help")
	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d\nstderr:\n%s", exitCode, stderr)
	}

	for _, flag := range []string{"--speaker", "--speed", "--pitch", "--intonation", "--verbose"} {
		if !strings.Contains(stdout, flag) {
			t.Errorf("expected help to contain %q\nstdout:\n%s", flag, stdout)
		}
	}
	for _, flag := range []string{"--engine-url", "--dry-run"} {
		if strings.Contains(stdout, flag) {
			t.Errorf("expected help NOT to contain %q\nstdout:\n%s", flag, stdout)
		}
	}
}

func TestScriptAppendE2E_EnvVar_VOXSpeaker(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dest := filepath.Join(dir, "out.jsonl")

	env := map[string]string{"VOX_SPEAKER": "11"}
	_, stderr, exitCode := runCLI(t, env, "script", "append", dest, "環境変数")
	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d\nstderr:\n%s", exitCode, stderr)
	}

	// VOX_SPEAKER はデフォルト値を設定するが、cobra の Changed() は false のため omitempty で省略される。
	s := readScriptJSON(t, dest)
	if s.Speaker != nil {
		t.Errorf("expected speaker to be absent when only env var set (not --speaker flag), got %v", *s.Speaker)
	}
}

// ─────────────────────────────────────────
// F. ログ内容
// ─────────────────────────────────────────

func TestScriptAppendE2E_Log_OutputCompleted(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dest := filepath.Join(dir, "out.jsonl")

	_, stderr, exitCode := runCLI(t, nil, "script", "append", dest, "ログテスト")
	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d\nstderr:\n%s", exitCode, stderr)
	}

	line := extractLineContaining(t, stderr, "output completed")
	if !strings.Contains(line, "path=") {
		t.Errorf("expected 'output completed' line to contain 'path='\nline: %s", line)
	}
}

func TestScriptAppendE2E_Log_Verbose(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dest := filepath.Join(dir, "out.jsonl")

	_, stderr, exitCode := runCLI(t, nil, "script", "append", "--verbose", dest, "詳細ログ")
	if exitCode != 0 {
		t.Fatalf("expected exit 0 with --verbose, got %d\nstderr:\n%s", exitCode, stderr)
	}
	if !strings.Contains(stderr, "output completed") {
		t.Errorf("expected 'output completed' log with --verbose\nstderr:\n%s", stderr)
	}
}

// ─────────────────────────────────────────
// G. 文字エンコーディング / 特殊文字
// ─────────────────────────────────────────

func TestScriptAppendE2E_Encoding_Multibyte(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dest := filepath.Join(dir, "out.jsonl")
	text := "日本語テスト🎉"

	_, stderr, exitCode := runCLI(t, nil, "script", "append", dest, text)
	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d\nstderr:\n%s", exitCode, stderr)
	}

	s := readScriptJSON(t, dest)
	if s.Text != text {
		t.Errorf("expected text=%q, got %q", text, s.Text)
	}
}

func TestScriptAppendE2E_Encoding_SpecialChars(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dest := filepath.Join(dir, "out.jsonl")
	text := `"quoted" and \backslash`

	_, stderr, exitCode := runCLI(t, nil, "script", "append", dest, text)
	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d\nstderr:\n%s", exitCode, stderr)
	}

	s := readScriptJSON(t, dest)
	if s.Text != text {
		t.Errorf("expected text=%q, got %q", text, s.Text)
	}
}

func TestScriptAppendE2E_Encoding_EmptyText(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dest := filepath.Join(dir, "out.jsonl")

	_, stderr, exitCode := runCLI(t, nil, "script", "append", dest, "")
	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d\nstderr:\n%s", exitCode, stderr)
	}

	s := readScriptJSON(t, dest)
	if s.Text != "" {
		t.Errorf("expected empty text, got %q", s.Text)
	}
}
