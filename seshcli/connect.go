package seshcli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/joshmedeski/sesh/v2/lister"
	"github.com/joshmedeski/sesh/v2/model"
)

func NewConnectCommand(base *BaseDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "connect",
		Aliases: []string{"cn"},
		Short:   "Connect to the given session",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("please provide a session name")
			}
			name := strings.Join(args, " ")
			if name == "" {
				return nil
			}

			backendValue, _ := cmd.Flags().GetString("backend")
			backend, err := parseBackendFlag(backendValue)
			if err != nil {
				return err
			}
			sourceHint, _ := cmd.Flags().GetString(model.SourceHintFlag)
			bypassHandoff, _ := cmd.Flags().GetBool(model.BypassHandoffFlag)
			configPath, _ := cmd.Root().PersistentFlags().GetString("config")

			deps, err := buildDeps(cmd, base)
			if err != nil {
				return err
			}

			switchFlag, _ := cmd.Flags().GetBool("switch")
			command, _ := cmd.Flags().GetString("command")
			tmuxinator, _ := cmd.Flags().GetBool("tmuxinator")
			root, _ := cmd.Flags().GetBool("root")

			if root {
				hasRootDir, rootDir := base.Dir.RootDir(name)
				if hasRootDir {
					name = rootDir
				}
			}

			trimmedName := deps.Icon.RemoveIcon(name)

			kittyWindowID := ""
			if base != nil && base.Os != nil {
				kittyWindowID = base.Os.Getenv("KITTY_WINDOW_ID")
			}

			opts := model.ConnectOpts{
				Switch:        switchFlag,
				Command:       command,
				Tmuxinator:    tmuxinator,
				Backend:       backend,
				SourceHint:    sourceHint,
				ReplayName:    trimmedName,
				ConfigPath:    configPath,
				KittyWindowID: kittyWindowID,
				BypassHandoff: bypassHandoff,
			}
			if _, err := deps.Connector.Connect(trimmedName, opts); err != nil {
				// TODO: add to logging
				return err
			}
			// Refresh cache in background so next sesh list has fresh data
			if deps.CachingLister != nil {
				deps.CachingLister.RefreshCache(lister.ListOptions{})
				deps.CachingLister.Wait()
			}
			return nil
		},
	}

	cmd.Flags().BoolP("switch", "s", false, "Switch the session (rather than attach). This is useful for actions triggered outside the terminal.")
	cmd.Flags().StringP("backend", "b", "", "Connect using a specific backend (tmux or zmx)")
	cmd.Flags().StringP("command", "c", "", "Execute a command when connecting to a new session. Will be ignored if the session exists.")
	cmd.Flags().BoolP("tmuxinator", "T", false, "Use tmuxinator to start session if it doesnt exist")
	cmd.Flags().BoolP("root", "r", false, "Switches to the root of the current session")
	cmd.Flags().String(model.SourceHintFlag, "", "Internal source hint for replayed connect commands")
	cmd.Flags().Bool(model.BypassHandoffFlag, false, "Internal one-shot guard to bypass zmx handoff replay")
	_ = cmd.Flags().MarkHidden(model.SourceHintFlag)
	_ = cmd.Flags().MarkHidden(model.BypassHandoffFlag)

	return cmd
}

func parseBackendFlag(raw string) (model.Backend, error) {
	if raw == "" {
		return "", nil
	}

	backend := model.Backend(raw)
	if !backend.IsValid() {
		return "", fmt.Errorf("invalid backend %q: allowed values are tmux or zmx", raw)
	}

	return backend, nil
}
