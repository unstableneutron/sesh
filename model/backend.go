package model

type Backend string

const (
	BackendTmux Backend = "tmux"
	BackendZmx  Backend = "zmx"
)

func (b Backend) IsValid() bool {
	switch b {
	case BackendTmux, BackendZmx:
		return true
	default:
		return false
	}
}
