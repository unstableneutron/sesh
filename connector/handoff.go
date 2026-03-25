package connector

import (
	"errors"
	"fmt"
	"strings"

	"github.com/joshmedeski/sesh/v2/model"
)

func buildReplayConnectCommand(name string, backend model.Backend, opts model.ConnectOpts, includeBypass bool) string {
	replayName := name
	if opts.ReplayName != "" {
		replayName = opts.ReplayName
	}

	parts := []string{"sesh", "connect"}
	if opts.ConfigPath != "" {
		parts = append(parts, "--config", shellQuote(opts.ConfigPath))
	}
	if backend != "" {
		parts = append(parts, "--backend", string(backend))
	}
	if opts.Switch {
		parts = append(parts, "--switch")
	}
	if opts.Command != "" {
		parts = append(parts, "--command", shellQuote(opts.Command))
	}
	if opts.Tmuxinator {
		parts = append(parts, "--tmuxinator")
	}
	if opts.SourceHint != "" {
		parts = append(parts, "--"+model.SourceHintFlag, opts.SourceHint)
	}
	if includeBypass {
		parts = append(parts, "--"+model.BypassHandoffFlag)
	}

	parts = append(parts, "--", shellQuote(replayName))
	return strings.Join(parts, " ")
}

func queueKittyHandoff(c *RealConnector, targetName string, backend model.Backend, opts model.ConnectOpts) (string, error) {
	if c.kitty == nil {
		return "", manualZmxHandoffError("kitty integration is not configured", targetName, backend, opts, nil)
	}
	if !c.kitty.CanRemoteControl() {
		return "", manualZmxHandoffError("kitty remote control environment is missing", targetName, backend, opts, nil)
	}
	if opts.KittyWindowID == "" {
		return "", manualZmxHandoffError("KITTY_WINDOW_ID is missing", targetName, backend, opts, nil)
	}

	replayCommand := buildReplayConnectCommand(targetName, backend, opts, true)
	if err := c.kitty.SendDetach(opts.KittyWindowID); err != nil {
		return "", manualZmxHandoffError("failed to send kitty detach key", targetName, backend, opts, err)
	}
	if err := c.kitty.QueueCommand(opts.KittyWindowID, replayCommand); err != nil {
		return "", manualZmxHandoffError("failed to queue kitty reconnect command", targetName, backend, opts, err)
	}

	return fmt.Sprintf("queued kitty handoff to %s backend session: %s", backend, targetName), nil
}

func manualZmxHandoffError(reason, targetName string, backend model.Backend, opts model.ConnectOpts, cause error) error {
	reconnect := buildReplayConnectCommand(targetName, backend, opts, false)
	message := fmt.Sprintf("automatic zmx handoff is unavailable (%s). Press Ctrl+\\ to detach, then run: %s", reason, reconnect)
	if cause != nil {
		return fmt.Errorf("%s: %w", message, cause)
	}
	return errors.New(message)
}

func manualTmuxToZmxError(targetName string, opts model.ConnectOpts) error {
	reconnect := buildReplayConnectCommand(targetName, model.BackendZmx, opts, false)
	return fmt.Errorf("no automatic handoff from tmux to zmx in MVP; detach tmux manually, then run: %s", reconnect)
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}
