package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gedaliah/oops/internal/config"
	"github.com/gedaliah/oops/internal/style"
	"github.com/spf13/cobra"
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove oops from your system",
	RunE:  runUninstall,
}

func init() {
	rootCmd.AddCommand(uninstallCmd)
}

func runUninstall(cmd *cobra.Command, args []string) error {
	fmt.Println(style.Bold.Render("oops uninstaller"))
	fmt.Println()

	// 1. Remove shell hook from rc file
	home, _ := os.UserHomeDir()
	shell := filepath.Base(os.Getenv("SHELL"))

	rcFiles := map[string]string{
		"zsh":  filepath.Join(home, ".zshrc"),
		"bash": filepath.Join(home, ".bashrc"),
		"fish": filepath.Join(home, ".config", "fish", "config.fish"),
	}

	rcFile := rcFiles[shell]
	if rcFile == "" {
		for _, f := range rcFiles {
			removeHookFromFile(f)
		}
	} else {
		removeHookFromFile(rcFile)
	}

	// 2. Remove ~/.oops directory
	oopsDir := config.OopsDir()
	if _, err := os.Stat(oopsDir); err == nil {
		os.RemoveAll(oopsDir)
		fmt.Println(style.Success("Removed " + oopsDir))
	}

	// 3. Tell user how to remove the binary
	bin, err := os.Executable()
	if err == nil {
		fmt.Println()
		fmt.Println("  To finish, remove the binary:")
		fmt.Println()
		fmt.Println("    " + style.Bold.Render("sudo rm "+bin))
		fmt.Println()
	}

	fmt.Println(style.Success("Done. Open a new terminal tab to finish."))
	return nil
}

func removeHookFromFile(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	lines := strings.Split(string(data), "\n")
	var filtered []string
	found := false
	for _, line := range lines {
		if strings.Contains(line, "oops init") {
			found = true
			continue
		}
		filtered = append(filtered, line)
	}

	if !found {
		return
	}

	for len(filtered) > 0 && filtered[len(filtered)-1] == "" {
		filtered = filtered[:len(filtered)-1]
	}
	filtered = append(filtered, "")

	os.WriteFile(path, []byte(strings.Join(filtered, "\n")), 0o644)
	fmt.Println(style.Success("Removed hook from " + path))
}
