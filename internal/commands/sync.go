package commands

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"bitbucket.org/proworkflow/pwf-cli/internal/api"
	"bitbucket.org/proworkflow/pwf-cli/internal/config"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync workstages from the API",
	Long: `Pull the latest workstage list from the API and update ~/.config/pwf/workstages.toml.

Run this whenever workstages are added or renamed in ProWorkflow.
Also runs automatically during 'pwf init'.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		if err := syncWorkstages(client); err != nil {
			return err
		}
		ws, _ := config.LoadWorkstages()
		fmt.Printf("Synced %d workstages to %s\n", len(ws.Stages), config.WorkstagesPath())
		for name, id := range ws.Stages {
			fmt.Printf("  %s  %s\n", id, name)
		}
		return nil
	},
}

func syncWorkstages(client *api.Client) error {
	stages, err := client.GetItemWorkstages()
	if err != nil {
		return err
	}
	ws := &config.Workstages{Stages: make(map[string]string)}
	for _, s := range stages {
		ws.Stages[strconv.Itoa(s.ID)] = s.Name
	}
	return ws.Save()
}

func init() {
	Root.AddCommand(syncCmd)
}
