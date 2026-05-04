package commands

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/Proactive-Software/pwf-cli/internal/state"
)

var browseCmd = &cobra.Command{
	Use:   "browse",
	Short: "Pick any item in the active project and set it as active",
	Long: `Shows all active items in the current project regardless of assignment.
Useful for picking up work from your project's backlog.

Selecting an item sets it as active. If the item is unassigned, you will be
prompted to assign yourself.`,
	Example: `  pwf browse`,
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

		items, err := client.ProjectItems(a.ProjectID)
		if err != nil {
			return err
		}

		return pickAndActivate(client, items, false)
	},
}
