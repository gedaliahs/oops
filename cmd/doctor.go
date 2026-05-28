package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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
	fmt.Println(style.Success(fmt.Sprintf("Config: retention=%dh, max_trash=%s", cfg.RetentionHours, style.FormatSize(cfg.MaxTrashBytes))))

	if latest, err := fetchLatestVersion(); err == nil && latest != "" {
		if latest == Version {
			fmt.Println(style.Success("Version: v" + Version + " (latest)"))
		} else {
			fmt.Println(style.Warning("Version: v" + Version + " installed, v" + latest + " available"))
		}
	} else {
		fmt.Println(style.Dim.Render("  " + style.SymBackup + " Latest version check skipped"))
	}

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
	if os.Getenv("OOPS_HOOK") == "1" {
		fmt.Println(style.Success("Shell hook loaded in current shell"))
	} else {
		fmt.Println(style.Warning("Shell hook is not loaded in this process (open a new terminal tab after install)"))
	}

	shellName := filepath.Base(shellPath)
	home, _ := os.UserHomeDir()
	hookFound := false

	rcFiles := map[string][]string{
		"zsh":  {filepath.Join(home, ".zshenv"), filepath.Join(home, ".zshrc")},
		"bash": {filepath.Join(home, ".bashrc"), filepath.Join(home, ".bash_profile")},
		"fish": {filepath.Join(home, ".config", "fish", "config.fish")},
	}

	if candidates, exists := rcFiles[shellName]; exists {
		for _, rcFile := range candidates {
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
		for _, candidates := range rcFiles {
			for _, rcFile := range candidates {
				if data, err := os.ReadFile(rcFile); err == nil {
					if strings.Contains(string(data), "oops init") {
						hookFound = true
						fmt.Println(style.Success("Shell hook in " + rcFile))
						break
					}
				}
			}
			if hookFound {
				break
			}
		}
	}

	if err := runDoctorSelfTest(); err != nil {
		fmt.Println(style.Error("Self-test failed: " + err.Error()))
		ok = false
	} else {
		fmt.Println(style.Success("Self-test restored a temporary file"))
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

func fetchLatestVersion() (string, error) {
	var out []byte
	var err error
	if curl, lookErr := exec.LookPath("curl"); lookErr == nil {
		out, err = exec.Command(curl, "-fsSL", "--max-time", "3", "https://oops-cli.com/install.sh").Output()
	} else if wget, lookErr := exec.LookPath("wget"); lookErr == nil {
		out, err = exec.Command(wget, "-qO-", "--timeout=3", "https://oops-cli.com/install.sh").Output()
	} else {
		return "", fmt.Errorf("curl or wget not found")
	}
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "VERSION=") {
			return strings.Trim(strings.TrimPrefix(line, "VERSION="), `"`), nil
		}
	}
	return "", fmt.Errorf("VERSION not found")
}

func runDoctorSelfTest() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	tmp, err := os.MkdirTemp("", "oops-doctor-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	testHome := filepath.Join(tmp, "home")
	testWork := filepath.Join(tmp, "work")
	if err := os.MkdirAll(testHome, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(testWork, 0o755); err != nil {
		return err
	}
	testFile := filepath.Join(testWork, "victim.txt")
	if err := os.WriteFile(testFile, []byte("doctor self-test\n"), 0o644); err != nil {
		return err
	}

	protect := exec.Command(exe, "protect", "--", "rm "+testFile)
	protect.Env = append(os.Environ(), "HOME="+testHome)
	protect.Dir = testWork
	if out, err := protect.CombinedOutput(); err != nil {
		return fmt.Errorf("protect failed on %s/%s: %s", runtime.GOOS, runtime.GOARCH, strings.TrimSpace(string(out)))
	}
	if err := os.Remove(testFile); err != nil {
		return err
	}

	restore := exec.Command(exe)
	restore.Env = append(os.Environ(), "HOME="+testHome)
	restore.Dir = testWork
	if out, err := restore.CombinedOutput(); err != nil {
		return fmt.Errorf("restore failed: %s", strings.TrimSpace(string(out)))
	}
	data, err := os.ReadFile(testFile)
	if err != nil {
		return err
	}
	if string(data) != "doctor self-test\n" {
		return fmt.Errorf("restored content mismatch")
	}
	return nil
}
