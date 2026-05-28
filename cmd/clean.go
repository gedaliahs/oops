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
	cleanAll   bool
	cleanHours int
	cleanDays  int
)

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Remove old backups to free disk space",
	RunE:  runClean,
}

func init() {
	cleanCmd.Flags().BoolVar(&cleanAll, "all", false, "Remove all backups")
	cleanCmd.Flags().IntVar(&cleanHours, "older-than-hours", 0, "Remove entries older than N hours")
	cleanCmd.Flags().IntVar(&cleanDays, "older-than", 0, "Remove entries older than N days")
	_ = cleanCmd.Flags().MarkDeprecated("older-than", "use --older-than-hours")
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

	if cleanHours < 0 {
		return fmt.Errorf("--older-than-hours must be positive")
	}
	if cleanDays < 0 {
		return fmt.Errorf("--older-than must be positive")
	}
	if cleanHours > 0 && cleanDays > 0 {
		return fmt.Errorf("use only one of --older-than-hours or --older-than")
	}

	retention := config.Load().RetentionDuration()
	if cleanHours > 0 {
		retention = time.Duration(cleanHours) * time.Hour
	} else if cleanDays > 0 {
		retention = time.Duration(cleanDays) * 24 * time.Hour
	}

	cutoff := time.Now().Add(-retention)
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
