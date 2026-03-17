package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gedaliah/oops/internal/config"
	"github.com/gedaliah/oops/internal/style"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check oops installation health",
	RunE:  runDoctor,
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

func runDoctor(cmd *cobra.Command, args []string) error {
	fmt.Println(style.Banner())
	fmt.Println()

	ok := true

	// Check oops directory
	if _, err := os.Stat(config.OopsDir()); err != nil {
		fmt.Println(style.Error("~/.oops directory missing"))
		ok = false
	} else {
		fmt.Println(style.Success("~/.oops directory exists"))
	}

	// Check trash directory
	if _, err := os.Stat(config.TrashDir()); err != nil {
		fmt.Println(style.Error("~/.oops/trash directory missing"))
		ok = false
	} else {
		fmt.Println(style.Success("~/.oops/trash directory exists"))
	}

	// Check config
	cfg := config.Load()
	fmt.Println(style.Success(fmt.Sprintf("Config: retention=%dd, max_trash=%s", cfg.RetentionDays, style.FormatSize(cfg.MaxTrashBytes))))

	// Check journal
	if _, err := os.Stat(config.JournalPath()); err != nil {
		fmt.Println(style.Dim.Render("  " + style.SymBackup + " No journal yet (normal for first run)"))
	} else {
		fmt.Println(style.Success("Journal file exists"))
	}

	// Check git availability
	if _, err := exec.LookPath("git"); err != nil {
		fmt.Println(style.Warning("git not found (git-related undo will not work)"))
	} else {
		fmt.Println(style.Success("git available"))
	}

	// Check shell and hook
	shellPath := os.Getenv("SHELL")
	if shellPath != "" {
		fmt.Println(style.Success("Shell: " + shellPath))
	}

	shellName := filepath.Base(shellPath)
	home, _ := os.UserHomeDir()
	hookFound := false

	rcFiles := map[string]string{
		"zsh":  filepath.Join(home, ".zshrc"),
		"bash": filepath.Join(home, ".bashrc"),
		"fish": filepath.Join(home, ".config", "fish", "config.fish"),
	}

	if rcFile, exists := rcFiles[shellName]; exists {
		if data, err := os.ReadFile(rcFile); err == nil {
			if strings.Contains(string(data), "oops init") {
				hookFound = true
				fmt.Println(style.Success("Shell hook in " + rcFile))
			}
		}
	}

	if !hookFound {
		// Check all rc files as fallback
		for _, rcFile := range rcFiles {
			if data, err := os.ReadFile(rcFile); err == nil {
				if strings.Contains(string(data), "oops init") {
					hookFound = true
					fmt.Println(style.Success("Shell hook in " + rcFile))
					break
				}
			}
		}
	}

	if !hookFound {
		fmt.Println(style.Error("Shell hook not found — run the installer or add it manually:"))
		fmt.Println(style.Dim.Render("  eval \"$(oops init " + shellName + ")\""))
		ok = false
	}

	fmt.Println()
	if ok {
		fmt.Println(style.Success("All checks passed"))
	} else {
		fmt.Println(style.Error("Some checks failed"))
	}

	return nil
}
