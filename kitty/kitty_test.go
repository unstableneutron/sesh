package kitty

import (
	"fmt"
	"testing"

	"github.com/joshmedeski/sesh/v2/oswrap"
	"github.com/joshmedeski/sesh/v2/shell"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCanRemoteControl(t *testing.T) {
	mockOs := oswrap.NewMockOs(t)
	mockShell := shell.NewMockShell(t)
	k := &RealKitty{os: mockOs, shell: mockShell}

	mockOs.EXPECT().Getenv("KITTY_WINDOW_ID").Return("123").Once()
	mockOs.EXPECT().Getenv("KITTY_LISTEN_ON").Return("unix:/tmp/kitty.sock").Once()
	mockOs.EXPECT().Getenv("KITTY_PUBLIC_KEY").Return("key").Once()
	assert.True(t, k.CanRemoteControl())

	mockOs.EXPECT().Getenv("KITTY_WINDOW_ID").Return("").Once()
	assert.False(t, k.CanRemoteControl())
}

func TestSendDetach(t *testing.T) {
	mockOs := oswrap.NewMockOs(t)
	mockShell := shell.NewMockShell(t)
	k := &RealKitty{os: mockOs, shell: mockShell}

	mockShell.EXPECT().Cmd("kitten", "@", "send-key", "--match", "id:171", "ctrl+backslash").Return("", nil).Once()
	err := k.SendDetach("171")
	require.NoError(t, err)
}

func TestQueueCommand(t *testing.T) {
	mockOs := oswrap.NewMockOs(t)
	mockShell := shell.NewMockShell(t)
	k := &RealKitty{os: mockOs, shell: mockShell}

	mockShell.EXPECT().Cmd("kitten", "@", "send-text", "--match", "id:171", "sesh connect work\n").Return("", nil).Once()
	err := k.QueueCommand("171", "sesh connect work")
	require.NoError(t, err)
}

func TestSendDetachRequiresWindowID(t *testing.T) {
	mockOs := oswrap.NewMockOs(t)
	mockShell := shell.NewMockShell(t)
	k := &RealKitty{os: mockOs, shell: mockShell}

	err := k.SendDetach("")
	assert.EqualError(t, err, "kitty window id is required")
}

func TestQueueCommandRequiresCommand(t *testing.T) {
	mockOs := oswrap.NewMockOs(t)
	mockShell := shell.NewMockShell(t)
	k := &RealKitty{os: mockOs, shell: mockShell}

	err := k.QueueCommand("171", "")
	assert.EqualError(t, err, "kitty queued command is required")
}

func TestQueueCommandWrapsCommandError(t *testing.T) {
	mockOs := oswrap.NewMockOs(t)
	mockShell := shell.NewMockShell(t)
	k := &RealKitty{os: mockOs, shell: mockShell}

	mockShell.EXPECT().Cmd("kitten", "@", "send-text", "--match", "id:171", "sesh connect work\n").Return("", fmt.Errorf("kitten missing")).Once()
	err := k.QueueCommand("171", "sesh connect work")
	assert.ErrorContains(t, err, "failed to queue kitty command")
}
