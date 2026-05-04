package commands

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/browser"
	"github.com/spf13/cobra"
	"github.com/Proactive-Software/pwf-cli/internal/config"
	"github.com/Proactive-Software/pwf-cli/internal/state"
)

var openProject bool

var openCmd = &cobra.Command{
	Use:   "open [id]",
	Short: "Open the active item (or given ID) in browser",
	Long: `Open the active item in your default browser.

Opens the project detail view with the item highlighted. Pass an item ID
to open a specific item without changing the active state. Use --project
to open the item's parent project instead.`,
	Example: `  pwf open
  pwf open 2151
  pwf open --project`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		if len(args) == 1 {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid item ID: %s", args[0])
			}
			client, err := newClient()
			if err != nil {
				return err
			}
			detail, err := client.GetItem(id)
			if err != nil {
				return err
			}
			u := itemURL(cfg.AppBase, cfg.Subdomain, detail.ProjectID, detail.ID)
			fmt.Println(u)
			return browser.OpenURL(u)
		}

		a, err := state.Load()
		if err != nil {
			if errors.Is(err, state.ErrNoActive) {
				return fmt.Errorf("no active item (run `pwf start`)")
			}
			return err
		}

		if openProject {
			u := projectURL(cfg.AppBase, cfg.Subdomain, a.ProjectID)
			fmt.Println(u)
			return browser.OpenURL(u)
		}

		u := itemURL(cfg.AppBase, cfg.Subdomain, a.ProjectID, a.ID)
		fmt.Println(u)
		return browser.OpenURL(u)
	},
}

func init() {
	openCmd.Flags().BoolVar(&openProject, "project", false, "Open the active item's project instead")
}

func appBase(cfg *config.Config) string {
	if cfg.AppBase != "" {
		return strings.TrimRight(cfg.AppBase, "/")
	}
	return "https://app.proworkflow.com"
}

func itemURL(base, subdomain string, projectID, itemID int) string {
	base = strings.TrimRight(base, "/")
	return fmt.Sprintf("%s/%s/?fuseaction=trackedprojects&fusesubaction=details&Jobs_currentJobID=%d&item=%d",
		base, subdomain, projectID, itemID)
}

func projectURL(base, subdomain string, projectID int) string {
	base = strings.TrimRight(base, "/")
	return fmt.Sprintf("%s/%s/?fuseaction=trackedprojects&fusesubaction=details&Jobs_currentJobID=%d",
		base, subdomain, projectID)
}

// openItem is used by start --open
func openItem(a *state.ActiveItem) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	u := itemURL(appBase(cfg), cfg.Subdomain, a.ProjectID, a.ID)
	fmt.Println(u)
	return browser.OpenURL(u)
}
