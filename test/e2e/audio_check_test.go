//go:build e2e

package e2e

import (
	"strings"
	"testing"
)

func TestAudioCheckE2E_UnexpectedArg_ExitCode2(t *testing.T) {
	t.Parallel()

	_, stderr, exitCode := runCLI(t, nil, "audio-check", "unexpected")
	if exitCode != 2 {
		t.Fatalf("expected exit code 2, got %d\nstderr:\n%s", exitCode, stderr)
	}
}

func TestAudioCheckE2E_UnknownFlag_NonZeroExit(t *testing.T) {
	t.Parallel()

	_, _, exitCode := runCLI(t, nil, "audio-check", "--bogus")
	if exitCode == 0 {
		t.Fatalf("expected non-zero exit code for unknown flag, got 0")
	}
}

func TestAudioCheckE2E_Help_ExitZeroWithUsage(t *testing.T) {
	t.Parallel()

	stdout, stderr, exitCode := runCLI(t, nil, "audio-check", "--help")
	if exitCode != 0 {
		t.Fatalf("expected exit 0 for audio-check --help, got %d\nstderr:\n%s", exitCode, stderr)
	}
	for _, want := range []string{"audio-check", "--verbose"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("expected audio-check --help to contain %q\nstdout:\n%s", want, stdout)
		}
	}
}
