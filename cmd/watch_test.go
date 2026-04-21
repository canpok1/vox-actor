package cmd

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"

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

	stream, _ := watchCmd.Flags().GetBool("stream")
	if stream {
		t.Error("expected --stream default to be false")
	}

	streamAddr, _ := watchCmd.Flags().GetString("stream-addr")
	if streamAddr != "127.0.0.1:8080" {
		t.Errorf("expected default stream-addr '127.0.0.1:8080', got: %s", streamAddr)
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
	flags := []string{"--engine-url", "--speaker", "--speed", "--pitch", "--intonation", "--delete", "--stream", "--stream-addr", "--verbose", "--dry-run"}
	for _, flag := range flags {
		if !strings.Contains(output, flag) {
			t.Errorf("expected help output to contain '%s'", flag)
		}
	}
}

func TestWatchCmd_StreamAndDryRun_ReturnsUsageError(t *testing.T) {
	dir := t.TempDir()

	rootCmd := makeRootCmd()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"watch", "--stream", "--dry-run", dir})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when --stream and --dry-run are combined")
	}
	if !errors.Is(err, ErrUsage) {
		t.Errorf("expected ErrUsage, got: %v", err)
	}
}

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
