package zmx

import (
	"testing"

	"github.com/joshmedeski/sesh/v2/oswrap"
	"github.com/joshmedeski/sesh/v2/shell"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListSessionsParsesClientsAndName(t *testing.T) {
	mockShell := shell.NewMockShell(t)
	mockOs := oswrap.NewMockOs(t)
	z := &RealZmx{os: mockOs, shell: mockShell}

	mockShell.EXPECT().ListCmd("zmx", "list").Return([]string{
		"name=work\tpid=101\tclients=1\tcreated=1774473535\tstart_dir=/tmp/work",
		"name=notes\tpid=102\tclients=0\tcreated=1774473535\tstart_dir=/tmp/notes",
	}, nil)

	got, err := z.ListSessions()
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "work", got[0].Name)
	assert.Equal(t, 1, got[0].Clients)
	assert.Equal(t, "/tmp/work", got[0].StartDir)
}

func TestIsAttachedAndCurrentSessionName(t *testing.T) {
	mockShell := shell.NewMockShell(t)
	mockOs := oswrap.NewMockOs(t)
	z := &RealZmx{os: mockOs, shell: mockShell}

	mockOs.EXPECT().Getenv("ZMX_SESSION").Return("dev").Once()
	assert.True(t, z.IsAttached())

	mockOs.EXPECT().Getenv("ZMX_SESSION").Return("work").Once()
	assert.Equal(t, "work", z.CurrentSessionName())
}

func TestAttachAndRun(t *testing.T) {
	mockShell := shell.NewMockShell(t)
	mockOs := oswrap.NewMockOs(t)
	z := &RealZmx{os: mockOs, shell: mockShell}

	mockShell.EXPECT().Cmd("zmx", "attach", "work").Return("attached", nil)
	msg, err := z.Attach("work")
	require.NoError(t, err)
	assert.Equal(t, "attached", msg)

	mockShell.EXPECT().Cmd("zmx", "run", "work", "echo", "hi").Return("queued", nil)
	msg, err = z.Run("work", "echo", "hi")
	require.NoError(t, err)
	assert.Equal(t, "queued", msg)
}
