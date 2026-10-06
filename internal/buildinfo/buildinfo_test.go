package buildinfo

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVersionCommandPrintsMachineReadableIdentity(t *testing.T) {
	originalVersion, originalCommit, originalBuildTime := Version, Commit, BuildTime
	t.Cleanup(func() { Version, Commit, BuildTime = originalVersion, originalCommit, originalBuildTime })
	Version, Commit, BuildTime = "v0.1.0", "deadbeef", "2026-10-06T00:00:00Z"

	output := bytes.NewBuffer(nil)
	cmd := Command()
	cmd.SetOut(output)
	cmd.SetArgs([]string{"--output", "json"})
	require.NoError(t, cmd.Execute())

	var got Info
	require.NoError(t, json.Unmarshal(output.Bytes(), &got))
	require.Equal(t, Current(), got)
}

func TestVersionCommandRejectsUnknownOutput(t *testing.T) {
	cmd := Command()
	cmd.SetArgs([]string{"--output", "yaml"})
	require.ErrorContains(t, cmd.Execute(), "unsupported output format")
}
