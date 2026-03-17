package cmd

import (
	"fmt"
	"os"

	"github.com/gedaliah/oops/internal/config"
	"github.com/gedaliah/oops/internal/shell"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:       "init <shell>",
	Short:     "Print the shell hook for your shell (zsh, bash, fish)",
	Long:      "Print the shell hook to stdout. Add `eval \"$(oops init zsh)\"` to your .zshrc.",
	Args:      cobra.ExactArgs(1),
	ValidArgs: []string{"zsh", "bash", "fish"},
	RunE:      runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	if err := config.EnsureDir(); err != nil {
		return fmt.Errorf("creating oops directory: %w", err)
	}

	bin, err := os.Executable()
	if err != nil {
		bin = "oops"
	}

	switch args[0] {
	case "zsh":
		fmt.Print(shell.ZshHook(bin))
	case "bash":
		fmt.Print(shell.BashHook(bin))
	case "fish":
		fmt.Print(shell.FishHook(bin))
	default:
		return fmt.Errorf("unsupported shell: %s (use zsh, bash, or fish)", args[0])
	}

	return nil
}
