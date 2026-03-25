package model

type ConnectOpts struct {
	Command       string
	Switch        bool
	Tmuxinator    bool
	Backend       Backend
	SourceHint    string
	BypassHandoff bool
}
