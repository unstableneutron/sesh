package seshcli

import (
	"testing"

	"github.com/joshmedeski/sesh/v2/model"
	"github.com/stretchr/testify/assert"
)

func TestListOutput_DuplicateNames_ShowBackendTag(t *testing.T) {
	sessions := model.SeshSessions{
		OrderedIndex: []string{"tmux:work", "zmx:work", "config:docs"},
		Directory: model.SeshSessionMap{
			"tmux:work": {
				Src:     "tmux",
				Backend: model.BackendTmux,
				Name:    "work",
			},
			"zmx:work": {
				Src:     "zmx",
				Backend: model.BackendZmx,
				Name:    "work",
			},
			"config:docs": {
				Src:  "config",
				Name: "docs",
			},
		},
	}

	lines := formatListOutput(sessions, false)
	assert.Contains(t, lines, "work [tmux]")
	assert.Contains(t, lines, "work [zmx]")
	assert.Contains(t, lines, "docs")
}
