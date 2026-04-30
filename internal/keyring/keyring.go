package keyring

import (
	"fmt"

	"github.com/zalando/go-keyring"
)

const service = "pwf-cli"
const tokensKey = "oauth-tokens"

func SetTokens(json string) error {
	if err := keyring.Set(service, tokensKey, json); err != nil {
		return fmt.Errorf("store tokens: %w", err)
	}
	return nil
}

func GetTokens() (string, error) {
	val, err := keyring.Get(service, tokensKey)
	if err != nil {
		return "", fmt.Errorf("get tokens (run `pwf init`): %w", err)
	}
	return val, nil
}

func Delete() error {
	return keyring.Delete(service, tokensKey)
}
