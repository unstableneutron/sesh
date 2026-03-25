package lister

import (
	"errors"
	"fmt"
	"os/exec"

	"github.com/joshmedeski/sesh/v2/model"
)

var errZmxUnavailable = errors.New("zmx backend unavailable")

func zmxKey(name string) string {
	return fmt.Sprintf("zmx:%s", name)
}

func listZmx(l *RealLister) (model.SeshSessions, error) {
	if l.zmx == nil {
		return model.SeshSessions{}, errZmxUnavailable
	}

	zmxSessions, err := l.zmx.ListSessions()
	if err != nil {
		return model.SeshSessions{}, fmt.Errorf("couldn't list zmx sessions: %w", err)
	}

	numZmxSessions := len(zmxSessions)
	orderedIndex := make([]string, numZmxSessions)
	directory := make(model.SeshSessionMap)

	for i, session := range zmxSessions {
		key := zmxKey(session.Name)
		orderedIndex[i] = key
		directory[key] = model.SeshSession{
			Src:      "zmx",
			Backend:  model.BackendZmx,
			Name:     session.Name,
			Path:     session.StartDir,
			Attached: session.Clients,
		}
	}

	return model.SeshSessions{Directory: directory, OrderedIndex: orderedIndex}, nil
}

func isZmxUnavailableError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, errZmxUnavailable) {
		return true
	}

	var execErr *exec.Error
	if errors.As(err, &execErr) {
		return errors.Is(execErr.Err, exec.ErrNotFound)
	}

	return false
}

func (l *RealLister) FindZmxSession(name string) (model.SeshSession, bool) {
	sessions, err := listZmx(l)
	if err != nil {
		return model.SeshSession{}, false
	}

	key := zmxKey(name)
	session, exists := sessions.Directory[key]
	if !exists {
		return model.SeshSession{}, false
	}

	return session, true
}

func (l *RealLister) GetAttachedZmxSession() (model.SeshSession, bool) {
	if l.zmx == nil || !l.zmx.IsAttached() {
		return model.SeshSession{}, false
	}

	name := l.zmx.CurrentSessionName()
	if name == "" {
		return model.SeshSession{}, false
	}

	if session, exists := l.FindZmxSession(name); exists {
		return session, true
	}

	return model.SeshSession{Src: "zmx", Backend: model.BackendZmx, Name: name}, true
}
