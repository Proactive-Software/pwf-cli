package commands

import (
	"errors"
	"fmt"
	"strings"

	htmlmd "github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/charmbracelet/glamour"
	"github.com/spf13/cobra"
	"github.com/Proactive-Software/pwf-cli/internal/state"
)

var descCmd = &cobra.Command{
	Use:   "desc",
	Short: "Show the active item's description",
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := state.Load()
		if err != nil {
			if errors.Is(err, state.ErrNoActive) {
				return fmt.Errorf("no active item (run `pwf start`)")
			}
			return err
		}

		client, err := newClient()
		if err != nil {
			return err
		}

		code := a.Code
		if code == "" {
			code = fmt.Sprintf("#%d", a.ID)
		}
		fmt.Printf("%s  %s\n\n", codeStyle.Render(code), titleStyle.Render(a.Title))

		detail, err := client.GetItem(a.ID)
		if err != nil {
			return fmt.Errorf("fetch item: %w", err)
		}

		if strings.TrimSpace(detail.Description) == "" {
			fmt.Println(projectStyle.Render("  No description."))
			return nil
		}

		md, err := htmlmd.ConvertString(detail.Description)
		if err != nil {
			fmt.Println(detail.Description)
			return nil
		}

		rendered, err := glamour.Render(md, "dark")
		if err != nil {
			fmt.Println(md)
			return nil
		}

		fmt.Print(rendered)
		return nil
	},
}
