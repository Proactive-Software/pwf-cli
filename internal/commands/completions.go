package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

var installCompletionsCmd = &cobra.Command{
	Use:   "install-completions",
	Short: "Install shell completions for pwf",
	Long: `Install shell completions for your current shell.

Supports zsh, bash, and fish. Detects your shell automatically,
or pass a shell name explicitly.`,
	Example: `  pwf install-completions
  pwf install-completions zsh
  pwf install-completions bash`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		shell := detectShell()
		if len(args) == 1 {
			shell = args[0]
		}

		switch shell {
		case "zsh":
			return installZsh()
		case "bash":
			return installBash()
		case "fish":
			return installFish()
		default:
			return fmt.Errorf("unsupported shell %q — run: pwf completion %s --help", shell, shell)
		}
	},
}

func init() {
	Root.AddCommand(installCompletionsCmd)
}

func detectShell() string {
	shell := os.Getenv("SHELL")
	return filepath.Base(shell)
}

func installZsh() error {
	dir, err := zshCompletionsDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	dest := filepath.Join(dir, "_pwf")
	f, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("create %s: %w", dest, err)
	}
	defer func() { _ = f.Close() }()
	if err := Root.GenZshCompletion(f); err != nil {
		return err
	}
	fmt.Printf("Installed zsh completions: %s\n", dest)
	fmt.Println("Restart your shell or run: exec zsh")
	return nil
}

func installBash() error {
	var dir string
	if runtime.GOOS == "darwin" {
		// Homebrew bash completions
		prefix, err := homebrewPrefix()
		if err == nil {
			dir = filepath.Join(prefix, "etc", "bash_completion.d")
		}
	}
	if dir == "" {
		dir = "/etc/bash_completion.d"
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		// Fall back to ~/.local/share/bash-completion/completions
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".local", "share", "bash-completion", "completions")
		if err2 := os.MkdirAll(dir, 0755); err2 != nil {
			return fmt.Errorf("could not create completions dir: %w", err2)
		}
	}
	dest := filepath.Join(dir, "pwf")
	f, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("create %s: %w", dest, err)
	}
	defer func() { _ = f.Close() }()
	if err := Root.GenBashCompletion(f); err != nil {
		return err
	}
	fmt.Printf("Installed bash completions: %s\n", dest)
	fmt.Println("Restart your shell or run: source ~/.bashrc")
	return nil
}

func installFish() error {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".config", "fish", "completions")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	dest := filepath.Join(dir, "pwf.fish")
	f, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("create %s: %w", dest, err)
	}
	defer func() { _ = f.Close() }()
	if err := Root.GenFishCompletion(f, true); err != nil {
		return err
	}
	fmt.Printf("Installed fish completions: %s\n", dest)
	return nil
}

func zshCompletionsDir() (string, error) {
	// Prefer Homebrew site-functions (works on both Intel and Apple Silicon)
	prefix, err := homebrewPrefix()
	if err == nil {
		dir := filepath.Join(prefix, "share", "zsh", "site-functions")
		if _, err := os.Stat(dir); err == nil {
			return dir, nil
		}
	}
	// Fall back to ~/.zsh/completions (user must add to $FPATH)
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".zsh", "completions")
	fmt.Printf("Note: add this to your .zshrc if not already present:\n  fpath=(%s $fpath)\n\n", dir)
	return dir, nil
}

func homebrewPrefix() (string, error) {
	out, err := exec.Command("brew", "--prefix").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
