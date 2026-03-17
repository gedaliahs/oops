package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var tutorialCmd = &cobra.Command{
	Use:   "tutorial",
	Short: "Interactive tutorial to see oops in action",
	RunE:  runTutorial,
}

func init() {
	rootCmd.AddCommand(tutorialCmd)
}

var (
	tutDim   = lipgloss.NewStyle().Foreground(lipgloss.Color("#6b7280"))
	tutBold  = lipgloss.NewStyle().Bold(true)
	tutRed   = lipgloss.NewStyle().Foreground(lipgloss.Color("#e05252")).Bold(true)
	tutGreen = lipgloss.NewStyle().Foreground(lipgloss.Color("#22c55e"))
	tutCyan  = lipgloss.NewStyle().Foreground(lipgloss.Color("#06b6d4"))
)

func pause() {
	fmt.Print(tutDim.Render("  press enter to continue..."))
	fmt.Scanln()
}

func runTutorial(cmd *cobra.Command, args []string) error {
	cwd, _ := os.Getwd()
	testFile := filepath.Join(cwd, "oops-tutorial.txt")
	content := "This is an important file.\nIt contains data you don't want to lose.\nCreated by oops tutorial at " + time.Now().Format("3:04 PM") + ".\n"

	// Step 1: Create the file
	fmt.Println()
	fmt.Println("  " + tutRed.Render("oops") + tutDim.Render(" tutorial"))
	fmt.Println()
	fmt.Println("  " + tutBold.Render("Step 1:") + " Creating a test file")
	fmt.Println()

	if err := os.WriteFile(testFile, []byte(content), 0o644); err != nil {
		return fmt.Errorf("could not create test file: %w", err)
	}

	fmt.Println("  " + tutGreen.Render("✓") + " Created " + tutCyan.Render("oops-tutorial.txt"))
	fmt.Println()
	fmt.Println(tutDim.Render("  Contents:"))
	fmt.Println(tutDim.Render("  ─────────────────────────────────"))
	fmt.Println(tutDim.Render("  This is an important file."))
	fmt.Println(tutDim.Render("  It contains data you don't want to lose."))
	fmt.Println(tutDim.Render("  ─────────────────────────────────"))
	fmt.Println()
	pause()

	// Step 2: Delete it (with oops protect)
	fmt.Println()
	fmt.Println("  " + tutBold.Render("Step 2:") + " Deleting the file (oops will back it up first)")
	fmt.Println()
	fmt.Println("  " + tutDim.Render("$") + " rm oops-tutorial.txt")
	fmt.Println()

	// Run protect manually
	doProtect("rm " + testFile)

	// Actually delete it
	os.Remove(testFile)

	// Verify it's gone
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		fmt.Println("  " + tutDim.Render("File is gone. Try") + " ls oops-tutorial.txt " + tutDim.Render("— it won't be there."))
	}
	fmt.Println()
	pause()

	// Step 3: Restore it
	fmt.Println()
	fmt.Println("  " + tutBold.Render("Step 3:") + " Restoring the file")
	fmt.Println()
	fmt.Println("  " + tutDim.Render("$") + " " + tutRed.Render("oops"))
	fmt.Println()

	// Run the actual undo
	err := runUndo(cmd, []string{})
	if err != nil {
		return fmt.Errorf("restore failed: %w", err)
	}

	fmt.Println()

	// Verify it's back
	if _, err := os.Stat(testFile); err == nil {
		data, _ := os.ReadFile(testFile)
		fmt.Println("  " + tutBold.Render("File is back!") + " Contents:")
		fmt.Println()
		fmt.Println(tutDim.Render("  ─────────────────────────────────"))
		for _, line := range splitLines(string(data)) {
			if line != "" {
				fmt.Println(tutDim.Render("  " + line))
			}
		}
		fmt.Println(tutDim.Render("  ─────────────────────────────────"))
	}

	fmt.Println()
	fmt.Println("  " + tutGreen.Render("✓") + " " + tutBold.Render("That's it.") + " oops works in the background — just use your")
	fmt.Println("    terminal normally and type " + tutRed.Render("oops") + " when you need to undo.")
	fmt.Println()
	fmt.Println(tutDim.Render("  Other commands:"))
	fmt.Println(tutDim.Render("    oops log        see undo history"))
	fmt.Println(tutDim.Render("    oops 2          undo second-to-last"))
	fmt.Println(tutDim.Render("    oops clean      free up backup space"))
	fmt.Println()

	// Clean up the tutorial file
	os.Remove(testFile)

	return nil
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
