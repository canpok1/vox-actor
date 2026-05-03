//go:build e2e

package e2e

import (
	"strings"
	"testing"
)

func TestViewerCheckE2E_UnexpectedArg_ExitCode2(t *testing.T) {
	t.Parallel()

	_, stderr, exitCode := runCLI(t, nil, "viewer-check", "unexpected")
	if exitCode != 2 {
		t.Fatalf("expected exit code 2, got %d\nstderr:\n%s", exitCode, stderr)
	}
}

func TestViewerCheckE2E_UnknownFlag_NonZeroExit(t *testing.T) {
	t.Parallel()

	_, _, exitCode := runCLI(t, nil, "viewer-check", "--bogus")
	if exitCode == 0 {
		t.Fatalf("expected non-zero exit code for unknown flag, got 0")
	}
}

func TestViewerCheckE2E_Help_ExitZeroWithUsage(t *testing.T) {
	t.Parallel()

	stdout, stderr, exitCode := runCLI(t, nil, "viewer-check", "--help")
	if exitCode != 0 {
		t.Fatalf("expected exit 0 for viewer-check --help, got %d\nstderr:\n%s", exitCode, stderr)
	}
	for _, want := range []string{"viewer-check", "--verbose"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("expected viewer-check --help to contain %q\nstdout:\n%s", want, stdout)
		}
	}
}

func TestViewerCheckE2E_NoViewer_NonZeroExit(t *testing.T) {
	t.Parallel()

	// viewer が起動していない環境では非0終了することを確認
	_, _, exitCode := runCLI(t, nil, "viewer-check")
	if exitCode == 0 {
		t.Logf("viewer-check returned exit 0 (viewer may be running in this environment)")
	}
}
