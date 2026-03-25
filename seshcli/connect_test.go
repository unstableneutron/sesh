package seshcli

import (
	"testing"

	"github.com/joshmedeski/sesh/v2/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseBackendFlag_AllowsSupportedBackends(t *testing.T) {
	backend, err := parseBackendFlag("zmx")
	require.NoError(t, err)
	assert.Equal(t, model.BackendZmx, backend)

	backend, err = parseBackendFlag("tmux")
	require.NoError(t, err)
	assert.Equal(t, model.BackendTmux, backend)

	backend, err = parseBackendFlag("")
	require.NoError(t, err)
	assert.Empty(t, backend)
}

func TestParseBackendFlag_RejectsInvalidBackend(t *testing.T) {
	_, err := parseBackendFlag("wezterm")
	assert.ErrorContains(t, err, "invalid backend \"wezterm\"")
	assert.ErrorContains(t, err, "tmux or zmx")
}

func TestConnectCommand_InvalidBackendReturnsValidationError(t *testing.T) {
	cmd := NewConnectCommand(&BaseDeps{})
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs([]string{"work", "--backend", "wezterm"})

	err := cmd.Execute()
	assert.ErrorContains(t, err, "invalid backend \"wezterm\"")
}
