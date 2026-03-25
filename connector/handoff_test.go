package connector

import (
	"testing"

	"github.com/joshmedeski/sesh/v2/model"
	"github.com/stretchr/testify/assert"
)

func TestBuildReplayConnectCommand_PreservesConnectIntent(t *testing.T) {
	command := buildReplayConnectCommand("fallback", model.BackendZmx, model.ConnectOpts{
		Switch:     true,
		Command:    "echo hello world",
		Tmuxinator: true,
		ConfigPath: "/tmp/sesh.toml",
		SourceHint: "config",
		ReplayName: "work session",
	}, true)

	assert.Contains(t, command, "sesh connect")
	assert.Contains(t, command, "--config")
	assert.Contains(t, command, "/tmp/sesh.toml")
	assert.Contains(t, command, "--backend zmx")
	assert.Contains(t, command, "--switch")
	assert.Contains(t, command, "--command")
	assert.Contains(t, command, "echo hello world")
	assert.Contains(t, command, "--tmuxinator")
	assert.Contains(t, command, "--source-hint config")
	assert.Contains(t, command, "--"+model.BypassHandoffFlag)
	assert.Contains(t, command, "work session")
	assert.NotContains(t, command, "fallback")
}

func TestBuildReplayConnectCommand_WithoutBypassMarker(t *testing.T) {
	command := buildReplayConnectCommand("work", model.BackendTmux, model.ConnectOpts{}, false)
	assert.NotContains(t, command, "--"+model.BypassHandoffFlag)
	assert.Contains(t, command, "--backend tmux")
}
