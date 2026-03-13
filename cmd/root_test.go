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

func TestVersion_DefaultIsDev(t *testing.T) {
	if version != "dev" {
		t.Errorf("expected default version to be 'dev', got: %s", version)
	}
}

func TestRootCmd_HasVersion(t *testing.T) {
	rootCmd := makeRootCmd()
	if rootCmd.Version != version {
		t.Errorf("expected Version to be '%s', got: '%s'", version, rootCmd.Version)
	}
}

func TestVersion_Flag(t *testing.T) {
	rootCmd := makeRootCmd()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"--version"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Execute() with --version returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, version) {
		t.Errorf("expected version output to contain '%s', got: %s", version, output)
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
