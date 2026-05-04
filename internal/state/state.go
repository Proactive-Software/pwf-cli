package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Proactive-Software/pwf-cli/internal/config"
)

// ActiveItem is persisted to ~/.config/pwf/active.json
type ActiveItem struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Code        string `json:"code"`
	UniqueToken string `json:"unique_token"`
	ProjectID   int    `json:"project_id"`
	ProjectName string `json:"project_name"`
	PhaseName   string `json:"phase_name"`
	PhaseID     int    `json:"phase_id"`
}

func activePath() string {
	return filepath.Join(config.Dir(), "active.json")
}

var ErrNoActive = errors.New("no active item (run `pwf start`)")

func Load() (*ActiveItem, error) {
	data, err := os.ReadFile(activePath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNoActive
		}
		return nil, fmt.Errorf("read active: %w", err)
	}
	var a ActiveItem
	if err := json.Unmarshal(data, &a); err != nil {
		return nil, fmt.Errorf("parse active: %w", err)
	}
	return &a, nil
}

func Save(a *ActiveItem) error {
	if err := os.MkdirAll(config.Dir(), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(a, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(activePath(), data, 0600)
}

func Clear() error {
	err := os.Remove(activePath())
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
