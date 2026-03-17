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

var wrapCmd = &cobra.Command{
	Use:    "wrap",
	Short:  "Install PATH wrappers to catch all destructive commands (including from AI agents)",
	Hidden: true,
	RunE:   runWrap,
}

var unwrapCmd = &cobra.Command{
	Use:    "unwrap",
	Short:  "Remove PATH wrappers",
	Hidden: true,
	RunE:   runUnwrap,
}

func init() {
	rootCmd.AddCommand(wrapCmd)
	rootCmd.AddCommand(unwrapCmd)
}

var wrappedCommands = []struct {
	name    string
	realBin string
}{
	{"rm", ""},
	{"mv", ""},
	{"sed", ""},
	{"chmod", ""},
	{"chown", ""},
	{"truncate", ""},
	{"git", ""},
}

func wrapperDir() string {
	return filepath.Join(config.OopsDir(), "bin")
}

func runWrap(cmd *cobra.Command, args []string) error {
	dir := wrapperDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	oopsBin, _ := os.Executable()

	for i, c := range wrappedCommands {
		// Find the real binary
		realBin, err := findRealBin(c.name, dir)
		if err != nil {
			fmt.Println(style.Dim.Render("  skipping " + c.name + " (not found)"))
			continue
		}
		wrappedCommands[i].realBin = realBin

		wrapper := fmt.Sprintf(`#!/bin/sh
%s protect -- "%s $*" 2>/dev/null
%s "$@"
`, oopsBin, c.name, realBin)

		wrapperPath := filepath.Join(dir, c.name)
		if err := os.WriteFile(wrapperPath, []byte(wrapper), 0o755); err != nil {
			return fmt.Errorf("writing wrapper for %s: %w", c.name, err)
		}
		fmt.Println(style.Success(c.name + " → " + realBin))
	}

	// Check if wrapper dir is in PATH
	pathDirs := strings.Split(os.Getenv("PATH"), ":")
	inPath := false
	for _, d := range pathDirs {
		if d == dir {
			inPath = true
			break
		}
	}

	if !inPath {
		fmt.Println()
		fmt.Println(style.Success("Wrappers installed to " + dir))
		fmt.Println()
		fmt.Println(style.Dim.Render("  PATH will be updated on next shell restart."))
	} else {
		fmt.Println()
		fmt.Println(style.Success("Wrappers active"))
	}

	return nil
}

func runUnwrap(cmd *cobra.Command, args []string) error {
	dir := wrapperDir()
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		fmt.Println(style.Dim.Render("  No wrappers installed"))
		return nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, e := range entries {
		os.Remove(filepath.Join(dir, e.Name()))
		fmt.Println(style.Success("Removed " + e.Name() + " wrapper"))
	}

	return nil
}

// findRealBin finds the actual binary, skipping our wrapper dir
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
	// Fallback: use which
	out, err := exec.Command("which", name).Output()
	if err != nil {
		return "", fmt.Errorf("%s not found", name)
	}
	return strings.TrimSpace(string(out)), nil
}
