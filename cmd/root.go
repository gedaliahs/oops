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
)

var (
	helpRed  = lipgloss.NewStyle().Foreground(lipgloss.Color("#e05252"))
	helpDim  = lipgloss.NewStyle().Foreground(lipgloss.Color("#6b7280"))
	helpBold = lipgloss.NewStyle().Bold(true)
	helpCmd  = lipgloss.NewStyle().Foreground(lipgloss.Color("#e05252")).Bold(true)
	helpDesc = lipgloss.NewStyle().Foreground(lipgloss.Color("#9ca3af"))
)

var Version = "0.3.2"

var versionFlag bool
var upgradeFlag bool

var rootCmd = &cobra.Command{
	Use:   "oops [N]",
	Short: "Terminal undo — restore your last destructive command",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runUndo,
}

func init() {
	rootCmd.SetHelpFunc(customHelp)
	rootCmd.Flags().BoolVarP(&versionFlag, "version", "v", false, "Print version")
	rootCmd.Flags().BoolVar(&upgradeFlag, "upgrade", false, "Upgrade oops to the latest version")
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
	printCmd("oops size", "show backup disk usage")
	printCmd("oops clean", "remove old backups")
	printCmd("oops config", "view or change settings")
	printCmd("oops doctor", "check installation health")
	printCmd("oops init <shell>", "print shell hook (zsh, bash, fish)")
	printCmd("oops tutorial", "interactive walkthrough")
	printCmd("oops uninstall", "remove oops from your system")
	fmt.Println()
	fmt.Println("  " + helpBold.Render("Flags"))
	printCmd("--version, -v", "print version")
	printCmd("--upgrade", "upgrade to the latest version")
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
		return undoGit(entry)
	}

	if entry.TrashDir == "" {
		return fmt.Errorf("no backup found for this action")
	}

	restored, err := trash.Restore(entry.TrashDir)
	if err != nil {
		return fmt.Errorf("restoring files: %w", err)
	}

	if err := journal.MarkUndone(entry.ID); err != nil {
		fmt.Fprintln(os.Stderr, style.Warning("Could not mark entry as undone: "+err.Error()))
	}

	fmt.Println(style.Success(fmt.Sprintf("Undid: %s", entry.Desc)))
	for _, f := range restored {
		fmt.Println(style.Restored(f))
	}

	return nil
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
		fmt.Println(style.Success(fmt.Sprintf("Undid: %s", entry.Desc)))
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
		fmt.Println(style.Success(fmt.Sprintf("Undid: %s", entry.Desc)))
		fmt.Println(style.Green.Render("  Restored branch: ") + branchName + " at " + entry.GitSHA[:8])

	default:
		return fmt.Errorf("unknown git action: %s", entry.GitAction)
	}

	return nil
}
