package commands

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/Proactive-Software/pwf-cli/internal/config"
	"github.com/Proactive-Software/pwf-cli/internal/state"
)

func completeWorkstages(_ *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	ws, err := config.LoadWorkstages()
	if err != nil || len(ws.Stages) == 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	q := strings.ToLower(toComplete)
	var matches []string
	for _, name := range ws.UniqueNames() {
		if strings.HasPrefix(strings.ToLower(name), q) {
			matches = append(matches, name)
		}
	}
	return matches, cobra.ShellCompDirectiveNoFileComp
}

var stageCmd = &cobra.Command{
	Use:   "stage <name>",
	Short: "Move active item to a workstage (fuzzy matched)",
	Long: `Move the active item to a workstage by name.

Names are fuzzy matched — a prefix or substring is enough. Workstages
are synced from the API via 'pwf sync' or 'pwf init'.`,
	Example: `  pwf stage "in progress"
  pwf stage testing       # matches "Needs Testing"
  pwf stage done`,
	Args:              cobra.MinimumNArgs(1),
	ValidArgsFunction: completeWorkstages,
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := state.Load()
		if err != nil {
			if errors.Is(err, state.ErrNoActive) {
				return fmt.Errorf("no active item (run `pwf start`)")
			}
			return err
		}

		query := strings.Join(args, " ")
		ws, err := config.LoadWorkstages()
		if err != nil {
			return err
		}
		if len(ws.Stages) == 0 {
			return fmt.Errorf("no workstages configured (edit %s)", config.WorkstagesPath())
		}

		id, name, ok := ws.FuzzyMatch(query)
		if !ok {
			fmt.Printf("No workstage matching %q. Available:\n", query)
			for _, n := range ws.UniqueNames() {
				fmt.Printf("  %s\n", n)
			}
			return nil
		}

		client, err := newClient()
		if err != nil {
			return err
		}

		if err := client.SetWorkstage(a.ID, id); err != nil {
			return fmt.Errorf("update workstage: %w", err)
		}

		code := a.Code
		if code == "" {
			code = fmt.Sprintf("#%d", a.ID)
		}
		fmt.Printf("%s → %s\n", titleStyle.Render(a.Title), titleStyle.Render(name))
		return nil
	},
}
