package cmd

import (
	"fmt"

	"github.com/gedaliah/oops/internal/config"
	"github.com/gedaliah/oops/internal/style"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config [key] [value]",
	Short: "Get or set configuration values",
	Long: `Get or set oops configuration.

Available keys:
  retention_days   Days to keep backups (default: 7)
  max_trash_bytes  Maximum trash size in bytes (default: 5368709120)
  risk_warning     Show warnings for high-risk commands (default: true)

Examples:
  oops config                        # show all settings
  oops config retention_days         # get a value
  oops config retention_days 14      # set a value`,
	Args: cobra.MaximumNArgs(2),
	RunE: runConfig,
}

func init() {
	rootCmd.AddCommand(configCmd)
}

func runConfig(cmd *cobra.Command, args []string) error {
	switch len(args) {
	case 0:
		// Show all config
		cfg := config.Load()
		fmt.Printf("retention_days   = %d\n", cfg.RetentionDays)
		fmt.Printf("max_trash_bytes  = %d (%s)\n", cfg.MaxTrashBytes, style.FormatSize(cfg.MaxTrashBytes))
		fmt.Printf("risk_warning     = %v\n", cfg.RiskWarning)
		fmt.Printf("confirm_mode     = %s\n", cfg.ConfirmMode)
		return nil

	case 1:
		// Get value
		v := config.Get(args[0])
		if v == "" {
			return fmt.Errorf("unknown key: %s", args[0])
		}
		fmt.Println(v)
		return nil

	case 2:
		// Set value
		if err := config.Set(args[0], args[1]); err != nil {
			return err
		}
		fmt.Println(style.Success(args[0] + " = " + args[1]))
		return nil
	}

	return nil
}
