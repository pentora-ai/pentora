package commands

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFingerprintSyncCommand_SourceRequiredSuggestions(t *testing.T) {
	cmd := NewFingerprintCommand()
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	cmd.SetArgs([]string{"sync"})

	err := cmd.Execute()
	require.NoError(t, err)

	output := out.String()
	require.Contains(t, output, "✗ Failed to sync fingerprint catalog")
	require.Contains(t, output, "--file <path> or --url <address>")
	require.Contains(t, output, "pentora fingerprint sync --url")
}
