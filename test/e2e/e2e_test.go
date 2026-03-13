//go:build e2e

package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
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

func TestCLIVersion(t *testing.T) {
	cmd := exec.Command(binaryPath, "--version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to run CLI: %v\n%s", err, out)
	}
	if len(out) == 0 {
		t.Fatal("expected version output, got empty")
	}
}
