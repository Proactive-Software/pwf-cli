package commands

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/Proactive-Software/pwf-cli/internal/state"
)

var inboxCmd = &cobra.Command{
	Use:   "inbox",
	Short: "Pick an unassigned item in the active project, assign yourself, and set it active",
	Long: `Shows unassigned active items in the current project.
Selecting an item assigns it to you and sets it as active.`,
	Example: `  pwf inbox`,
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := state.Load()
		if err != nil {
			if errors.Is(err, state.ErrNoActive) {
				return fmt.Errorf("no active item (run `pwf start` to set project context)")
			}
			return err
		}

		client, err := newClient()
		if err != nil {
			return err
		}

		items, err := client.InboxItems(a.ProjectID)
		if err != nil {
			return err
		}

		return pickAndActivate(client, items, true)
	},
}
