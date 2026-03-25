package zmx

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/joshmedeski/sesh/v2/model"
)

func (z *RealZmx) ListSessions() ([]*model.ZmxSession, error) {
	lines, err := z.shell.ListCmd("zmx", "list")
	if err != nil {
		return nil, err
	}

	sessions := make([]*model.ZmxSession, 0, len(lines))
	for _, line := range lines {
		session, ok, err := parseListLine(line)
		if err != nil {
			return nil, err
		}
		if ok {
			sessions = append(sessions, session)
		}
	}

	return sessions, nil
}

func parseListLine(line string) (*model.ZmxSession, bool, error) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "no sessions found") {
		return nil, false, nil
	}

	fields := map[string]string{}
	for _, part := range strings.Split(trimmed, "\t") {
		key, value, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		fields[key] = strings.Trim(value, "'\"")
	}

	name := fields["name"]
	if name == "" {
		return nil, false, nil
	}

	clients := 0
	if rawClients, ok := fields["clients"]; ok && rawClients != "" {
		parsedClients, err := strconv.Atoi(rawClients)
		if err != nil {
			return nil, false, fmt.Errorf("failed to parse zmx clients for %q: %w", name, err)
		}
		clients = parsedClients
	}

	return &model.ZmxSession{
		Name:     name,
		Clients:  clients,
		StartDir: fields["start_dir"],
	}, true, nil
}
