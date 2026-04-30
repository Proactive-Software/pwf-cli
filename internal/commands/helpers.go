package commands

import (
	"fmt"

	"bitbucket.org/proworkflow/pwf-cli/internal/api"
	"bitbucket.org/proworkflow/pwf-cli/internal/auth"
	"bitbucket.org/proworkflow/pwf-cli/internal/config"
	"bitbucket.org/proworkflow/pwf-cli/internal/keyring"
)

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
