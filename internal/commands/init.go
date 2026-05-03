package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/pkg/browser"
	"github.com/spf13/cobra"
	"bitbucket.org/proworkflow/pwf-cli/internal/api"
	"bitbucket.org/proworkflow/pwf-cli/internal/auth"
	"bitbucket.org/proworkflow/pwf-cli/internal/config"
	"bitbucket.org/proworkflow/pwf-cli/internal/keyring"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "One-time setup: authenticate via browser OAuth flow",
	RunE: func(cmd *cobra.Command, args []string) error {
		r := bufio.NewReader(os.Stdin)

		fmt.Print("API base URL [https://api.proworkflow.com/api/v4]: ")
		apiBase, _ := r.ReadString('\n')
		apiBase = strings.TrimSpace(apiBase)
		if apiBase == "" {
			apiBase = "https://api.proworkflow.com/api/v4"
		}

		fmt.Print("App base URL [https://app.proworkflow.com]: ")
		appBase, _ := r.ReadString('\n')
		appBase = strings.TrimSpace(appBase)
		if appBase == "" {
			appBase = "https://app.proworkflow.com"
		}

		fmt.Print("Account subdomain (e.g. pwfdevwork): ")
		subdomain, _ := r.ReadString('\n')
		subdomain = strings.TrimSpace(subdomain)

		fmt.Print("OAuth client ID: ")
		clientID, _ := r.ReadString('\n')
		clientID = strings.TrimSpace(clientID)
		if clientID == "" {
			return fmt.Errorf("OAuth client ID required")
		}

		fmt.Print("OAuth client secret: ")
		clientSecret, _ := r.ReadString('\n')
		clientSecret = strings.TrimSpace(clientSecret)
		if clientSecret == "" {
			return fmt.Errorf("OAuth client secret required")
		}

		fmt.Println("\nStarting browser login...")
		tokens, err := auth.LoginWithBrowserOpener(clientID, clientSecret, browser.OpenURL)
		if err != nil {
			return fmt.Errorf("OAuth login: %w", err)
		}

		tokensJSON, err := tokens.Marshal()
		if err != nil {
			return fmt.Errorf("marshal tokens: %w", err)
		}
		if err := keyring.SetTokens(tokensJSON); err != nil {
			return fmt.Errorf("store tokens: %w", err)
		}

		cfg := &config.Config{
			APIBase:           apiBase,
			AppBase:           appBase,
			Subdomain:         subdomain,
			OAuthClientID:     clientID,
			OAuthClientSecret: clientSecret,
		}
		if err := cfg.Save(); err != nil {
			return err
		}

		fmt.Printf("Config saved to %s\n", config.Path())
		fmt.Println("Tokens stored in OS keyring.")

		fmt.Print("Syncing workstages... ")
		client := api.New(apiBase, tokens.AccessToken)
		if err := syncWorkstages(client); err != nil {
			fmt.Printf("failed (%v)\n", err)
		} else {
			fmt.Println("done.")
		}

		return nil
	},
}
