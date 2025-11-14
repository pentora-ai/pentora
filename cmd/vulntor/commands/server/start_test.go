package server

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStartCommand_InvalidPortShowsSuggestions(t *testing.T) {
	cmd := newStartServerCommand()
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	cmd.SetArgs([]string{"--port", "0"})

	err := cmd.Execute()
	require.NoError(t, err)

	output := out.String()
	require.Contains(t, output, "✗ Failed to start server")
	require.Contains(t, output, "Use a port between 1 and 65535")
	require.Contains(t, output, "vulntor server start --port 8080")
}
