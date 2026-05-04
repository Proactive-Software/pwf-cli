package commands

import (
	"fmt"

	"github.com/Proactive-Software/pwf-cli/internal/api"
	"github.com/Proactive-Software/pwf-cli/internal/auth"
	"github.com/Proactive-Software/pwf-cli/internal/config"
	"github.com/Proactive-Software/pwf-cli/internal/keyring"
	"github.com/Proactive-Software/pwf-cli/internal/picker"
	"github.com/Proactive-Software/pwf-cli/internal/state"
)

func contactSummary(contacts []api.ItemContact) string {
	if len(contacts) == 0 {
		return "Unassigned"
	}
	name := contacts[0].FirstName + " " + contacts[0].LastName
	if len(contacts) == 1 {
		return name
	}
	return fmt.Sprintf("%s + %d other(s)", name, len(contacts)-1)
}

// pickAndActivate runs the fuzzy picker over items, optionally assigns the
// current user, fetches full detail, and saves as the active item.
func pickAndActivate(client *api.Client, items []api.Item, assign bool) error {
	if len(items) == 0 {
		fmt.Println("No items found.")
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
			Sub:     fmt.Sprintf("%s · %s · %s · %s", it.ProjectTitle, it.PhaseName, stage, contactSummary(it.Contacts)),
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

	if assign {
		me, err := client.Me()
		if err != nil {
			return fmt.Errorf("get my contact: %w", err)
		}
		contactID, ok := me["id"].(float64)
		if !ok {
			return fmt.Errorf("unexpected contact ID type")
		}
		if err := client.AssignContact(chosen.ID, int(contactID)); err != nil {
			return fmt.Errorf("assign contact: %w", err)
		}
	}

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
	return nil
}

func newClient() (*api.Client, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	if cfg.APIBase == "" {
		return nil, fmt.Errorf("no API base URL (run `pwf init`)")
	}

	raw, err := keyring.GetTokens()
	if err != nil {
		return nil, err
	}

	tokens, err := auth.UnmarshalTokens(raw)
	if err != nil {
		return nil, fmt.Errorf("parse stored tokens: %w", err)
	}

	if tokens.IsExpired() && tokens.RefreshToken != "" {
		if cfg.OAuthClientID == "" {
			return nil, fmt.Errorf("access token expired but no OAuth client configured (run `pwf init`)")
		}
		tokens, err = auth.Refresh(cfg.OAuthClientID, cfg.OAuthClientSecret, tokens.RefreshToken)
		if err != nil {
			return nil, fmt.Errorf("refresh token: %w — run `pwf init` to re-authenticate", err)
		}
		tokensJSON, err := tokens.Marshal()
		if err != nil {
			return nil, fmt.Errorf("marshal refreshed tokens: %w", err)
		}
		if err := keyring.SetTokens(tokensJSON); err != nil {
			return nil, fmt.Errorf("store refreshed tokens: %w", err)
		}
	}

	return api.New(cfg.APIBase, tokens.AccessToken), nil
}
