package commands

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/Proactive-Software/pwf-cli/internal/config"
	"github.com/Proactive-Software/pwf-cli/internal/picker"
	"github.com/Proactive-Software/pwf-cli/internal/state"
)

var startOpen bool

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Pick an item and set it as active",
	Long: `Opens an interactive fuzzy picker over your active items.
The selected item becomes the global active item — shared across all repos.

The active item's uniqueToken is cached at selection time so pwf token
and the git commit hook work instantly without extra API calls.`,
	Example: `  pwf start
  pwf start --open    # open in browser immediately after picking`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}

		items, err := client.MyItems()
		if err != nil {
			return err
		}
		if len(items) == 0 {
			fmt.Println("No active items to pick from.")
			return nil
		}

		ws, _ := config.LoadWorkstages()

		pickerItems := make([]picker.Item, len(items))
		for i, it := range items {
			code := it.Code
			if code == "" {
				code = fmt.Sprintf("#%d", it.ID)
			}
			stage := ws.Name(it.WorkstageID)
			if stage == "" {
				stage = fmt.Sprintf("stage:%d", it.WorkstageID)
			}
			pickerItems[i] = picker.Item{
				ID:      it.ID,
				Display: fmt.Sprintf("%s  %s", code, it.Name),
				Sub:     fmt.Sprintf("%s · %s · %s", it.ProjectTitle, it.PhaseName, stage),
			}
		}

		chosen, err := picker.Run(pickerItems)
		if err != nil {
			return err
		}
		if chosen == nil {
			fmt.Println("Cancelled.")
			return nil
		}

		// Fetch detail to get uniquetoken and full info
		detail, err := client.GetItem(chosen.ID)
		if err != nil {
			return fmt.Errorf("fetch item detail: %w", err)
		}

		active := &state.ActiveItem{
			ID:          detail.ID,
			Title:       detail.Name,
			Code:        detail.Code,
			UniqueToken: detail.UniqueToken,
			ProjectID:   detail.ProjectID,
			ProjectName: detail.ProjectTitle,
			PhaseName:   detail.PhaseName,
			PhaseID:     detail.ItemCollectionID,
		}
		if err := state.Save(active); err != nil {
			return fmt.Errorf("save active: %w", err)
		}

		code := active.Code
		if code == "" {
			code = fmt.Sprintf("#%d", active.ID)
		}
		fmt.Printf("Active: %s  %s\n", codeStyle.Render(code), titleStyle.Render(active.Title))

		if startOpen {
			return openItem(active)
		}
		return nil
	},
}

func init() {
	startCmd.Flags().BoolVar(&startOpen, "open", false, "Open in browser after selecting")
}
