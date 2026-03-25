package connector

import (
	"fmt"
	"strings"

	"github.com/joshmedeski/sesh/v2/model"
)

func zmxStrategy(c *RealConnector, name string) (model.Connection, error) {
	session, exists := c.lister.FindZmxSession(name)
	if !exists {
		return model.Connection{Found: false}, nil
	}

	return model.Connection{
		Found:       true,
		Session:     session,
		Backend:     model.BackendZmx,
		New:         false,
		AddToZoxide: true,
	}, nil
}

func connectToZmx(c *RealConnector, connection model.Connection, opts model.ConnectOpts) (string, error) {
	if c.zmx == nil {
		return "", fmt.Errorf("zmx backend unavailable")
	}

	if c.zmx != nil && c.zmx.IsAttached() {
		if c.zmx.CurrentSessionName() == connection.Session.Name {
			return fmt.Sprintf("already in zmx session: %s", connection.Session.Name), nil
		}
		if opts.BypassHandoff {
			return "", manualZmxHandoffError("handoff replay marker is set", connection.Session.Name, model.BackendZmx, opts, nil)
		}
		return queueKittyHandoff(c, connection.Session.Name, model.BackendZmx, opts)
	}

	if c.tmux != nil && c.tmux.IsAttached() {
		return "", manualTmuxToZmxError(connection.Session.Name, opts)
	}

	if connection.New {
		startupCommand := resolveZmxStartupCommand(c, connection.Session, opts)
		if startupCommand != "" {
			if _, err := c.zmx.Run(connection.Session.Name, startupCommand); err != nil {
				return "", fmt.Errorf("failed to run command in zmx session: %w", err)
			}
		}
	}

	if _, err := c.zmx.Attach(connection.Session.Name); err != nil {
		return "", fmt.Errorf("failed to attach to zmx session: %w", err)
	}

	return fmt.Sprintf("attaching to zmx session: %s", connection.Session.Name), nil
}

func resolveZmxStartupCommand(c *RealConnector, session model.SeshSession, opts model.ConnectOpts) string {
	if opts.Command != "" {
		return opts.Command
	}

	if session.StartupCommand != "" {
		return applySessionPath(session.StartupCommand, session.Path)
	}

	if session.Src != "config" {
		if wc, found := c.lister.FindConfigWildcard(session.Path); found {
			if wc.DisableStartCommand {
				return ""
			}
			if wc.StartupCommand != "" {
				return applySessionPath(wc.StartupCommand, session.Path)
			}
		}
	}

	if session.DisableStartupCommand {
		return ""
	}

	if c.config.DefaultSessionConfig.StartupCommand != "" {
		return applySessionPath(c.config.DefaultSessionConfig.StartupCommand, session.Path)
	}

	return ""
}

func applySessionPath(command, path string) string {
	return strings.ReplaceAll(command, "{}", path)
}
