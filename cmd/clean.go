package cmd

import (
	"fmt"
	"time"

	"github.com/gedaliah/oops/internal/cleanup"
	"github.com/gedaliah/oops/internal/config"
	"github.com/gedaliah/oops/internal/style"
	"github.com/spf13/cobra"
)

var (
	cleanAll  bool
	cleanDays int
)

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Remove old backups to free disk space",
	RunE:  runClean,
}

func init() {
	cleanCmd.Flags().BoolVar(&cleanAll, "all", false, "Remove all backups")
	cleanCmd.Flags().IntVar(&cleanDays, "older-than", 0, "Remove entries older than N days")
	rootCmd.AddCommand(cleanCmd)
}

func runClean(cmd *cobra.Command, args []string) error {
	if cleanAll {
		if err := cleanup.Purge(); err != nil {
			return err
		}
		fmt.Println(style.Success("All backups removed"))
		return nil
	}

	days := cleanDays
	if days == 0 {
		cfg := config.Load()
		days = cfg.RetentionDays
	}

	cutoff := time.Now().Add(-time.Duration(days) * 24 * time.Hour)
	removed, freed, err := cleanup.PurgeBefore(cutoff)
	if err != nil {
		return err
	}

	if removed == 0 {
		fmt.Println(style.Dim.Render("Nothing to clean up"))
	} else {
		fmt.Println(style.Success(fmt.Sprintf("Removed %d entries, freed %s", removed, style.FormatSize(freed))))
	}

	return nil
}
