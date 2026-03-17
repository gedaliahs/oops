package detect

import (
	"os"
	"path/filepath"
	"strings"
)

// RiskLevel indicates the severity of a destructive operation.
type RiskLevel int

const (
	RiskNone   RiskLevel = 0
	RiskLow    RiskLevel = 1
	RiskMedium RiskLevel = 2
	RiskHigh   RiskLevel = 3
)

func (r RiskLevel) String() string {
	switch r {
	case RiskLow:
		return "low"
	case RiskMedium:
		return "medium"
	case RiskHigh:
		return "high"
	default:
		return "none"
	}
}

// ActionType describes the kind of destructive action.
type ActionType string

const (
	ActionRM       ActionType = "rm"
	ActionMV       ActionType = "mv"
	ActionSed      ActionType = "sed"
	ActionChmod    ActionType = "chmod"
	ActionChown    ActionType = "chown"
	ActionTruncate ActionType = "truncate"
	ActionGit      ActionType = "git"
	ActionRedirect ActionType = "redirect"
)

// Protection describes what needs to be backed up and how risky it is.
type Protection struct {
	Action ActionType
	Risk   RiskLevel
	Files  []string // Files/dirs to back up before the command runs
	Desc   string   // Human-readable description of the action

	// Git-specific fields
	GitAction string // "stash", "log-branch"
	GitRef    string // Branch name or ref
}

// ParseRM analyzes an rm command and returns a Protection.
func ParseRM(args []string) *Protection {
	var paths []string
	recursive := false

	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			paths = append(paths, args[i+1:]...)
			break
		}
		if strings.HasPrefix(a, "-") && a != "-" {
			if strings.Contains(a, "r") || strings.Contains(a, "R") || a == "--recursive" {
				recursive = true
			}
		} else {
			paths = append(paths, a)
		}
	}

	if len(paths) == 0 {
		return nil
	}

	resolved := resolvePaths(paths)
	if len(resolved) == 0 {
		return nil
	}

	risk := RiskMedium
	if recursive {
		risk = RiskHigh
	}

	// Reduce risk for temp/build paths
	allLowRisk := true
	for _, p := range resolved {
		if !isLowRiskPath(p) {
			allLowRisk = false
			break
		}
	}
	if allLowRisk && risk < RiskHigh {
		risk = RiskLow
	}

	// Elevate risk for home/src/config paths
	for _, p := range resolved {
		if isHighRiskPath(p) {
			risk = RiskHigh
			break
		}
	}

	desc := "rm"
	if recursive {
		desc = "rm -r"
	}
	desc += " " + strings.Join(paths, " ")

	return &Protection{
		Action: ActionRM,
		Risk:   risk,
		Files:  resolved,
		Desc:   desc,
	}
}

// ParseMV analyzes an mv command and returns a Protection if the target exists.
func ParseMV(args []string) *Protection {
	var paths []string

	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			paths = append(paths, args[i+1:]...)
			break
		}
		if strings.HasPrefix(a, "-") && a != "-" {
			// Skip flags like -f, -i, -n, -v, --target-directory
			if a == "-t" || a == "--target-directory" {
				i++ // skip next arg (the target dir)
			}
			continue
		}
		paths = append(paths, a)
	}

	if len(paths) < 2 {
		return nil
	}

	// The last path is the destination
	dest := paths[len(paths)-1]
	dest = resolvePath(dest)

	// Only protect if destination exists (would be overwritten)
	info, err := os.Lstat(dest)
	if err != nil {
		return nil // dest doesn't exist, no overwrite
	}

	// If dest is a directory, check if any source files would overwrite files inside it
	var toBackup []string
	if info.IsDir() {
		sources := paths[:len(paths)-1]
		for _, src := range sources {
			base := filepath.Base(src)
			target := filepath.Join(dest, base)
			if _, err := os.Lstat(target); err == nil {
				toBackup = append(toBackup, target)
			}
		}
	} else {
		toBackup = []string{dest}
	}

	if len(toBackup) == 0 {
		return nil
	}

	return &Protection{
		Action: ActionMV,
		Risk:   RiskMedium,
		Files:  toBackup,
		Desc:   "mv overwrite " + dest,
	}
}

// ParseSed analyzes a sed command for in-place edits.
func ParseSed(args []string) *Protection {
	inPlace := false
	var files []string

	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "-i" || strings.HasPrefix(a, "-i") || a == "--in-place" {
			inPlace = true
			continue
		}
		if a == "-e" || a == "--expression" || a == "-f" || a == "--file" {
			i++ // skip the expression/file arg
			continue
		}
		if strings.HasPrefix(a, "-") && a != "-" {
			continue
		}
		// Non-flag arg — could be a script or a file
		// After -i, non-flag args that aren't the first are files
		if inPlace {
			files = append(files, a)
		}
	}

	if !inPlace || len(files) == 0 {
		return nil
	}

	resolved := resolvePaths(files)
	if len(resolved) == 0 {
		return nil
	}

	return &Protection{
		Action: ActionSed,
		Risk:   RiskMedium,
		Files:  resolved,
		Desc:   "sed -i " + strings.Join(files, " "),
	}
}

// ParseChmod analyzes a chmod command.
func ParseChmod(args []string) *Protection {
	var paths []string
	recursive := false

	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			paths = append(paths, args[i+1:]...)
			break
		}
		if a == "-R" || a == "--recursive" {
			recursive = true
			continue
		}
		if strings.HasPrefix(a, "-") && a != "-" {
			continue
		}
		paths = append(paths, a)
	}

	// First non-flag arg is the mode, rest are files
	if len(paths) < 2 {
		return nil
	}
	files := paths[1:] // skip the mode argument

	resolved := resolvePaths(files)
	if len(resolved) == 0 {
		return nil
	}

	risk := RiskLow
	if recursive {
		risk = RiskMedium
	}

	return &Protection{
		Action: ActionChmod,
		Risk:   risk,
		Files:  resolved,
		Desc:   "chmod " + strings.Join(args, " "),
	}
}

// ParseChown analyzes a chown command.
func ParseChown(args []string) *Protection {
	var paths []string
	recursive := false

	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			paths = append(paths, args[i+1:]...)
			break
		}
		if a == "-R" || a == "--recursive" {
			recursive = true
			continue
		}
		if strings.HasPrefix(a, "-") && a != "-" {
			continue
		}
		paths = append(paths, a)
	}

	// First non-flag arg is the owner, rest are files
	if len(paths) < 2 {
		return nil
	}
	files := paths[1:]

	resolved := resolvePaths(files)
	if len(resolved) == 0 {
		return nil
	}

	risk := RiskMedium
	if recursive {
		risk = RiskHigh
	}

	return &Protection{
		Action: ActionChown,
		Risk:   risk,
		Files:  resolved,
		Desc:   "chown " + strings.Join(args, " "),
	}
}

// ParseTruncate analyzes a truncate command.
func ParseTruncate(args []string) *Protection {
	var files []string

	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "-s" || a == "--size" {
			i++ // skip size arg
			continue
		}
		if strings.HasPrefix(a, "-s") || strings.HasPrefix(a, "--size=") {
			continue
		}
		if strings.HasPrefix(a, "-") && a != "-" {
			continue
		}
		files = append(files, a)
	}

	if len(files) == 0 {
		return nil
	}

	resolved := resolvePaths(files)
	if len(resolved) == 0 {
		return nil
	}

	return &Protection{
		Action: ActionTruncate,
		Risk:   RiskMedium,
		Files:  resolved,
		Desc:   "truncate " + strings.Join(files, " "),
	}
}

// ParseGit analyzes destructive git subcommands.
func ParseGit(args []string) *Protection {
	if len(args) == 0 {
		return nil
	}

	sub := args[0]
	rest := args[1:]

	switch sub {
	case "reset":
		return parseGitReset(rest)
	case "checkout":
		return parseGitCheckout(rest)
	case "clean":
		return parseGitClean(rest)
	case "branch":
		return parseGitBranch(rest)
	default:
		return nil
	}
}

func parseGitReset(args []string) *Protection {
	hard := false
	for _, a := range args {
		if a == "--hard" {
			hard = true
		}
	}
	if !hard {
		return nil
	}
	return &Protection{
		Action:    ActionGit,
		Risk:      RiskHigh,
		GitAction: "stash",
		Desc:      "git reset --hard",
	}
}

func parseGitCheckout(args []string) *Protection {
	// Detect: git checkout . or git checkout -- <files>
	for _, a := range args {
		if a == "." {
			return &Protection{
				Action:    ActionGit,
				Risk:      RiskHigh,
				GitAction: "stash",
				Desc:      "git checkout .",
			}
		}
	}
	// git checkout -- <files>
	afterDash := false
	var files []string
	for _, a := range args {
		if a == "--" {
			afterDash = true
			continue
		}
		if afterDash {
			files = append(files, a)
		}
	}
	if len(files) > 0 {
		resolved := resolvePaths(files)
		return &Protection{
			Action:    ActionGit,
			Risk:      RiskMedium,
			GitAction: "stash",
			Files:     resolved,
			Desc:      "git checkout -- " + strings.Join(files, " "),
		}
	}
	return nil
}

func parseGitClean(args []string) *Protection {
	force := false
	for _, a := range args {
		if strings.Contains(a, "f") && strings.HasPrefix(a, "-") {
			force = true
		}
	}
	if !force {
		return nil
	}
	return &Protection{
		Action:    ActionGit,
		Risk:      RiskHigh,
		GitAction: "stash",
		Desc:      "git clean " + strings.Join(args, " "),
	}
}

func parseGitBranch(args []string) *Protection {
	deleteName := ""
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "-D" || a == "--delete" || a == "-d" {
			if i+1 < len(args) {
				deleteName = args[i+1]
			}
		}
	}
	if deleteName == "" {
		return nil
	}
	return &Protection{
		Action:    ActionGit,
		Risk:      RiskMedium,
		GitAction: "log-branch",
		GitRef:    deleteName,
		Desc:      "git branch -D " + deleteName,
	}
}

// resolvePaths resolves a list of paths, filtering out those that don't exist.
func resolvePaths(paths []string) []string {
	var resolved []string
	for _, p := range paths {
		r := resolvePath(p)
		if _, err := os.Lstat(r); err == nil {
			resolved = append(resolved, r)
		}
	}
	return resolved
}

// resolvePath resolves a single path to an absolute path.
func resolvePath(p string) string {
	if strings.HasPrefix(p, "~/") {
		home, _ := os.UserHomeDir()
		p = filepath.Join(home, p[2:])
	}
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return p
	}
	return abs
}

// isLowRiskPath returns true if the path is in a temp/build directory.
func isLowRiskPath(p string) bool {
	lowRiskDirs := []string{
		"/tmp/", "/var/tmp/",
		"node_modules/", ".cache/", "dist/", "build/",
		"__pycache__/", ".pytest_cache/", ".mypy_cache/",
		"target/", ".next/", ".nuxt/",
	}
	for _, d := range lowRiskDirs {
		if strings.Contains(p, d) {
			return true
		}
	}
	return false
}

// isHighRiskPath returns true if the path is in a sensitive directory.
func isHighRiskPath(p string) bool {
	home, _ := os.UserHomeDir()
	highRiskDirs := []string{
		"/src/", "/config/", "/etc/",
		"/.ssh/", "/.gnupg/", "/.aws/",
	}
	// Direct children of home are high risk
	if home != "" && filepath.Dir(p) == home {
		return true
	}
	for _, d := range highRiskDirs {
		if strings.Contains(p, d) {
			return true
		}
	}
	return false
}
