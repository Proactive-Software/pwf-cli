package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search for items by name across the account",
	Long: `Search active items by name across the entire account.
Selecting an item sets it as active.`,
	Example: `  pwf search "login bug"
  pwf search "sprint 51"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}

		items, err := client.SearchItems(args[0])
		if err != nil {
			return err
		}

		if len(items) == 0 {
			fmt.Printf("No items found matching %q.\n", args[0])
			return nil
		}

		return pickAndActivate(client, items, false)
	},
}
