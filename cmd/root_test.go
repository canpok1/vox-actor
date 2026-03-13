package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestExecute_NoArgs_ShowsHelp(t *testing.T) {
	rootCmd := makeRootCmd()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Execute() without args should not return error, got: %v", err)
	}

	output := buf.String()
	if len(output) == 0 {
		t.Error("expected help output, got empty string")
	}
	if !strings.Contains(output, "vox-actor") {
		t.Errorf("expected help output to contain 'vox-actor', got: %s", output)
	}
}

func TestHelp_ContainsCommandName(t *testing.T) {
	rootCmd := makeRootCmd()

	if rootCmd.Use != "vox-actor" {
		t.Errorf("expected Use to be 'vox-actor', got: %s", rootCmd.Use)
	}
}

func TestHelp_ShowsHelpOutput(t *testing.T) {
	rootCmd := makeRootCmd()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"--help"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	output := buf.String()
	if len(output) == 0 {
		t.Error("help output should not be empty")
	}
}
