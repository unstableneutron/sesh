package seshcli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/joshmedeski/sesh/v2/lister"
	"github.com/joshmedeski/sesh/v2/model"
)

func NewListCommand(base *BaseDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"l"},
		Short:   "List sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			deps, err := buildDeps(cmd, base)
			if err != nil {
				return err
			}
			if deps.CachingLister != nil {
				defer deps.CachingLister.Wait()
			}

			config, _ := cmd.Flags().GetBool("config")
			jsonOutput, _ := cmd.Flags().GetBool("json")
			tmux, _ := cmd.Flags().GetBool("tmux")
			zmx, _ := cmd.Flags().GetBool("zmx")
			zoxide, _ := cmd.Flags().GetBool("zoxide")
			hideAttached, _ := cmd.Flags().GetBool("hide-attached")
			icons, _ := cmd.Flags().GetBool("icons")
			noColor, _ := cmd.Flags().GetBool("no-color")
			tmuxinator, _ := cmd.Flags().GetBool("tmuxinator")
			hideDuplicates, _ := cmd.Flags().GetBool("hide-duplicates")

			sessions, err := deps.Lister.List(lister.ListOptions{
				Config:         config,
				HideAttached:   hideAttached,
				Icons:          icons,
				NoColor:        noColor,
				Json:           jsonOutput,
				Tmux:           tmux,
				Zmx:            zmx,
				Zoxide:         zoxide,
				Tmuxinator:     tmuxinator,
				HideDuplicates: hideDuplicates,
			})
			if err != nil {
				return fmt.Errorf("couldn't list sessions: %q", err)
			}

			if jsonOutput {
				var sessionsArray []model.SeshSession
				for _, i := range sessions.OrderedIndex {
					sessionsArray = append(sessionsArray, sessions.Directory[i])
				}
				fmt.Println(base.Json.EncodeSessions(sessionsArray))
				return nil
			}

			lines := formatListOutput(sessions, icons)
			for i, index := range sessions.OrderedIndex {
				name := lines[i]
				if icons {
					session := sessions.Directory[index]
					session.Name = name
					if noColor {
						name = deps.Icon.AddIconNoColor(session)
					} else {
						name = deps.Icon.AddIcon(session)
					}
				}
				fmt.Println(name)
			}

			return nil
		},
	}

	cmd.Flags().BoolP("config", "c", false, "show configured sessions")
	cmd.Flags().BoolP("json", "j", false, "output as json")
	cmd.Flags().BoolP("tmux", "t", false, "show tmux sessions")
	cmd.Flags().BoolP("zmx", "x", false, "show zmx sessions")
	cmd.Flags().BoolP("zoxide", "z", false, "show zoxide results")
	cmd.Flags().BoolP("hide-attached", "H", false, "don't show currently attached sessions")
	cmd.Flags().BoolP("icons", "i", false, "show icons")
	cmd.Flags().BoolP("no-color", "n", false, "show icons without color (requires --icons)")
	cmd.Flags().BoolP("tmuxinator", "T", false, "show tmuxinator configs")
	cmd.Flags().BoolP("hide-duplicates", "d", false, "hide duplicate entries")

	return cmd
}

func formatListOutput(sessions model.SeshSessions, _ bool) []string {
	collisionNames := backendCollisionNames(sessions)
	lines := make([]string, 0, len(sessions.OrderedIndex))
	for _, index := range sessions.OrderedIndex {
		session := sessions.Directory[index]
		name := session.Name
		if collisionNames[session.Name] && session.Backend != "" {
			name = fmt.Sprintf("%s [%s]", session.Name, session.Backend)
		}
		lines = append(lines, name)
	}
	return lines
}

func backendCollisionNames(sessions model.SeshSessions) map[string]bool {
	backendByName := make(map[string]map[model.Backend]struct{})
	for _, index := range sessions.OrderedIndex {
		session := sessions.Directory[index]
		if session.Backend == "" {
			continue
		}
		if _, ok := backendByName[session.Name]; !ok {
			backendByName[session.Name] = make(map[model.Backend]struct{})
		}
		backendByName[session.Name][session.Backend] = struct{}{}
	}

	collisionNames := make(map[string]bool)
	for name, backends := range backendByName {
		if len(backends) > 1 {
			collisionNames[name] = true
		}
	}

	return collisionNames
}
