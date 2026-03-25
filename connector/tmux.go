package connector

import "github.com/joshmedeski/sesh/v2/model"

func tmuxStrategy(c *RealConnector, name string) (model.Connection, error) {
	session, exists := c.lister.FindTmuxSession(name)
	if !exists {
		return model.Connection{Found: false}, nil
	}
	return model.Connection{
		Found:       true,
		Session:     session,
		Backend:     model.BackendTmux,
		New:         false,
		AddToZoxide: true,
	}, nil
}

func connectToTmux(c *RealConnector, connection model.Connection, opts model.ConnectOpts) (string, error) {
	if c.zmx != nil && c.zmx.IsAttached() {
		if opts.BypassHandoff {
			return "", manualZmxHandoffError("handoff replay marker is set", connection.Session.Name, model.BackendTmux, opts, nil)
		}
		return queueKittyHandoff(c, connection.Session.Name, model.BackendTmux, opts)
	}

	if connection.New {
		c.tmux.NewSession(connection.Session.Name, connection.Session.Path)
		if opts.Command != "" {
			c.tmux.SendKeys(connection.Session.Name, opts.Command)
		} else {
			c.startup.Exec(connection.Session)
		}
	}
	return c.tmux.SwitchOrAttach(connection.Session.Name, opts)
}
