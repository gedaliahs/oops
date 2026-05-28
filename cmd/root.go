package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"github.com/charmbracelet/lipgloss"
	"github.com/gedaliah/oops/internal/cleanup"
	"github.com/gedaliah/oops/internal/journal"
	"github.com/gedaliah/oops/internal/style"
	"github.com/gedaliah/oops/internal/trash"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	helpRed  = lipgloss.NewStyle().Foreground(lipgloss.Color("#e05252"))
	helpDim  = lipgloss.NewStyle().Foreground(lipgloss.Color("#6b7280"))
	helpBold = lipgloss.NewStyle().Bold(true)
	helpCmd  = lipgloss.NewStyle().Foreground(lipgloss.Color("#e05252")).Bold(true)
	helpDesc = lipgloss.NewStyle().Foreground(lipgloss.Color("#9ca3af"))
)

var Version = "0.5.3"

var versionFlag bool
var upgradeFlag bool
var restoreOverwrite bool
var restoreBackupCurrent bool
var restoreToDir string
var restoreDryRun bool
var restorePlan bool

var rootCmd = &cobra.Command{
	Use:          "oops [N]",
	Short:        "Terminal undo — restore your last destructive command",
	SilenceUsage: true,
	Args:         cobra.MaximumNArgs(1),
	RunE:         runUndo,
}

var undoCmd = &cobra.Command{
	Use:     "undo [N]",
	Aliases: []string{"restore"},
	Short:   "Undo a protected destructive command",
	Args:    cobra.MaximumNArgs(1),
	RunE:    runUndo,
}

func init() {
	rootCmd.SetHelpFunc(customHelp)
	rootCmd.Flags().BoolVarP(&versionFlag, "version", "v", false, "Print version")
	rootCmd.Flags().BoolVar(&upgradeFlag, "upgrade", false, "Upgrade oops to the latest version")
	addRestoreFlags(rootCmd.Flags())
	addRestoreFlags(undoCmd.Flags())
	rootCmd.AddCommand(undoCmd)
}

func customHelp(cmd *cobra.Command, args []string) {
	fmt.Println()
	fmt.Println("  " + helpRed.Render("oops") + helpDim.Render(" v"+Version+" — undo for your terminal"))
	fmt.Println()
	fmt.Println("  " + helpBold.Render("Usage"))
	fmt.Println("    oops" + helpDim.Render("           undo the last destructive action"))
	fmt.Println("    oops " + helpDim.Render("<N>") + helpDim.Render("        undo the Nth most recent action"))
	fmt.Println()
	fmt.Println("  " + helpBold.Render("Commands"))
	printCmd("oops log", "show undo history")
	printCmd("oops undo", "undo the last destructive action")
	printCmd("oops restore", "alias for oops undo")
	printCmd("oops status", "show health and backup state")
	printCmd("oops show", "preview an undo")
	printCmd("oops diff", "compare backup with current files")
	printCmd("oops keep", "keep a backup from cleanup")
	printCmd("oops size", "show backup disk usage")
	printCmd("oops clean", "remove old backups")
	printCmd("oops cleanup-service", "run cleanup in the background")
	printCmd("oops config", "view or change settings")
	printCmd("oops protect-path", "manage high-safety paths")
	printCmd("oops doctor", "check installation health")
	printCmd("oops init <shell>", "print shell hook (zsh, bash, fish)")
	printCmd("oops agent-mode", "toggle AI agent protection")
	printCmd("oops tutorial", "interactive walkthrough")
	printCmd("oops uninstall", "remove oops from your system")
	fmt.Println()
	fmt.Println("  " + helpBold.Render("Flags"))
	printCmd("--version, -v", "print version")
	printCmd("--upgrade", "upgrade to the latest version")
	printCmd("--overwrite", "overwrite existing restore targets")
	printCmd("--backup-current", "move existing targets aside")
	printCmd("--to <dir>", "restore into another directory")
	printCmd("--dry-run", "show restore actions without changing files")
	printCmd("--plan", "show restore plan with conflicts and backups")
	fmt.Println()
	fmt.Println("  " + helpBold.Render("Examples"))
	fmt.Println("    " + helpDim.Render("$") + " rm important-file.txt")
	fmt.Println("    " + helpDim.Render("$") + " " + helpRed.Render("oops"))
	fmt.Println("    " + style.Green.Render("✓") + " restored important-file.txt")
	fmt.Println()
	fmt.Println("    " + helpDim.Render("$") + " oops log" + helpDim.Render("          # see what you can undo"))
	fmt.Println("    " + helpDim.Render("$") + " oops 2" + helpDim.Render("            # undo second-to-last"))
	fmt.Println("    " + helpDim.Render("$") + " oops clean --all" + helpDim.Render("  # clear all backups"))
	fmt.Println()
	fmt.Println()
	fmt.Println("  " + helpDim.Render("https://oops-cli.com  ·  https://github.com/gedaliahs/oops"))
	fmt.Println()
}

func printCmd(name, desc string) {
	padding := 20 - len(name)
	if padding < 2 {
		padding = 2
	}
	spaces := ""
	for i := 0; i < padding; i++ {
		spaces += " "
	}
	fmt.Println("    " + helpCmd.Render(name) + spaces + helpDesc.Render(desc))
}

func addRestoreFlags(flags *pflag.FlagSet) {
	flags.BoolVar(&restoreOverwrite, "overwrite", false, "Overwrite existing restore targets")
	flags.BoolVar(&restoreBackupCurrent, "backup-current", false, "Move existing restore targets aside before restoring")
	flags.StringVar(&restoreToDir, "to", "", "Restore into a directory instead of original paths")
	flags.BoolVar(&restoreDryRun, "dry-run", false, "Show restore actions without changing files")
	flags.BoolVar(&restorePlan, "plan", false, "Show detailed restore plan without changing files")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runUndo(cmd *cobra.Command, args []string) error {
	if versionFlag {
		fmt.Println(helpRed.Render("oops") + " " + helpDim.Render("v"+Version))
		return nil
	}

	if upgradeFlag {
		return runUpgrade()
	}

	cleanup.RunIfNeeded()

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
		fmt.Println(style.Warning("Nothing to undo"))
		return nil
	}

	if n > len(entries) {
		return fmt.Errorf("only %d undoable actions in history", len(entries))
	}

	entry := entries[n-1]

	if entry.GitAction != "" {
		if restoreDryRun || restorePlan {
			printGitRestorePlan(entry)
			return nil
		}
		return undoGit(entry)
	}

	if entry.TrashDir == "" {
		return fmt.Errorf("no backup found for this action")
	}

	if restoreOverwrite && restoreBackupCurrent {
		return fmt.Errorf("use only one of --overwrite or --backup-current")
	}

	opts := trash.RestoreOptions{
		Overwrite:     restoreOverwrite,
		BackupCurrent: restoreBackupCurrent,
		ToDir:         restoreToDir,
	}

	plan, err := trash.PlanRestore(entry.TrashDir, opts)
	if err != nil {
		return fmt.Errorf("planning restore: %w", err)
	}

	if restoreDryRun || restorePlan {
		printRestorePlan(entry, plan)
		return nil
	}

	restored, err := trash.ExecuteRestorePlan(plan)
	if err != nil {
		return fmt.Errorf("restoring files: %w", err)
	}

	if restoreToDir == "" {
		if err := journal.MarkUndone(entry.ID); err != nil {
			fmt.Fprintln(os.Stderr, style.Warning("Could not mark entry as undone: "+err.Error()))
		}
	}

	fmt.Println(style.Success(fmt.Sprintf("Undid: %s", style.ShortenPath(entry.Desc))))
	for _, f := range restored {
		fmt.Println(style.Restored(f.Path))
		if f.BackupCurrent != "" {
			fmt.Println(style.Dim.Render("  saved current as ") + style.Cyan.Render(style.ShortenPath(f.BackupCurrent)))
		}
	}
	if restoreToDir != "" {
		fmt.Println(style.Dim.Render("  original undo entry was left available"))
	}

	return nil
}

func printRestorePlan(entry journal.Entry, plan trash.RestorePlan) {
	title := "Restore dry run"
	if restorePlan {
		title = "Restore plan"
	}
	conflicts := make(map[string]string, len(plan.Conflicts))
	for _, conflict := range plan.Conflicts {
		conflicts[conflict.Path] = conflict.Reason
	}

	fmt.Println(style.Bold.Render(title))
	fmt.Println(style.Dim.Render("  Command: ") + style.ShortenPath(entry.Command))
	fmt.Println(style.Dim.Render("  Action: ") + style.ShortenPath(entry.Desc))
	if plan.Options.ToDir != "" {
		fmt.Println(style.Dim.Render("  Destination: ") + style.ShortenPath(plan.Options.ToDir))
	} else {
		fmt.Println(style.Dim.Render("  Destination: ") + "original paths")
	}
	fmt.Println(style.Dim.Render("  Undo entry: ") + "left available")
	if len(plan.Files) == 0 {
		fmt.Println(style.Dim.Render("  Files: none"))
		return
	}

	fmt.Println(style.Dim.Render("  Files:"))
	for _, item := range plan.Files {
		action := plannedRestoreAction(item, conflicts[item.Target])
		fmt.Println("    " + style.Cyan.Render(style.ShortenPath(item.Target)) + style.Dim.Render("  "+action))
		if restorePlan {
			fmt.Println(style.Dim.Render("      from: ") + style.ShortenPath(item.Backup))
			if item.BackupCurrent != "" {
				fmt.Println(style.Dim.Render("      save current as: ") + style.ShortenPath(item.BackupCurrent))
			}
		}
	}

	if len(plan.Conflicts) > 0 {
		fmt.Println(style.Warning("Conflicts:"))
		for _, conflict := range plan.Conflicts {
			fmt.Println("    " + style.ShortenPath(conflict.Path) + style.Dim.Render(" - "+conflict.Reason))
		}
		fmt.Println(style.Dim.Render("  Add --overwrite, --backup-current, or --to <dir>."))
	}
	fmt.Println(style.Dim.Render("  No files changed."))
}

func plannedRestoreAction(item trash.PlannedRestore, conflict string) string {
	switch {
	case conflict != "":
		return "conflict: " + conflict
	case item.WillBackupCurrent:
		return "backup current, then restore"
	case item.WillOverwrite:
		return "overwrite"
	case item.TargetExists:
		return "target exists"
	default:
		return "create"
	}
}

func printGitRestorePlan(entry journal.Entry) {
	title := "Restore dry run"
	if restorePlan {
		title = "Restore plan"
	}
	fmt.Println(style.Bold.Render(title))
	fmt.Println(style.Dim.Render("  Command: ") + style.ShortenPath(entry.Command))
	fmt.Println(style.Dim.Render("  Action: ") + style.ShortenPath(entry.Desc))
	switch entry.GitAction {
	case "stash":
		stashRef := entry.GitStash
		if stashRef == "" {
			stashRef = "stash@{0}"
		}
		fmt.Println(style.Dim.Render("  Git action: ") + "git stash apply " + stashRef)
	case "log-branch":
		fmt.Println(style.Dim.Render("  Git action: ") + "git branch " + entry.GitRef + " " + entry.GitSHA)
	default:
		fmt.Println(style.Dim.Render("  Git action: ") + entry.GitAction)
	}
	fmt.Println(style.Dim.Render("  Undo entry: ") + "left available")
	fmt.Println(style.Dim.Render("  No files changed."))
}

func runUpgrade() error {
	upgradeCmd := exec.Command("bash", "-c", "curl -fsSL oops-cli.com/install.sh | bash")
	upgradeCmd.Stdout = os.Stdout
	upgradeCmd.Stderr = os.Stderr
	upgradeCmd.Stdin = os.Stdin
	return upgradeCmd.Run()
}

func undoGit(entry journal.Entry) error {
	switch entry.GitAction {
	case "stash":
		stashRef := entry.GitStash
		if stashRef == "" {
			stashRef = "stash@{0}"
		}
		out, err := exec.Command("git", "stash", "apply", stashRef).CombinedOutput()
		if err != nil {
			return fmt.Errorf("git stash apply failed: %s", string(out))
		}
		if err := journal.MarkUndone(entry.ID); err != nil {
			fmt.Fprintln(os.Stderr, style.Warning("Could not mark entry as undone: "+err.Error()))
		}
		fmt.Println(style.Success(fmt.Sprintf("Undid: %s", style.ShortenPath(entry.Desc))))
		fmt.Println(style.Green.Render("  Applied stash: ") + stashRef)

	case "log-branch":
		if entry.GitSHA == "" {
			return fmt.Errorf("no SHA recorded for deleted branch")
		}
		branchName := entry.GitRef
		out, err := exec.Command("git", "branch", branchName, entry.GitSHA).CombinedOutput()
		if err != nil {
			return fmt.Errorf("git branch restore failed: %s", string(out))
		}
		if err := journal.MarkUndone(entry.ID); err != nil {
			fmt.Fprintln(os.Stderr, style.Warning("Could not mark entry as undone: "+err.Error()))
		}
		fmt.Println(style.Success(fmt.Sprintf("Undid: %s", style.ShortenPath(entry.Desc))))
		fmt.Println(style.Green.Render("  Restored branch: ") + branchName + " at " + entry.GitSHA[:8])

	default:
		return fmt.Errorf("unknown git action: %s", entry.GitAction)
	}

	return nil
}
