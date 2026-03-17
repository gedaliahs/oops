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

var agentModeCmd = &cobra.Command{
	Use:   "agent-mode",
	Short: "Toggle AI agent protection (catches commands from Claude Code, Cursor, etc.)",
	RunE:  runAgentMode,
}

// Keep hidden aliases for internal use
var wrapCmd = &cobra.Command{
	Use:    "wrap",
	Hidden: true,
	RunE:   func(cmd *cobra.Command, args []string) error { return enableAgentMode() },
}

var unwrapCmd = &cobra.Command{
	Use:    "unwrap",
	Hidden: true,
	RunE:   func(cmd *cobra.Command, args []string) error { return disableAgentMode() },
}

var agentOnCmd = &cobra.Command{
	Use:   "on",
	Short: "Enable agent protection",
	RunE:  func(cmd *cobra.Command, args []string) error { return enableAgentMode() },
}

var agentOffCmd = &cobra.Command{
	Use:   "off",
	Short: "Disable agent protection",
	RunE:  func(cmd *cobra.Command, args []string) error { return disableAgentMode() },
}

func init() {
	agentModeCmd.AddCommand(agentOnCmd)
	agentModeCmd.AddCommand(agentOffCmd)
	rootCmd.AddCommand(agentModeCmd)
	rootCmd.AddCommand(wrapCmd)
	rootCmd.AddCommand(unwrapCmd)
}

var wrappedCommands = []string{
	"rm", "mv", "sed", "chmod", "chown", "truncate", "git",
}

func wrapperDir() string {
	return filepath.Join(config.OopsDir(), "bin")
}

func isAgentModeOn() bool {
	dir := wrapperDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	return len(entries) > 0
}

func runAgentMode(cmd *cobra.Command, args []string) error {
	if isAgentModeOn() {
		fmt.Println(style.Success("Agent mode is " + style.Bold.Render("on")))
		fmt.Println()
		fmt.Println(style.Dim.Render("  Commands from AI agents, scripts, and builds are"))
		fmt.Println(style.Dim.Render("  intercepted via PATH wrappers in ~/.oops/bin"))
		fmt.Println()
		fmt.Printf("  Turn off? Run %s\n", style.Red.Render("oops agent-mode off"))
		fmt.Println()
	} else {
		fmt.Println(style.Dim.Render("  Agent mode is " + style.Bold.Render("off")))
		fmt.Println()
		fmt.Println(style.Dim.Render("  Only your interactive terminal is protected."))
		fmt.Println(style.Dim.Render("  AI agents (Claude Code, Cursor, Aider) and scripts"))
		fmt.Println(style.Dim.Render("  can delete files without oops catching them."))
		fmt.Println()
		fmt.Printf("  Turn on? Run %s\n", style.Red.Render("oops agent-mode on"))
		fmt.Println()
	}
	return nil
}

func enableAgentMode() error {
	dir := wrapperDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	oopsBin, err := exec.LookPath("oops")
	if err != nil {
		oopsBin, _ = os.Executable()
	}

	for _, name := range wrappedCommands {
		realBin, err := findRealBin(name, dir)
		if err != nil {
			continue
		}

		wrapper := fmt.Sprintf("#!/bin/sh\n%s protect -- \"%s $*\" 2>/dev/null\n%s \"$@\"\n", oopsBin, name, realBin)

		if err := os.WriteFile(filepath.Join(dir, name), []byte(wrapper), 0o755); err != nil {
			return fmt.Errorf("writing wrapper for %s: %w", name, err)
		}
		fmt.Println(style.Success(name + " → " + realBin))
	}

	// Add to PATH in shell config
	home, _ := os.UserHomeDir()
	pathLine := fmt.Sprintf("export PATH=\"%s:$PATH\"", dir)

	rcFiles := []string{
		filepath.Join(home, ".zshenv"),
		filepath.Join(home, ".zshrc"),
		filepath.Join(home, ".bashrc"),
		filepath.Join(home, ".bash_profile"),
	}

	for _, rc := range rcFiles {
		if _, err := os.Stat(rc); err == nil {
			data, _ := os.ReadFile(rc)
			if !strings.Contains(string(data), ".oops/bin") {
				f, err := os.OpenFile(rc, os.O_APPEND|os.O_WRONLY, 0o644)
				if err == nil {
					f.WriteString("\n" + pathLine + "\n")
					f.Close()
					fmt.Println(style.Success("Added ~/.oops/bin to PATH in " + rc))
					break
				}
			}
		}
	}

	fmt.Println()
	fmt.Println(style.Success("Agent mode " + style.Bold.Render("on")))
	fmt.Println(style.Dim.Render("  Open a new terminal tab to activate."))
	fmt.Println()
	return nil
}

func disableAgentMode() error {
	dir := wrapperDir()

	// Remove wrappers
	entries, err := os.ReadDir(dir)
	if err == nil {
		for _, e := range entries {
			os.Remove(filepath.Join(dir, e.Name()))
		}
	}

	// Remove PATH line from rc files
	home, _ := os.UserHomeDir()
	rcFiles := []string{
		filepath.Join(home, ".zshenv"),
		filepath.Join(home, ".zshrc"),
		filepath.Join(home, ".bashrc"),
		filepath.Join(home, ".bash_profile"),
	}

	for _, rc := range rcFiles {
		data, err := os.ReadFile(rc)
		if err != nil {
			continue
		}
		if strings.Contains(string(data), ".oops/bin") {
			lines := strings.Split(string(data), "\n")
			var filtered []string
			for _, line := range lines {
				if !strings.Contains(line, ".oops/bin") {
					filtered = append(filtered, line)
				}
			}
			os.WriteFile(rc, []byte(strings.Join(filtered, "\n")), 0o644)
		}
	}

	fmt.Println(style.Success("Agent mode " + style.Bold.Render("off")))
	fmt.Println(style.Dim.Render("  Open a new terminal tab to deactivate."))
	fmt.Println()
	return nil
}

func findRealBin(name, skipDir string) (string, error) {
	pathDirs := strings.Split(os.Getenv("PATH"), ":")
	for _, dir := range pathDirs {
		if dir == skipDir {
			continue
		}
		candidate := filepath.Join(dir, name)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}
	}
	out, err := exec.Command("which", name).Output()
	if err != nil {
		return "", fmt.Errorf("%s not found", name)
	}
	return strings.TrimSpace(string(out)), nil
}
