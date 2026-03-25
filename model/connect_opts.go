package model

const (
	BypassHandoffFlag = "_skip-zmx-handoff"
	SourceHintFlag    = "source-hint"
)

type ConnectOpts struct {
	Command       string
	Switch        bool
	Tmuxinator    bool
	Backend       Backend
	SourceHint    string
	ReplayName    string
	ConfigPath    string
	KittyWindowID string
	BypassHandoff bool
}
