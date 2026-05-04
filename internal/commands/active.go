package commands

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/Proactive-Software/pwf-cli/internal/config"
	"github.com/Proactive-Software/pwf-cli/internal/state"
)

var activeCmd = &cobra.Command{
	Use:   "active",
	Short: "Show the currently active item and its workstage",
	Long:  "Fetch the active item from the API and show its current workstage, project, and phase.",
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := state.Load()
		if err != nil {
			if errors.Is(err, state.ErrNoActive) {
				fmt.Println("No active item. Run `pwf start`.")
				return nil
			}
			return err
		}

		client, err := newClient()
		if err != nil {
			return err
		}

		detail, err := client.GetItem(a.ID)
		if err != nil {
			return err
		}

		ws, _ := config.LoadWorkstages()
		stageName := ws.Name(detail.WorkstageID)
		if stageName == "" {
			stageName = fmt.Sprintf("stage:%d", detail.WorkstageID)
		}

		code := detail.Code
		if code == "" {
			code = fmt.Sprintf("#%d", detail.ID)
		}

		fmt.Printf("%s  %s\n",
			codeStyle.Render(code),
			titleStyle.Render(detail.Name),
		)
		fmt.Printf("   %s · %s · %s\n",
			projectStyle.Render(detail.ProjectTitle),
			phaseStyle.Render(detail.PhaseName),
			stageStyle.Render(stageName),
		)
		return nil
	},
}

var clearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Unset the active item",
	Long:  "Unset the active item. Run 'pwf start' to set a new one.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := state.Clear(); err != nil {
			return err
		}
		fmt.Println("Active item cleared.")
		return nil
	},
}
