package lister

import (
	"os/exec"
	"testing"

	"github.com/joshmedeski/sesh/v2/home"
	"github.com/joshmedeski/sesh/v2/model"
	"github.com/joshmedeski/sesh/v2/tmux"
	"github.com/joshmedeski/sesh/v2/tmuxinator"
	"github.com/joshmedeski/sesh/v2/zmx"
	"github.com/joshmedeski/sesh/v2/zoxide"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestList_OmitsZmxWhenBinaryMissing(t *testing.T) {
	mockTmux := new(tmux.MockTmux)
	mockZmx := zmx.NewMockZmx(t)
	mockHome := new(home.MockHome)
	mockZoxide := new(zoxide.MockZoxide)
	mockTmuxinator := new(tmuxinator.MockTmuxinator)

	mockTmux.On("ListSessions").Return([]*model.TmuxSession{}, nil)
	mockZmx.EXPECT().ListSessions().Return(nil, &exec.Error{Name: "zmx", Err: exec.ErrNotFound}).Once()
	mockZoxide.On("ListResults").Return([]*model.ZoxideResult{}, nil)
	mockTmuxinator.On("List").Return([]*model.TmuxinatorConfig{}, nil)

	l := NewLister(model.Config{}, mockHome, mockTmux, mockZmx, mockZoxide, mockTmuxinator)
	sessions, err := l.List(ListOptions{})

	require.NoError(t, err)
	for _, key := range sessions.OrderedIndex {
		assert.NotContains(t, key, "zmx:")
	}
}

func TestListZmxSessions(t *testing.T) {
	mockTmux := new(tmux.MockTmux)
	mockZmx := zmx.NewMockZmx(t)
	mockHome := new(home.MockHome)
	mockZoxide := new(zoxide.MockZoxide)
	mockTmuxinator := new(tmuxinator.MockTmuxinator)

	mockZmx.EXPECT().ListSessions().Return([]*model.ZmxSession{
		{Name: "work", Clients: 1, StartDir: "/tmp/work"},
		{Name: "notes", Clients: 0, StartDir: "/tmp/notes"},
	}, nil).Once()

	l := NewLister(model.Config{}, mockHome, mockTmux, mockZmx, mockZoxide, mockTmuxinator)
	sessions, err := l.List(ListOptions{Zmx: true})

	require.NoError(t, err)
	require.Equal(t, []string{"zmx:work", "zmx:notes"}, sessions.OrderedIndex)
	assert.Equal(t, model.BackendZmx, sessions.Directory["zmx:work"].Backend)
	assert.Equal(t, "/tmp/work", sessions.Directory["zmx:work"].Path)
}

func TestGetAttachedZmxSession(t *testing.T) {
	mockTmux := new(tmux.MockTmux)
	mockZmx := zmx.NewMockZmx(t)
	mockHome := new(home.MockHome)
	mockZoxide := new(zoxide.MockZoxide)
	mockTmuxinator := new(tmuxinator.MockTmuxinator)

	mockZmx.EXPECT().IsAttached().Return(true).Once()
	mockZmx.EXPECT().CurrentSessionName().Return("work").Once()
	mockZmx.EXPECT().ListSessions().Return([]*model.ZmxSession{{Name: "work", Clients: 2, StartDir: "/tmp/work"}}, nil).Once()

	l := NewLister(model.Config{}, mockHome, mockTmux, mockZmx, mockZoxide, mockTmuxinator)
	attached, ok := l.GetAttachedZmxSession()

	require.True(t, ok)
	assert.Equal(t, "work", attached.Name)
	assert.Equal(t, "zmx", attached.Src)
	assert.Equal(t, model.BackendZmx, attached.Backend)
}
