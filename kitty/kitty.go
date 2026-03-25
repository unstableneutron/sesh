package kitty

import (
	"fmt"

	"github.com/joshmedeski/sesh/v2/oswrap"
	"github.com/joshmedeski/sesh/v2/shell"
)

type Kitty interface {
	CanRemoteControl() bool
	SendDetach(windowID string) error
	QueueCommand(windowID string, command string) error
}

type RealKitty struct {
	os    oswrap.Os
	shell shell.Shell
}

func NewKitty(os oswrap.Os, shell shell.Shell) Kitty {
	return &RealKitty{os: os, shell: shell}
}

func (k *RealKitty) CanRemoteControl() bool {
	return k.os.Getenv("KITTY_WINDOW_ID") != "" &&
		k.os.Getenv("KITTY_LISTEN_ON") != "" &&
		k.os.Getenv("KITTY_PUBLIC_KEY") != ""
}

func (k *RealKitty) SendDetach(windowID string) error {
	if windowID == "" {
		return fmt.Errorf("kitty window id is required")
	}
	_, err := k.shell.Cmd("kitten", "@", "send-key", "--match", fmt.Sprintf("id:%s", windowID), "ctrl+backslash")
	if err != nil {
		return fmt.Errorf("failed to send kitty detach key: %w", err)
	}
	return nil
}

func (k *RealKitty) QueueCommand(windowID string, command string) error {
	if windowID == "" {
		return fmt.Errorf("kitty window id is required")
	}
	if command == "" {
		return fmt.Errorf("kitty queued command is required")
	}

	_, err := k.shell.Cmd("kitten", "@", "send-text", "--match", fmt.Sprintf("id:%s", windowID), command+"\n")
	if err != nil {
		return fmt.Errorf("failed to queue kitty command: %w", err)
	}
	return nil
}
