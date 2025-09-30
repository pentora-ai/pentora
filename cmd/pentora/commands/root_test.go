package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestRootCommandPreparesWorkspaceAndRunsVersion(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("PENTORA_WORKSPACE", tmp)

	cmd := NewCommand()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"version", "--short"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("command execution failed: %v", err)
	}

	expected := []string{"scans", "logs", "queue", "cache", "reports", "audit"}
	for _, sub := range expected {
		if _, err := os.Stat(filepath.Join(tmp, sub)); err != nil {
			t.Fatalf("expected workspace subdir %q: %v", sub, err)
		}
	}
}
