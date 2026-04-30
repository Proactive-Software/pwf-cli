package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var Root = &cobra.Command{
	Use:   "pwf",
	Short: "ProWorkflow CLI",
	Long: `pwf — personal ProWorkflow CLI.

Gets you to the right place fast without switching to the browser.
Default command: list your active items.

Configuration:
  ~/.config/pwf/config.toml    API base URL, app URL, subdomain
  ~/.config/pwf/workstages.toml  workstage name→ID map (auto-synced)
  OS keyring                    JWT token (set via pwf init)

Quick start:
  pwf init          one-time setup
  pwf start         pick active item
  pwf               list your items
  pwf stage done    move active item to a workstage
  pwf open          open active item in browser`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return listCmd.RunE(cmd, args)
	},
}

func init() {
	Root.AddCommand(listCmd)
	Root.AddCommand(startCmd)
	Root.AddCommand(activeCmd)
	Root.AddCommand(clearCmd)
	Root.AddCommand(tokenCmd)
	Root.AddCommand(stageCmd)
	Root.AddCommand(openCmd)
	Root.AddCommand(newCmd)
	Root.AddCommand(initCmd)
}

func die(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	os.Exit(1)
}
