package commands

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/Proactive-Software/pwf-cli/internal/api"
	"github.com/Proactive-Software/pwf-cli/internal/config"
	"github.com/Proactive-Software/pwf-cli/internal/picker"
	"github.com/Proactive-Software/pwf-cli/internal/state"
)

var newPhase string
var newPickPhase bool

var newCmd = &cobra.Command{
	Use:   "new \"<title>\"",
	Short: "Create a new item in the active item's project and phase",
	Long: `Quick-create a new item in the same project as the active item.

By default inherits the active item's phase. Use --phase to fuzzy-match a
different phase by name, or --pick-phase to open an interactive picker.

The item is assigned to you automatically.`,
	Example: `  pwf new "Fix login redirect bug"
  pwf new "New task" --phase "disc"
  pwf new "New task" --pick-phase`,
	Args: cobra.ExactArgs(1),
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

		// Resolve phase
		phaseID := a.PhaseID
		phaseName := a.PhaseName

		if newPickPhase || newPhase != "" {
			phases, err := client.GetPhases(a.ProjectID)
			if err != nil {
				return fmt.Errorf("fetch phases: %w", err)
			}
			if len(phases) == 0 {
				return fmt.Errorf("no phases found for project %d", a.ProjectID)
			}

			if newPickPhase {
				chosen, err := pickPhase(phases)
				if err != nil {
					return err
				}
				if chosen == nil {
					fmt.Println("Cancelled.")
					return nil
				}
				phaseID = chosen.ID
				phaseName = chosen.Name
			} else {
				matched, ok := fuzzyPhase(phases, newPhase)
				if !ok {
					fmt.Printf("No phase matching %q. Available:\n", newPhase)
					for _, p := range phases {
						fmt.Printf("  %s\n", p.Name)
					}
					return nil
				}
				phaseID = matched.ID
				phaseName = matched.Name
			}
		}

		// Get my contact ID
		me, err := client.Me()
		if err != nil {
			return fmt.Errorf("get my contact: %w", err)
		}
		contactID, ok := me["id"].(float64)
		if !ok {
			return fmt.Errorf("unexpected contact ID type")
		}

		title := args[0]
		created, err := client.CreateItem(a.ProjectID, phaseID, int(contactID), title)
		if err != nil {
			return fmt.Errorf("create item: %w", err)
		}

		detail, err := client.GetItem(created.ID)
		if err != nil {
			return fmt.Errorf("fetch created item: %w", err)
		}

		ws, _ := config.LoadWorkstages()
		stage := ws.Name(detail.WorkstageID)
		if stage == "" {
			stage = fmt.Sprintf("stage:%d", detail.WorkstageID)
		}

		code := detail.Code
		if code == "" {
			code = fmt.Sprintf("#%d", detail.ID)
		}
		fmt.Printf("Created: %s  %s\n", codeStyle.Render(code), titleStyle.Render(detail.Name))
		fmt.Printf("   %s · %s · %s\n", projectStyle.Render(a.ProjectName), phaseStyle.Render(phaseName), phaseStyle.Render(stage))
		if detail.UniqueToken != "" {
			fmt.Printf("   token: %s\n", detail.UniqueToken)
		}
		fmt.Printf("   hint: pwf start to set as active\n")
		return nil
	},
}

func init() {
	newCmd.Flags().StringVar(&newPhase, "phase", "", "Phase name (fuzzy matched)")
	newCmd.Flags().BoolVar(&newPickPhase, "pick-phase", false, "Interactively pick a phase")
	_ = newCmd.RegisterFlagCompletionFunc("phase", completePhases)
}

func completePhases(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	a, err := state.Load()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	client, err := newClient()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	phases, err := client.GetPhases(a.ProjectID)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	q := strings.ToLower(toComplete)
	var matches []string
	for _, p := range phases {
		if toComplete == "" || strings.HasPrefix(strings.ToLower(p.Name), q) {
			matches = append(matches, p.Name)
		}
	}
	return matches, cobra.ShellCompDirectiveNoFileComp
}

func pickPhase(phases []api.Phase) (*api.Phase, error) {
	items := make([]picker.Item, len(phases))
	for i, p := range phases {
		items[i] = picker.Item{ID: p.ID, Display: p.Name}
	}
	chosen, err := picker.Run(items)
	if err != nil || chosen == nil {
		return nil, err
	}
	for _, p := range phases {
		if p.ID == chosen.ID {
			return &p, nil
		}
	}
	return nil, nil
}

func fuzzyPhase(phases []api.Phase, query string) (*api.Phase, bool) {
	q := strings.ToLower(query)
	// Exact
	for i, p := range phases {
		if strings.ToLower(p.Name) == q {
			return &phases[i], true
		}
	}
	// Prefix
	for i, p := range phases {
		if strings.HasPrefix(strings.ToLower(p.Name), q) {
			return &phases[i], true
		}
	}
	// Substring
	for i, p := range phases {
		if strings.Contains(strings.ToLower(p.Name), q) {
			return &phases[i], true
		}
	}
	return nil, false
}
