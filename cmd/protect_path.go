package cmd

import (
	"fmt"

	"github.com/gedaliah/oops/internal/config"
	"github.com/gedaliah/oops/internal/style"
	"github.com/spf13/cobra"
)

var (
	protectPathAlwaysConfirm bool
	protectPathRetention     int
	protectPathRemove        bool
	protectPathList          bool
)

var protectPathCmd = &cobra.Command{
	Use:   "protect-path [path]",
	Short: "Manage high-safety path rules",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runProtectPath,
}

func init() {
	protectPathCmd.Flags().BoolVar(&protectPathAlwaysConfirm, "always-confirm", false, "Always require confirmation for this path")
	protectPathCmd.Flags().IntVar(&protectPathRetention, "retention-hours", 0, "Keep matching backups for at least N hours")
	protectPathCmd.Flags().BoolVar(&protectPathRemove, "remove", false, "Remove a protected path rule")
	protectPathCmd.Flags().BoolVar(&protectPathList, "list", false, "List protected path rules")
	rootCmd.AddCommand(protectPathCmd)
}

func runProtectPath(cmd *cobra.Command, args []string) error {
	if protectPathList || len(args) == 0 && !protectPathRemove {
		return listProtectedPaths()
	}
	if len(args) != 1 {
		return fmt.Errorf("path is required")
	}
	if protectPathRemove {
		removed, err := config.RemoveProtectedPath(args[0])
		if err != nil {
			return err
		}
		if !removed {
			fmt.Println(style.Warning("No matching protected path rule"))
			return nil
		}
		fmt.Println(style.Success("Removed protected path: " + args[0]))
		return nil
	}

	rule, err := config.AddProtectedPath(args[0], protectPathAlwaysConfirm, protectPathRetention)
	if err != nil {
		return err
	}
	fmt.Println(style.Success("Protected path: " + style.ShortenPath(rule.Path)))
	if rule.AlwaysConfirm {
		fmt.Println(style.Dim.Render("  always confirm: yes"))
	}
	if rule.RetentionHours > 0 {
		fmt.Printf("%s %d hours\n", style.Dim.Render("  retention:"), rule.RetentionHours)
	}
	return nil
}

func listProtectedPaths() error {
	cfg := config.Load()
	if len(cfg.ProtectedPaths) == 0 {
		fmt.Println(style.Dim.Render("No protected paths configured."))
		return nil
	}
	for _, rule := range cfg.ProtectedPaths {
		fmt.Println(style.Cyan.Render(style.ShortenPath(rule.Path)))
		if rule.AlwaysConfirm {
			fmt.Println(style.Dim.Render("  always confirm: yes"))
		}
		if rule.RetentionHours > 0 {
			fmt.Printf("%s %d hours\n", style.Dim.Render("  retention:"), rule.RetentionHours)
		}
	}
	return nil
}
