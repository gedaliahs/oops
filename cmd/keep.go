package cmd

import (
	"fmt"
	"strconv"

	"github.com/gedaliah/oops/internal/journal"
	"github.com/gedaliah/oops/internal/style"
	"github.com/spf13/cobra"
)

var keepCmd = &cobra.Command{
	Use:     "keep [N]",
	Aliases: []string{"pin"},
	Short:   "Keep a backup from automatic cleanup",
	Args:    cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return setPinned(args, true)
	},
}

var unkeepCmd = &cobra.Command{
	Use:     "unkeep [N]",
	Aliases: []string{"unpin"},
	Short:   "Allow a kept backup to be cleaned up",
	Args:    cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return setPinned(args, false)
	},
}

func init() {
	rootCmd.AddCommand(keepCmd)
	rootCmd.AddCommand(unkeepCmd)
}

func setPinned(args []string, pinned bool) error {
	n := 1
	if len(args) == 1 {
		var err error
		n, err = strconv.Atoi(args[0])
		if err != nil || n < 1 {
			return fmt.Errorf("invalid argument: %s (expected a positive number)", args[0])
		}
	}

	entries, err := journal.Last(n)
	if err != nil {
		return fmt.Errorf("reading journal: %w", err)
	}
	if len(entries) == 0 {
		fmt.Println(style.Warning("Nothing to keep"))
		return nil
	}
	if n > len(entries) {
		return fmt.Errorf("only %d undoable actions in history", len(entries))
	}

	entry := entries[n-1]
	if err := journal.MarkPinned(entry.ID, pinned); err != nil {
		return err
	}

	action := "Kept"
	if !pinned {
		action = "Unkept"
	}
	fmt.Println(style.Success(fmt.Sprintf("%s: %s", action, style.ShortenPath(entry.Desc))))
	return nil
}
