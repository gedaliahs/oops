package cmd

import (
	"fmt"

	"github.com/gedaliah/oops/internal/style"
	"github.com/gedaliah/oops/internal/trash"
	"github.com/spf13/cobra"
)

var sizeCmd = &cobra.Command{
	Use:   "size",
	Short: "Show total disk usage of oops backups",
	RunE:  runSize,
}

func init() {
	rootCmd.AddCommand(sizeCmd)
}

func runSize(cmd *cobra.Command, args []string) error {
	total := trash.TotalSize()
	dirs, _ := trash.ListTrashDirs()

	fmt.Printf("%s %s in %d backups\n",
		style.Bold.Render("Trash size:"),
		style.Cyan.Render(style.FormatSize(total)),
		len(dirs),
	)

	return nil
}
