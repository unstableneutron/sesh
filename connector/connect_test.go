package connector

import (
	"testing"

	"github.com/joshmedeski/sesh/v2/dir"
	"github.com/joshmedeski/sesh/v2/home"
	"github.com/joshmedeski/sesh/v2/kitty"
	"github.com/joshmedeski/sesh/v2/lister"
	"github.com/joshmedeski/sesh/v2/model"
	"github.com/joshmedeski/sesh/v2/namer"
	"github.com/joshmedeski/sesh/v2/startup"
	"github.com/joshmedeski/sesh/v2/tmux"
	"github.com/joshmedeski/sesh/v2/tmuxinator"
	"github.com/joshmedeski/sesh/v2/zmx"
	"github.com/joshmedeski/sesh/v2/zoxide"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type connectorDeps struct {
	dir        *dir.MockDir
	home       *home.MockHome
	lister     *lister.MockLister
	namer      *namer.MockNamer
	startup    *startup.MockStartup
	tmux       *tmux.MockTmux
	zmx        *zmx.MockZmx
	kitty      *kitty.MockKitty
	zoxide     *zoxide.MockZoxide
	tmuxinator *tmuxinator.MockTmuxinator
}

func newConnectorDeps(t *testing.T) connectorDeps {
	return connectorDeps{
		dir:        new(dir.MockDir),
		home:       new(home.MockHome),
		lister:     lister.NewMockLister(t),
		namer:      new(namer.MockNamer),
		startup:    new(startup.MockStartup),
		tmux:       new(tmux.MockTmux),
		zmx:        zmx.NewMockZmx(t),
		kitty:      kitty.NewMockKitty(t),
		zoxide:     new(zoxide.MockZoxide),
		tmuxinator: new(tmuxinator.MockTmuxinator),
	}
}

func newTestConnector(config model.Config, deps connectorDeps) *RealConnector {
	return &RealConnector{
		config:     config,
		dir:        deps.dir,
		home:       deps.home,
		lister:     deps.lister,
		namer:      deps.namer,
		startup:    deps.startup,
		tmux:       deps.tmux,
		zmx:        deps.zmx,
		kitty:      deps.kitty,
		zoxide:     deps.zoxide,
		tmuxinator: deps.tmuxinator,
	}
}

func TestConnect_PrefersContextBackendOnNameCollision(t *testing.T) {
	deps := newConnectorDeps(t)
	c := newTestConnector(model.Config{DefaultBackend: model.BackendTmux}, deps)

	deps.tmux.On("IsAttached").Return(false).Twice()
	deps.zmx.EXPECT().IsAttached().Return(true).Times(2)
	deps.zmx.EXPECT().IsAttached().Return(false).Once()
	deps.zmx.EXPECT().CurrentSessionName().Return("other").Once()

	deps.lister.EXPECT().FindTmuxSession("work").Return(model.SeshSession{Src: "tmux", Backend: model.BackendTmux, Name: "work", Path: "/tmp/work"}, true).Once()
	deps.lister.EXPECT().FindZmxSession("work").Return(model.SeshSession{Src: "zmx", Backend: model.BackendZmx, Name: "work", Path: "/tmp/work-zmx"}, true).Once()
	deps.lister.EXPECT().FindTmuxinatorConfig("work").Return(model.SeshSession{}, false).Once()
	deps.lister.EXPECT().FindConfigSession("work").Return(model.SeshSession{}, false).Once()
	deps.lister.EXPECT().FindConfigWildcard("work").Return(model.WildcardConfig{}, false).Once()
	deps.home.On("ExpandHome", "work").Return("work", nil).Twice()
	deps.dir.On("Dir", "work").Return(false, "").Twice()
	deps.lister.EXPECT().FindZoxideSession("work").Return(model.SeshSession{}, false).Once()

	deps.zoxide.On("Add", "/tmp/work-zmx").Return(nil).Once()
	deps.zmx.EXPECT().Attach("work").Return("attached", nil).Once()

	_, err := c.Connect("work", model.ConnectOpts{})
	require.NoError(t, err)
}

func TestConnect_DualContextWithoutBackendErrors(t *testing.T) {
	deps := newConnectorDeps(t)
	c := newTestConnector(model.Config{DefaultBackend: model.BackendTmux}, deps)

	deps.tmux.On("IsAttached").Return(true).Once()
	deps.zmx.EXPECT().IsAttached().Return(true).Once()

	_, err := c.Connect("work", model.ConnectOpts{})
	assert.ErrorContains(t, err, "requires --backend")
}

func TestConnect_ConfigWildcard_RespectsBackendCompatibility(t *testing.T) {
	deps := newConnectorDeps(t)
	c := newTestConnector(model.Config{DefaultBackend: model.BackendTmux}, deps)

	deps.tmux.On("IsAttached").Return(false).Twice()
	deps.zmx.EXPECT().IsAttached().Return(false).Times(3)

	name := "~/code/project"
	deps.lister.EXPECT().FindTmuxSession(name).Return(model.SeshSession{}, false).Once()
	deps.lister.EXPECT().FindZmxSession(name).Return(model.SeshSession{}, false).Once()
	deps.lister.EXPECT().FindTmuxinatorConfig(name).Return(model.SeshSession{}, false).Once()
	deps.lister.EXPECT().FindConfigSession(name).Return(model.SeshSession{}, false).Once()
	deps.lister.EXPECT().FindConfigWildcard(name).Return(model.WildcardConfig{Pattern: "~/code/*"}, true).Once()
	deps.lister.EXPECT().FindConfigWildcard("/Users/test/code/project").Return(model.WildcardConfig{Pattern: "~/code/*"}, true).Once()
	deps.home.On("ExpandHome", name).Return("/Users/test/code/project", nil).Twice()
	deps.dir.On("Dir", "/Users/test/code/project").Return(true, "/Users/test/code/project").Twice()
	deps.namer.On("Name", "/Users/test/code/project").Return("project", nil).Twice()
	deps.lister.EXPECT().FindZoxideSession(name).Return(model.SeshSession{}, false).Once()

	deps.zoxide.On("Add", "/Users/test/code/project").Return(nil).Once()
	deps.zmx.EXPECT().Attach("project").Return("attached", nil).Once()

	_, err := c.Connect(name, model.ConnectOpts{Backend: model.BackendZmx})
	require.NoError(t, err)
}
