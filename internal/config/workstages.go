package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
)

// Workstages stores id→name. Keys are string representations of the int ID
// because TOML requires string keys.
type Workstages struct {
	Stages map[string]string `toml:"workstages"`
}

func WorkstagesPath() string {
	return filepath.Join(Dir(), "workstages.toml")
}

func LoadWorkstages() (*Workstages, error) {
	ws := &Workstages{Stages: make(map[string]string)}
	if _, err := toml.DecodeFile(WorkstagesPath(), ws); err != nil {
		if os.IsNotExist(err) {
			return ws, nil
		}
		return nil, fmt.Errorf("read workstages: %w", err)
	}
	return ws, nil
}

func (ws *Workstages) Save() error {
	if err := os.MkdirAll(Dir(), 0700); err != nil {
		return err
	}
	f, err := os.Create(WorkstagesPath())
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(ws)
}

// Name returns the workstage name for a given ID, or "" if not found.
func (ws *Workstages) Name(id int) string {
	return ws.Stages[strconv.Itoa(id)]
}

// FuzzyMatch finds the best matching workstage by name query.
// Returns id, matched name, ok.
func (ws *Workstages) FuzzyMatch(query string) (int, string, bool) {
	q := strings.ToLower(query)
	match := func(check func(name string) bool) (int, string, bool) {
		for idStr, name := range ws.Stages {
			if check(strings.ToLower(name)) {
				id, _ := strconv.Atoi(idStr)
				return id, name, true
			}
		}
		return 0, "", false
	}
	if id, name, ok := match(func(n string) bool { return n == q }); ok {
		return id, name, ok
	}
	if id, name, ok := match(func(n string) bool { return strings.HasPrefix(n, q) }); ok {
		return id, name, ok
	}
	return match(func(n string) bool { return strings.Contains(n, q) })
}

// UniqueNames returns deduplicated workstage names for completion/display.
func (ws *Workstages) UniqueNames() []string {
	seen := map[string]bool{}
	var names []string
	for _, name := range ws.Stages {
		if !seen[name] {
			seen[name] = true
			names = append(names, name)
		}
	}
	return names
}
