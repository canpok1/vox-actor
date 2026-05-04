//go:build e2e

package e2e

import (
	"strings"
	"testing"
)

func TestSayE2E_DryRun_Basic(t *testing.T) {
	_, stderr, exitCode := runCLI(t, nil, "say", "--dry-run", "こんにちは")

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d\nstderr:\n%s", exitCode, stderr)
	}

	wants := []string{
		"[dry run] synthesis completed",
		"[dry run] playback completed",
		"text=こんにちは",
	}
	for _, w := range wants {
		if !strings.Contains(stderr, w) {
			t.Errorf("expected stderr to contain %q\nstderr:\n%s", w, stderr)
		}
	}
}

func TestSayE2E_DryRun_Flags(t *testing.T) {
	_, stderr, exitCode := runCLI(t, nil,
		"say", "--dry-run",
		"--speaker", "8",
		"--speed", "1.2",
		"--pitch", "0.1",
		"--intonation", "1.5",
		"テスト",
	)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d\nstderr:\n%s", exitCode, stderr)
	}

	playbackLine := extractLineContaining(t, stderr, "[dry run] playback completed")
	wants := []string{
		"text=テスト",
		"speaker=8",
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

func TestSayE2E_DryRun_EnvVarVOXSpeaker(t *testing.T) {
	env := map[string]string{"VOX_SPEAKER": "11"}
	_, stderr, exitCode := runCLI(t, env, "say", "--dry-run", "環境変数")

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d\nstderr:\n%s", exitCode, stderr)
	}

	playbackLine := extractLineContaining(t, stderr, "[dry run] playback completed")
	if !strings.Contains(playbackLine, "speaker=11") {
		t.Errorf("expected playback line to contain 'speaker=11'\nline: %s", playbackLine)
	}
}

// TestSayE2E_DryRun_EnvVarVOXEngineURL_NotFailing はdry-run中はエンジンへ実通信しないため、
// 到達不能URLを指定しても異常終了しないことを検証する。
func TestSayE2E_DryRun_EnvVarVOXEngineURL_NotFailing(t *testing.T) {
	env := map[string]string{"VOX_ENGINE_URL": "http://example.invalid:50021"}
	_, stderr, exitCode := runCLI(t, env, "say", "--dry-run", "エンジンURL")

	if exitCode != 0 {
		t.Fatalf("expected exit code 0 with VOX_ENGINE_URL in dry-run, got %d\nstderr:\n%s", exitCode, stderr)
	}

	if !strings.Contains(stderr, "[dry run] playback completed") {
		t.Errorf("expected stderr to contain '[dry run] playback completed'\nstderr:\n%s", stderr)
	}
}

func TestSayE2E_ArgValidation_ExitCode2(t *testing.T) {
	cases := []struct {
		name string
		args []string
	}{
		{name: "no args", args: []string{"say"}},
		{name: "too many args", args: []string{"say", "a", "b"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, stderr, exitCode := runCLI(t, nil, tc.args...)
			if exitCode != 2 {
				t.Fatalf("expected exit code 2, got %d\nstderr:\n%s", exitCode, stderr)
			}
		})
	}
}

func TestSayE2E_DryRun_PitchFlag(t *testing.T) {
	_, stderr, exitCode := runCLI(t, nil, "say", "--dry-run", "--pitch", "0.05", "テスト")

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d\nstderr:\n%s", exitCode, stderr)
	}

	playbackLine := extractLineContaining(t, stderr, "[dry run] playback completed")
	if !strings.Contains(playbackLine, "pitch=0.05") {
		t.Errorf("expected playback line to contain 'pitch=0.05'\nline: %s", playbackLine)
	}
}

func TestSayE2E_DryRun_IntonationFlag(t *testing.T) {
	_, stderr, exitCode := runCLI(t, nil, "say", "--dry-run", "--intonation", "1.5", "テスト")

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d\nstderr:\n%s", exitCode, stderr)
	}

	playbackLine := extractLineContaining(t, stderr, "[dry run] playback completed")
	if !strings.Contains(playbackLine, "intonation=1.5") {
		t.Errorf("expected playback line to contain 'intonation=1.5'\nline: %s", playbackLine)
	}
}

func TestSayE2E_Verbose_IncreasesLogs(t *testing.T) {
	_, baseStderr, baseExit := runCLI(t, nil, "say", "--dry-run", "詳細ログ")
	if baseExit != 0 {
		t.Fatalf("expected exit code 0 for baseline, got %d\nstderr:\n%s", baseExit, baseStderr)
	}

	_, verboseStderr, verboseExit := runCLI(t, nil, "say", "--dry-run", "--verbose", "詳細ログ")
	if verboseExit != 0 {
		t.Fatalf("expected exit code 0 for --verbose, got %d\nstderr:\n%s", verboseExit, verboseStderr)
	}

	baseLines := countNonEmptyLines(baseStderr)
	verboseLines := countNonEmptyLines(verboseStderr)
	if verboseLines <= baseLines {
		t.Errorf("expected --verbose to produce more log lines than baseline: base=%d verbose=%d\nbase stderr:\n%s\nverbose stderr:\n%s",
			baseLines, verboseLines, baseStderr, verboseStderr)
	}
}
