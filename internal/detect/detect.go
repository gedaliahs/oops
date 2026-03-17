package detect

import "os"

// Analyze takes a full shell command string and returns all Protection actions needed.
func Analyze(command string) []*Protection {
	tokens := Tokenize(command)
	if len(tokens) == 0 {
		return nil
	}

	// Split on command separators first
	commands := SplitCommands(tokens)

	var protections []*Protection
	for _, cmd := range commands {
		// For pipelines, only the last command can write to files,
		// but we check all segments for destructive commands.
		segments := SplitPipeline(cmd)
		for _, seg := range segments {
			if p := analyzeSimple(seg); p != nil {
				protections = append(protections, p)
			}
		}
	}

	// Check for redirects in the original tokens
	if p := detectRedirect(tokens); p != nil {
		protections = append(protections, p)
	}

	return protections
}

// analyzeSimple analyzes a single, simple command (no pipes/separators).
func analyzeSimple(tokens []string) *Protection {
	if len(tokens) == 0 {
		return nil
	}

	// Skip env var assignments (FOO=bar cmd ...)
	i := 0
	for i < len(tokens) && isEnvAssignment(tokens[i]) {
		i++
	}
	if i >= len(tokens) {
		return nil
	}

	cmd := tokens[i]
	args := tokens[i+1:]

	// Strip path prefix from command
	cmd = basename(cmd)

	switch cmd {
	case "rm":
		return ParseRM(args)
	case "mv":
		return ParseMV(args)
	case "sed", "gsed":
		return ParseSed(args)
	case "chmod":
		return ParseChmod(args)
	case "chown":
		return ParseChown(args)
	case "truncate", "gtruncate":
		return ParseTruncate(args)
	case "git":
		return ParseGit(args)
	case "sudo":
		// Recurse with args to handle "sudo rm -rf /"
		if len(args) > 0 {
			return analyzeSimple(args)
		}
	}

	return nil
}

// detectRedirect checks for file-overwriting redirects (> file).
func detectRedirect(tokens []string) *Protection {
	for i, t := range tokens {
		if t == ">" && i+1 < len(tokens) {
			target := tokens[i+1]
			// Skip fd redirects like >&2
			if target == "&1" || target == "&2" || target == "/dev/null" {
				continue
			}
			resolved := resolvePath(target)
			if _, err := os.Lstat(resolved); err == nil {
				return &Protection{
					Action: ActionRedirect,
					Risk:   RiskMedium,
					Files:  []string{resolved},
					Desc:   "> " + target,
				}
			}
		}
	}
	return nil
}

func isEnvAssignment(s string) bool {
	for i, c := range s {
		if c == '=' && i > 0 {
			return true
		}
		if !isIdentChar(c, i == 0) {
			return false
		}
	}
	return false
}

func isIdentChar(c rune, first bool) bool {
	if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c == '_' {
		return true
	}
	if !first && c >= '0' && c <= '9' {
		return true
	}
	return false
}

func basename(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			return path[i+1:]
		}
	}
	return path
}

// QuickMatch does a fast check to see if a command string could be destructive.
// This is used by the shell hook to avoid spawning a subprocess for safe commands.
// It returns the list of command names that matched.
func QuickMatch(command string) []string {
	destructive := []string{
		"rm", "mv", "sed", "gsed", "chmod", "chown", "truncate", "gtruncate",
		"git", "dd", "shred",
	}

	var matches []string
	tokens := Tokenize(command)
	for _, t := range tokens {
		b := basename(t)
		for _, d := range destructive {
			if b == d {
				matches = append(matches, d)
			}
		}
	}

	// Also check for redirects
	for _, t := range tokens {
		if t == ">" {
			matches = append(matches, ">")
		}
	}

	return matches
}
