package commands

import (
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strconv"

	"github.com/spf13/cobra"
	"bitbucket.org/proworkflow/pwf-cli/internal/state"
)

var tokenCmd = &cobra.Command{
	Use:   "token [id]",
	Short: "Print the uniqueToken and copy it to clipboard",
	Long: `Print the uniqueToken of the active item and copy it to the clipboard.

If an item ID is provided, fetches the token via the API.
If no ID is given, uses the cached token from the active item (no API call).`,
	Example: `  pwf token
  pwf token 2151`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var token string

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
			token = detail.UniqueToken
		} else {
			a, err := state.Load()
			if err != nil {
				if errors.Is(err, state.ErrNoActive) {
					return fmt.Errorf("no active item (run `pwf start` or provide an item ID)")
				}
				return err
			}
			if a.UniqueToken == "" {
				client, err := newClient()
				if err != nil {
					return err
				}
				detail, err := client.GetItem(a.ID)
				if err != nil {
					return err
				}
				token = detail.UniqueToken
			} else {
				token = a.UniqueToken
			}
		}

		fmt.Println(token)
		if err := copyToClipboard(token); err == nil {
			fmt.Println("(copied to clipboard)")
		}
		return nil
	},
}

func copyToClipboard(s string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		cmd = exec.Command("xclip", "-selection", "clipboard")
	default:
		return fmt.Errorf("unsupported platform")
	}
	pipe, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	fmt.Fprint(pipe, s)
	pipe.Close()
	return cmd.Wait()
}
