package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	APIBase         string `toml:"api_base"`
	AppBase         string `toml:"app_base"`
	Subdomain       string `toml:"subdomain"`
	OAuthClientID   string `toml:"oauth_client_id"`
	OAuthClientSecret string `toml:"oauth_client_secret"`
}

func Dir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "pwf")
}

func Path() string {
	return filepath.Join(Dir(), "config.toml")
}

func Load() (*Config, error) {
	cfg := &Config{}
	if _, err := toml.DecodeFile(Path(), cfg); err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}
	return cfg, nil
}

func (c *Config) Save() error {
	if err := os.MkdirAll(Dir(), 0700); err != nil {
		return fmt.Errorf("mkdir config dir: %w", err)
	}
	f, err := os.Create(Path())
	if err != nil {
		return fmt.Errorf("create config: %w", err)
	}
	defer func() { _ = f.Close() }()
	return toml.NewEncoder(f).Encode(c)
}
