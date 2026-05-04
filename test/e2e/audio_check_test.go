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

// skipIfAudioDevice は audio-check が成功する環境（音声デバイス有り）でテストをスキップする。
// 音声デバイスが存在しないことを前提とするテストで使う。
func skipIfAudioDevice(t *testing.T) {
	t.Helper()
	_, _, exitCode := runCLI(t, nil, "audio-check")
	if exitCode == 0 {
		t.Skip("skipping: audio device available")
	}
}

func TestAudioCheckE2E_DeviceAvailable_ExitZero(t *testing.T) {
	t.Parallel()
	skipIfNoAudioDevice(t)

	stdout, stderr, exitCode := runCLI(t, nil, "audio-check")
	if exitCode != 0 {
		t.Fatalf("expected exit code 0 when audio device is available, got %d\nstderr:\n%s", exitCode, stderr)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("expected empty stdout on success, got: %s", stdout)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Errorf("expected empty stderr on success (no --verbose), got: %s", stderr)
	}
}

func TestAudioCheckE2E_VerboseBackend_Success(t *testing.T) {
	t.Parallel()
	skipIfNoAudioDevice(t)

	_, stderr, exitCode := runCLI(t, nil, "audio-check", "--verbose")
	if exitCode != 0 {
		t.Fatalf("expected exit code 0 with audio device available, got %d\nstderr:\n%s", exitCode, stderr)
	}
	if !strings.Contains(stderr, "audio backend:") {
		t.Errorf("expected stderr to contain 'audio backend:' with --verbose\nstderr:\n%s", stderr)
	}
}

func TestAudioCheckE2E_DeviceUnavailable_ExitOne(t *testing.T) {
	t.Parallel()
	skipIfAudioDevice(t)

	// ALSA ライブラリが stderr に診断メッセージを出力する場合があるため stderr は検証しない。
	stdout, stderr, exitCode := runCLI(t, nil, "audio-check")
	if exitCode != 1 {
		t.Fatalf("expected exit code 1 when audio device is unavailable, got %d\nstderr:\n%s", exitCode, stderr)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("expected empty stdout on failure, got: %s", stdout)
	}
}

func TestAudioCheckE2E_VerboseError_Failure(t *testing.T) {
	t.Parallel()
	skipIfAudioDevice(t)

	_, stderr, exitCode := runCLI(t, nil, "audio-check", "--verbose")
	if exitCode != 1 {
		t.Fatalf("expected exit code 1 when audio device is unavailable, got %d\nstderr:\n%s", exitCode, stderr)
	}
	if !strings.Contains(stderr, "Error:") {
		t.Errorf("expected stderr to contain 'Error:' with --verbose\nstderr:\n%s", stderr)
	}
}
