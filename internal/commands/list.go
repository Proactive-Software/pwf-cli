package commands

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"bitbucket.org/proworkflow/pwf-cli/internal/api"
	"bitbucket.org/proworkflow/pwf-cli/internal/config"
)

var (
	codeStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("99")).Bold(true)
	titleStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
	projectStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	phaseStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("243")).Italic(true)
	stageStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
)

var listAll bool

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List your active items",
	Long: `List active items assigned to you.

Use --all to browse all active items in the account (useful for picking up
unassigned work from the sprint).`,
	Example: `  pwf list
  pwf list --all`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}

		var items []api.Item
		if listAll {
			items, err = client.ActiveItems()
		} else {
			items, err = client.MyItems()
		}
		if err != nil {
			return err
		}

		if len(items) == 0 {
			fmt.Println("No active items.")
			return nil
		}

		ws, _ := config.LoadWorkstages()

		for _, it := range items {
			code := it.Code
			if code == "" {
				code = fmt.Sprintf("#%d", it.ID)
			}
			stage := ws.Name(it.WorkstageID)
			if stage == "" {
				stage = fmt.Sprintf("stage:%d", it.WorkstageID)
			}
			fmt.Printf("%s  %s\n",
				codeStyle.Render(code),
				titleStyle.Render(it.Name),
			)
			fmt.Printf("   %s · %s · %s\n",
				projectStyle.Render(it.ProjectTitle),
				phaseStyle.Render(it.PhaseName),
				stageStyle.Render(stage),
			)
		}
		return nil
	},
}

func init() {
	listCmd.Flags().BoolVar(&listAll, "all", false, "Show all active items (not just mine)")
	listCmd.Aliases = []string{"ls"}
}
