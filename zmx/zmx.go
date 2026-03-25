package zmx

import (
	"github.com/joshmedeski/sesh/v2/model"
	"github.com/joshmedeski/sesh/v2/oswrap"
	"github.com/joshmedeski/sesh/v2/shell"
)

type Zmx interface {
	ListSessions() ([]*model.ZmxSession, error)
	IsAttached() bool
	CurrentSessionName() string
	Attach(name string) (string, error)
	Run(name string, args ...string) (string, error)
}

type RealZmx struct {
	os    oswrap.Os
	shell shell.Shell
}

func NewZmx(os oswrap.Os, shell shell.Shell) Zmx {
	return &RealZmx{os: os, shell: shell}
}

func (z *RealZmx) IsAttached() bool {
	return z.CurrentSessionName() != ""
}

func (z *RealZmx) CurrentSessionName() string {
	return z.os.Getenv("ZMX_SESSION")
}

func (z *RealZmx) Attach(name string) (string, error) {
	return z.shell.Cmd("zmx", "attach", name)
}

func (z *RealZmx) Run(name string, args ...string) (string, error) {
	cmdArgs := append([]string{"run", name}, args...)
	return z.shell.Cmd("zmx", cmdArgs...)
}
