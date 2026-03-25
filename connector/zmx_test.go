package connector

import (
	"strings"
	"testing"

	"github.com/joshmedeski/sesh/v2/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func testBackendConnection(name string, backend model.Backend) model.Connection {
	return model.Connection{
		Found: true,
		Session: model.SeshSession{
			Name: name,
			Path: "/tmp/" + name,
		},
		Backend: backend,
	}
}

func TestConnectToZmx_InsideZmxWithKitty_QueuesReconnect(t *testing.T) {
	deps := newConnectorDeps(t)
	c := newTestConnector(model.Config{}, deps)
	connection := testBackendConnection("work", model.BackendZmx)
	opts := model.ConnectOpts{ReplayName: "work", KittyWindowID: "171"}

	deps.zmx.EXPECT().IsAttached().Return(true).Once()
	deps.zmx.EXPECT().CurrentSessionName().Return("other").Once()
	deps.kitty.EXPECT().CanRemoteControl().Return(true).Once()
	deps.kitty.EXPECT().SendDetach("171").Return(nil).Once()
	deps.kitty.EXPECT().QueueCommand("171", mock.MatchedBy(func(command string) bool {
		return strings.Contains(command, "sesh connect") &&
			strings.Contains(command, "--backend zmx") &&
			strings.Contains(command, "--"+model.BypassHandoffFlag)
	})).Return(nil).Once()

	message, err := connectToZmx(c, connection, opts)
	require.NoError(t, err)
	assert.Contains(t, message, "handoff")
}

func TestConnectToZmx_KittyUnavailable_ReturnsManualGuidance(t *testing.T) {
	deps := newConnectorDeps(t)
	c := newTestConnector(model.Config{}, deps)
	connection := testBackendConnection("work", model.BackendZmx)
	opts := model.ConnectOpts{ReplayName: "work", KittyWindowID: "171"}

	deps.zmx.EXPECT().IsAttached().Return(true).Once()
	deps.zmx.EXPECT().CurrentSessionName().Return("other").Once()
	deps.kitty.EXPECT().CanRemoteControl().Return(false).Once()

	_, err := connectToZmx(c, connection, opts)
	assert.ErrorContains(t, err, "Ctrl+\\")
	assert.ErrorContains(t, err, "sesh connect")
}

func TestConnectToTmux_InsideZmxWithKitty_QueuesReconnect(t *testing.T) {
	deps := newConnectorDeps(t)
	c := newTestConnector(model.Config{}, deps)
	connection := testBackendConnection("work", model.BackendTmux)
	opts := model.ConnectOpts{ReplayName: "work", KittyWindowID: "171", Switch: true}

	deps.zmx.EXPECT().IsAttached().Return(true).Once()
	deps.kitty.EXPECT().CanRemoteControl().Return(true).Once()
	deps.kitty.EXPECT().SendDetach("171").Return(nil).Once()
	deps.kitty.EXPECT().QueueCommand("171", mock.MatchedBy(func(command string) bool {
		return strings.Contains(command, "sesh connect") &&
			strings.Contains(command, "--backend tmux") &&
			strings.Contains(command, "--switch") &&
			strings.Contains(command, "--"+model.BypassHandoffFlag)
	})).Return(nil).Once()

	message, err := connectToTmux(c, connection, opts)
	require.NoError(t, err)
	assert.Contains(t, message, "handoff")
}

func TestConnectToZmx_SameSession_NoOp(t *testing.T) {
	deps := newConnectorDeps(t)
	c := newTestConnector(model.Config{}, deps)
	connection := testBackendConnection("work", model.BackendZmx)

	deps.zmx.EXPECT().IsAttached().Return(true).Once()
	deps.zmx.EXPECT().CurrentSessionName().Return("work").Once()

	message, err := connectToZmx(c, connection, model.ConnectOpts{ReplayName: "work"})
	require.NoError(t, err)
	assert.Contains(t, message, "already in zmx session")
}

func TestConnectToZmx_InsideTmux_ReturnsManualGuidance(t *testing.T) {
	deps := newConnectorDeps(t)
	c := newTestConnector(model.Config{}, deps)
	connection := testBackendConnection("work", model.BackendZmx)

	deps.zmx.EXPECT().IsAttached().Return(false).Once()
	deps.tmux.On("IsAttached").Return(true).Once()

	_, err := connectToZmx(c, connection, model.ConnectOpts{ReplayName: "work"})
	assert.ErrorContains(t, err, "no automatic handoff")
	assert.ErrorContains(t, err, "sesh connect")
}

func TestConnectToTmux_InsideZmxBypassMarker_ReturnsManualGuidance(t *testing.T) {
	deps := newConnectorDeps(t)
	c := newTestConnector(model.Config{}, deps)
	connection := testBackendConnection("work", model.BackendTmux)

	deps.zmx.EXPECT().IsAttached().Return(true).Once()

	_, err := connectToTmux(c, connection, model.ConnectOpts{ReplayName: "work", BypassHandoff: true})
	assert.ErrorContains(t, err, "handoff replay marker")
	assert.ErrorContains(t, err, "Ctrl+\\")
}

func TestConnectToZmx_InsideZmxBypassMarker_ReturnsManualGuidance(t *testing.T) {
	deps := newConnectorDeps(t)
	c := newTestConnector(model.Config{}, deps)
	connection := testBackendConnection("work", model.BackendZmx)

	deps.zmx.EXPECT().IsAttached().Return(true).Once()
	deps.zmx.EXPECT().CurrentSessionName().Return("other").Once()

	_, err := connectToZmx(c, connection, model.ConnectOpts{ReplayName: "work", BypassHandoff: true})
	assert.ErrorContains(t, err, "handoff replay marker")
	assert.ErrorContains(t, err, "Ctrl+\\")
}

func TestConnectToZmx_NewSessionRunsStartupCommand(t *testing.T) {
	deps := newConnectorDeps(t)
	c := newTestConnector(model.Config{}, deps)
	connection := testBackendConnection("work", model.BackendZmx)
	connection.New = true
	connection.Session.StartupCommand = "echo ready"

	deps.zmx.EXPECT().IsAttached().Return(false).Once()
	deps.tmux.On("IsAttached").Return(false).Once()
	deps.zmx.EXPECT().Run("work", "echo ready").Return("", nil).Once()
	deps.zmx.EXPECT().Attach("work").Return("attached", nil).Once()

	message, err := connectToZmx(c, connection, model.ConnectOpts{})
	require.NoError(t, err)
	assert.Contains(t, message, "attaching to zmx session")
}

func TestResolveZmxStartupCommand_ConfigSessionSkipsWildcardLookup(t *testing.T) {
	deps := newConnectorDeps(t)
	config := model.Config{}
	config.DefaultSessionConfig.StartupCommand = "echo {}"
	c := newTestConnector(config, deps)

	command := resolveZmxStartupCommand(c, model.SeshSession{Src: "config", Path: "/tmp/work"}, model.ConnectOpts{})
	assert.Equal(t, "echo /tmp/work", command)
}

func TestResolveZmxStartupCommand_ConfigSessionDisableStartupSkipsWildcardLookup(t *testing.T) {
	deps := newConnectorDeps(t)
	config := model.Config{}
	config.DefaultSessionConfig.StartupCommand = "echo {}"
	c := newTestConnector(config, deps)

	command := resolveZmxStartupCommand(c, model.SeshSession{Src: "config", Path: "/tmp/work", DisableStartupCommand: true}, model.ConnectOpts{})
	assert.Empty(t, command)
}

func TestResolveZmxStartupCommand_WildcardSessionUsesWildcardStartup(t *testing.T) {
	deps := newConnectorDeps(t)
	c := newTestConnector(model.Config{}, deps)
	deps.lister.EXPECT().FindConfigWildcard("/tmp/work").Return(model.WildcardConfig{StartupCommand: "echo {}"}, true).Once()

	command := resolveZmxStartupCommand(c, model.SeshSession{Src: "config_wildcard", Path: "/tmp/work"}, model.ConnectOpts{})
	assert.Equal(t, "echo /tmp/work", command)
}
