package connector

import (
	"fmt"

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
	if connection.New && opts.Command != "" {
		if _, err := c.zmx.Run(connection.Session.Name, opts.Command); err != nil {
			return "", fmt.Errorf("failed to run command in zmx session: %w", err)
		}
	}

	if _, err := c.zmx.Attach(connection.Session.Name); err != nil {
		return "", fmt.Errorf("failed to attach to zmx session: %w", err)
	}

	return fmt.Sprintf("attaching to zmx session: %s", connection.Session.Name), nil
}
