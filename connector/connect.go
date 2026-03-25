package connector

import (
	"fmt"
	"strings"

	"github.com/joshmedeski/sesh/v2/model"
)

func (c *RealConnector) Connect(name string, opts model.ConnectOpts) (string, error) {
	if opts.Backend != "" && !opts.Backend.IsValid() {
		return "", fmt.Errorf("invalid backend %q: allowed values are tmux or zmx", opts.Backend)
	}

	ctx := c.currentContext()
	if ctx.inTmux && ctx.inZmx && opts.Backend == "" {
		return "", fmt.Errorf("ambiguous terminal context (TMUX and ZMX_SESSION are both set); this operation requires --backend")
	}

	connection, err := c.resolveConnection(name, opts, ctx)
	if err != nil {
		return "", err
	}

	if connection.Backend == model.BackendZmx && c.zmx != nil && c.zmx.IsAttached() {
		if c.zmx.CurrentSessionName() == connection.Session.Name {
			return fmt.Sprintf("already in zmx session: %s", connection.Session.Name), nil
		}
	}

	if connection.AddToZoxide {
		c.zoxide.Add(connection.Session.Path)
	}

	switch connection.Backend {
	case model.BackendTmux:
		return connectToTmux(c, connection, opts)
	case model.BackendZmx:
		return connectToZmx(c, connection, opts)
	default:
		return "", fmt.Errorf("unsupported backend %q", connection.Backend)
	}
}

type connectorContext struct {
	inTmux        bool
	inZmx         bool
	activeBackend model.Backend
}

func (c *RealConnector) currentContext() connectorContext {
	inTmux := c.tmux.IsAttached()
	inZmx := c.zmx != nil && c.zmx.IsAttached()

	ctx := connectorContext{inTmux: inTmux, inZmx: inZmx}
	switch {
	case inTmux && !inZmx:
		ctx.activeBackend = model.BackendTmux
	case inZmx && !inTmux:
		ctx.activeBackend = model.BackendZmx
	}

	return ctx
}

func (c *RealConnector) resolveConnection(name string, opts model.ConnectOpts, ctx connectorContext) (model.Connection, error) {
	candidates, err := c.collectCandidates(name)
	if err != nil {
		return model.Connection{}, err
	}
	if len(candidates) == 0 {
		return model.Connection{}, fmt.Errorf("no connection found for '%s'", name)
	}

	candidateBackends := backendsFromOpts(opts)
	resolvedBackend := c.resolveBackend(ctx, candidateBackends, candidates)
	selected, err := selectConnection(name, opts.SourceHint, resolvedBackend, candidates)
	if err != nil {
		return model.Connection{}, err
	}

	selected.Backend = resolvedBackend
	if selected.Session.Backend == "" {
		selected.Session.Backend = resolvedBackend
	}

	if !isSourceCompatible(selected.Session.Src, resolvedBackend) {
		return model.Connection{}, fmt.Errorf("source %q is not compatible with backend %q", selected.Session.Src, resolvedBackend)
	}

	return selected, nil
}

func (c *RealConnector) collectCandidates(name string) ([]model.Connection, error) {
	strategies := []func(*RealConnector, string) (model.Connection, error){
		tmuxStrategy,
		zmxStrategy,
		tmuxinatorStrategy,
		configStrategy,
		configWildcardStrategy,
		dirStrategy,
		zoxideStrategy,
	}

	candidates := make([]model.Connection, 0, len(strategies))
	for _, strategy := range strategies {
		connection, err := strategy(c, name)
		if err != nil {
			return nil, fmt.Errorf("failed to establish connection: %w", err)
		}
		if connection.Found {
			candidates = append(candidates, connection)
		}
	}

	return candidates, nil
}

func backendsFromOpts(opts model.ConnectOpts) []model.Backend {
	if opts.Backend != "" {
		return []model.Backend{opts.Backend}
	}
	return []model.Backend{model.BackendTmux, model.BackendZmx}
}

func (c *RealConnector) resolveBackend(ctx connectorContext, candidateBackends []model.Backend, candidates []model.Connection) model.Backend {
	runtimeMatches := runtimeMatchesByBackend(candidateBackends, candidates)

	switch len(runtimeMatches) {
	case 0:
		if backend, ok := firstMatchingBackend(candidateBackends, ctx.activeBackend, c.config.DefaultBackend, model.BackendTmux); ok {
			return backend
		}
		return candidateBackends[0]
	case 1:
		for backend := range runtimeMatches {
			return backend
		}
	}

	if backend, ok := firstRuntimeMatch(runtimeMatches, ctx.activeBackend, c.config.DefaultBackend, model.BackendTmux); ok {
		return backend
	}
	if runtimeMatches[model.BackendTmux] {
		return model.BackendTmux
	}
	return model.BackendZmx
}

func runtimeMatchesByBackend(candidateBackends []model.Backend, candidates []model.Connection) map[model.Backend]bool {
	matches := make(map[model.Backend]bool)
	for _, candidate := range candidates {
		if !isRuntimeSource(candidate.Session.Src) {
			continue
		}

		backend := candidate.Session.Backend
		if backend == "" {
			backend = backendForRuntimeSource(candidate.Session.Src)
		}
		if containsBackend(candidateBackends, backend) {
			matches[backend] = true
		}
	}
	return matches
}

func firstRuntimeMatch(matches map[model.Backend]bool, preferred ...model.Backend) (model.Backend, bool) {
	for _, backend := range preferred {
		if backend == "" {
			continue
		}
		if matches[backend] {
			return backend, true
		}
	}
	return "", false
}

func firstMatchingBackend(candidateBackends []model.Backend, preferred ...model.Backend) (model.Backend, bool) {
	for _, backend := range preferred {
		if backend == "" {
			continue
		}
		if containsBackend(candidateBackends, backend) {
			return backend, true
		}
	}
	return "", false
}

func containsBackend(backends []model.Backend, target model.Backend) bool {
	for _, backend := range backends {
		if backend == target {
			return true
		}
	}
	return false
}

func selectConnection(name, sourceHint string, backend model.Backend, candidates []model.Connection) (model.Connection, error) {
	filtered := filterBySourceHint(candidates, sourceHint)
	if sourceHint != "" && len(filtered) == 0 {
		return model.Connection{}, fmt.Errorf("selected source %q is unavailable for '%s'", sourceHint, name)
	}

	runtimeSource := runtimeSourceForBackend(backend)
	if chosen, found, err := chooseFromTier(name, backend, filtered, []string{runtimeSource}); err != nil {
		return model.Connection{}, err
	} else if found {
		return chosen, nil
	}

	for _, tier := range nonRuntimeSourcePrecedence(backend) {
		chosen, found, err := chooseFromTier(name, backend, filtered, tier)
		if err != nil {
			return model.Connection{}, err
		}
		if found {
			return chosen, nil
		}
	}

	return model.Connection{}, fmt.Errorf("no compatible connection found for '%s' with backend %s", name, backend)
}

func filterBySourceHint(candidates []model.Connection, sourceHint string) []model.Connection {
	if sourceHint == "" {
		return candidates
	}

	filtered := make([]model.Connection, 0, len(candidates))
	for _, candidate := range candidates {
		if sourceMatchesHint(candidate.Session.Src, sourceHint) {
			filtered = append(filtered, candidate)
		}
	}
	return filtered
}

func sourceMatchesHint(source, hint string) bool {
	if source == hint {
		return true
	}
	return hint == "config" && source == "config_wildcard"
}

func chooseFromTier(name string, backend model.Backend, candidates []model.Connection, allowedSources []string) (model.Connection, bool, error) {
	matches := make([]model.Connection, 0, len(candidates))
	for _, candidate := range candidates {
		if !containsSource(allowedSources, candidate.Session.Src) {
			continue
		}
		if !isSourceCompatible(candidate.Session.Src, backend) {
			continue
		}
		matches = append(matches, candidate)
	}

	if len(matches) == 0 {
		return model.Connection{}, false, nil
	}
	if len(matches) > 1 {
		return model.Connection{}, false, fmt.Errorf("ambiguous connection for '%s' in sources [%s]", name, strings.Join(allowedSources, ","))
	}

	return matches[0], true, nil
}

func containsSource(sources []string, target string) bool {
	for _, source := range sources {
		if source == target {
			return true
		}
	}
	return false
}

func runtimeSourceForBackend(backend model.Backend) string {
	if backend == model.BackendZmx {
		return "zmx"
	}
	return "tmux"
}

func nonRuntimeSourcePrecedence(backend model.Backend) [][]string {
	if backend == model.BackendZmx {
		return [][]string{{"config", "config_wildcard"}, {"dir"}, {"zoxide"}}
	}
	return [][]string{{"tmuxinator"}, {"config", "config_wildcard"}, {"dir"}, {"zoxide"}}
}

func isRuntimeSource(source string) bool {
	return source == "tmux" || source == "zmx"
}

func backendForRuntimeSource(source string) model.Backend {
	if source == "zmx" {
		return model.BackendZmx
	}
	return model.BackendTmux
}

func isSourceCompatible(source string, backend model.Backend) bool {
	switch source {
	case "tmux", "tmuxinator":
		return backend == model.BackendTmux
	case "zmx":
		return backend == model.BackendZmx
	case "config", "config_wildcard", "dir", "zoxide":
		return backend == model.BackendTmux || backend == model.BackendZmx
	default:
		return false
	}
}
