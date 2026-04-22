//go:build e2e

package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var binaryPath string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "vox-actor-e2e-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	binaryPath = filepath.Join(dir, "vox-actor")
	cmd := exec.Command("go", "build", "-o", binaryPath, "github.com/canpok1/vox-actor")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		panic("failed to build binary: " + err.Error())
	}

	os.Exit(m.Run())
}

// versionLineRegexp は `--version` の出力に1行だけ現れる `vox-actor version <文字列>` 形式を検査する。
var versionLineRegexp = regexp.MustCompile(`(?m)^vox-actor version \S+`)

func TestCLIVersion(t *testing.T) {
	stdout, stderr, exitCode := runCLI(t, nil, "--version")

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d\nstderr:\n%s", exitCode, stderr)
	}

	matches := versionLineRegexp.FindAllString(stdout, -1)
	if len(matches) != 1 {
		t.Fatalf("expected exactly 1 line matching %q in stdout, got %d\nstdout:\n%s",
			versionLineRegexp.String(), len(matches), stdout)
	}
}

func TestCLIHelp_ListsSubcommands(t *testing.T) {
	stdout, stderr, exitCode := runCLI(t, nil, "--help")

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d\nstderr:\n%s", exitCode, stderr)
	}

	wants := []string{"say", "act", "watch", "config", "audio-check"}
	for _, w := range wants {
		if !strings.Contains(stdout, w) {
			t.Errorf("expected help output to contain subcommand %q\nstdout:\n%s", w, stdout)
		}
	}
}

func TestCLINoArgs_ShowsHelp(t *testing.T) {
	stdout, stderr, exitCode := runCLI(t, nil)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d\nstderr:\n%s", exitCode, stderr)
	}

	if !strings.Contains(stdout, "vox-actor") {
		t.Errorf("expected help output to contain 'vox-actor'\nstdout:\n%s", stdout)
	}
	if !strings.Contains(stdout, "Usage:") {
		t.Errorf("expected help output to contain 'Usage:'\nstdout:\n%s", stdout)
	}
}

func TestCLIUnknownSubcommand_NonZeroWithStderr(t *testing.T) {
	_, stderr, exitCode := runCLI(t, nil, "unknown-cmd")

	if exitCode == 0 {
		t.Fatalf("expected non-zero exit code for unknown subcommand, got 0")
	}
	if strings.TrimSpace(stderr) == "" {
		t.Errorf("expected non-empty stderr for unknown subcommand")
	}
}

func TestCLIUnknownFlag_NonZero(t *testing.T) {
	_, _, exitCode := runCLI(t, nil, "--no-such-flag")

	if exitCode == 0 {
		t.Fatalf("expected non-zero exit code for unknown flag, got 0")
	}
}
