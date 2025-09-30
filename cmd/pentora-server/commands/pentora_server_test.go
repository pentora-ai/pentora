package commands

import (
	"path/filepath"
	"testing"
)

func TestNewCommandPreparesWorkspace(t *testing.T) {
	temp := t.TempDir()
	ws := filepath.Join(temp, "ws")

	cmd := NewCommand()
	cmd.SetArgs([]string{"--workspace-dir", ws})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("command execute returned error: %v", err)
	}
}
