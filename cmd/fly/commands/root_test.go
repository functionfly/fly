package commands

import (
	"bytes"
	"testing"

	"github.com/functionfly/fly/internal/version"
)

// Smoke test: version subcommand runs without error.
func TestRootCmd_Version(t *testing.T) {
	root := NewRootCmd()
	root.SetArgs([]string{"version", "--short"})
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute() = %v", err)
	}
	_ = version.Short()
}

func TestRootCmd_Help(t *testing.T) {
	root := NewRootCmd()
	root.SetArgs([]string{"--help"})
	out := bytes.NewBuffer(nil)
	root.SetOut(out)
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute() = %v", err)
	}
	got := out.String()
	if !bytes.Contains([]byte(got), []byte("fly")) {
		t.Errorf("--help output should mention fly: %s", got)
	}
}
