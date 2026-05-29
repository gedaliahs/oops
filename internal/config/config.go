package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	RetentionHours  int             `json:"retention_hours"`
	MaxTrashBytes   int64           `json:"max_trash_bytes"`
	RiskWarning     bool            `json:"risk_warning"`
	ConfirmMode     string          `json:"confirm_mode"` // "off", "high", "all"
	OnboardingHints bool            `json:"onboarding_hints"`
	ProtectedPaths  []ProtectedPath `json:"protected_paths,omitempty"`
}

type ProtectedPath struct {
	Path           string `json:"path"`
	AlwaysConfirm  bool   `json:"always_confirm,omitempty"`
	RetentionHours int    `json:"retention_hours,omitempty"`
}

var Default = Config{
	RetentionHours:  2,
	MaxTrashBytes:   5 * 1024 * 1024 * 1024, // 5GB
	RiskWarning:     true,
	ConfirmMode:     "off",
	OnboardingHints: true,
}

type diskConfig struct {
	RetentionHours  *int             `json:"retention_hours"`
	RetentionDays   *int             `json:"retention_days"` // Legacy, migrated on load.
	MaxTrashBytes   *int64           `json:"max_trash_bytes"`
	RiskWarning     *bool            `json:"risk_warning"`
	ConfirmMode     *string          `json:"confirm_mode"`
	OnboardingHints *bool            `json:"onboarding_hints"`
	ProtectedPaths  *[]ProtectedPath `json:"protected_paths"`
}

func OopsDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".oops")
}

func TrashDir() string {
	return filepath.Join(OopsDir(), "trash")
}

func JournalPath() string {
	return filepath.Join(OopsDir(), "journal.jsonl")
}

func ConfigPath() string {
	return filepath.Join(OopsDir(), "config.json")
}

func LastCleanupPath() string {
	return filepath.Join(OopsDir(), ".last_cleanup")
}

func CatchCountPath() string {
	return filepath.Join(OopsDir(), ".catch_count")
}

func SeenUndoPath() string {
	return filepath.Join(OopsDir(), ".seen_undo")
}

func Load() Config {
	cfg := Default
	data, err := os.ReadFile(ConfigPath())
	if err != nil {
		return cfg
	}
	var disk diskConfig
	if err := json.Unmarshal(data, &disk); err != nil {
		return cfg
	}
	if disk.RetentionHours != nil {
		cfg.RetentionHours = *disk.RetentionHours
	} else if disk.RetentionDays != nil {
		cfg.RetentionHours = *disk.RetentionDays * 24
	}
	if disk.MaxTrashBytes != nil {
		cfg.MaxTrashBytes = *disk.MaxTrashBytes
	}
	if disk.RiskWarning != nil {
		cfg.RiskWarning = *disk.RiskWarning
	}
	if disk.ConfirmMode != nil {
		cfg.ConfirmMode = *disk.ConfirmMode
	}
	if disk.OnboardingHints != nil {
		cfg.OnboardingHints = *disk.OnboardingHints
	}
	if disk.ProtectedPaths != nil {
		cfg.ProtectedPaths = normalizeProtectedPaths(*disk.ProtectedPaths)
	}
	if cfg.RetentionHours <= 0 {
		cfg.RetentionHours = Default.RetentionHours
	}
	if cfg.MaxTrashBytes <= 0 {
		cfg.MaxTrashBytes = Default.MaxTrashBytes
	}
	if cfg.ConfirmMode != "off" && cfg.ConfirmMode != "high" && cfg.ConfirmMode != "all" {
		cfg.ConfirmMode = Default.ConfirmMode
	}
	return cfg
}

func Save(cfg Config) error {
	if err := os.MkdirAll(OopsDir(), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(ConfigPath(), data, 0o644)
}

func Get(key string) string {
	cfg := Load()
	switch key {
	case "retention_hours":
		return strconv.Itoa(cfg.RetentionHours)
	case "max_trash_bytes":
		return strconv.FormatInt(cfg.MaxTrashBytes, 10)
	case "risk_warning":
		return strconv.FormatBool(cfg.RiskWarning)
	case "confirm_mode":
		return cfg.ConfirmMode
	case "onboarding_hints":
		return strconv.FormatBool(cfg.OnboardingHints)
	case "protected_paths":
		return strconv.Itoa(len(cfg.ProtectedPaths))
	default:
		return ""
	}
}

func Set(key, value string) error {
	cfg := Load()
	switch key {
	case "retention_hours":
		n, err := strconv.Atoi(value)
		if err != nil || n <= 0 {
			return fmt.Errorf("invalid value for retention_hours: %s", value)
		}
		cfg.RetentionHours = n
	case "retention_days":
		n, err := strconv.Atoi(value)
		if err != nil || n <= 0 {
			return fmt.Errorf("invalid value for retention_days: %s", value)
		}
		cfg.RetentionHours = n * 24
	case "max_trash_bytes":
		n, err := strconv.ParseInt(value, 10, 64)
		if err != nil || n <= 0 {
			return fmt.Errorf("invalid value for max_trash_bytes: %s", value)
		}
		cfg.MaxTrashBytes = n
	case "risk_warning":
		cfg.RiskWarning = value == "true" || value == "1"
	case "confirm_mode":
		if value != "off" && value != "high" && value != "all" {
			return fmt.Errorf("invalid confirm_mode: %s (use off, high, or all)", value)
		}
		cfg.ConfirmMode = value
	case "onboarding_hints":
		cfg.OnboardingHints = value == "true" || value == "1"
	default:
		return fmt.Errorf("unknown key: %s", key)
	}
	return Save(cfg)
}

func ApplyPreset(name string) (Config, error) {
	cfg := Load()
	switch name {
	case "normal":
		cfg.RetentionHours = Default.RetentionHours
		cfg.RiskWarning = Default.RiskWarning
		cfg.ConfirmMode = Default.ConfirmMode
		cfg.OnboardingHints = true
	case "cautious":
		cfg.RetentionHours = 24
		cfg.RiskWarning = true
		cfg.ConfirmMode = "high"
		cfg.OnboardingHints = true
	case "agent":
		cfg.RetentionHours = 6
		cfg.RiskWarning = true
		cfg.ConfirmMode = "all"
		cfg.OnboardingHints = false
	case "quiet":
		cfg.RetentionHours = 2
		cfg.RiskWarning = false
		cfg.ConfirmMode = "off"
		cfg.OnboardingHints = false
	default:
		return cfg, fmt.Errorf("unknown preset: %s (use normal, cautious, agent, or quiet)", name)
	}
	return cfg, Save(cfg)
}

func AddProtectedPath(path string, alwaysConfirm bool, retentionHours int) (ProtectedPath, error) {
	if retentionHours < 0 {
		return ProtectedPath{}, fmt.Errorf("retention_hours must be positive")
	}
	normalized, err := NormalizePath(path)
	if err != nil {
		return ProtectedPath{}, err
	}

	cfg := Load()
	rule := ProtectedPath{
		Path:           normalized,
		AlwaysConfirm:  alwaysConfirm,
		RetentionHours: retentionHours,
	}
	replaced := false
	for i := range cfg.ProtectedPaths {
		if cfg.ProtectedPaths[i].Path == normalized {
			cfg.ProtectedPaths[i] = rule
			replaced = true
			break
		}
	}
	if !replaced {
		cfg.ProtectedPaths = append(cfg.ProtectedPaths, rule)
	}
	return rule, Save(cfg)
}

func RemoveProtectedPath(path string) (bool, error) {
	normalized, err := NormalizePath(path)
	if err != nil {
		return false, err
	}

	cfg := Load()
	filtered := cfg.ProtectedPaths[:0]
	removed := false
	for _, rule := range cfg.ProtectedPaths {
		if rule.Path == normalized {
			removed = true
			continue
		}
		filtered = append(filtered, rule)
	}
	cfg.ProtectedPaths = filtered
	if !removed {
		return false, nil
	}
	return true, Save(cfg)
}

func (cfg Config) MatchProtectedPath(files []string, cwd string) (ProtectedPath, bool) {
	if len(cfg.ProtectedPaths) == 0 {
		return ProtectedPath{}, false
	}

	candidates := files
	if len(candidates) == 0 && cwd != "" {
		candidates = []string{cwd}
	}

	for _, candidate := range candidates {
		normalized, err := normalizePathFrom(candidate, cwd)
		if err != nil {
			continue
		}
		for _, rule := range cfg.ProtectedPaths {
			if pathContains(rule.Path, normalized) {
				return rule, true
			}
		}
	}

	return ProtectedPath{}, false
}

func NormalizePath(path string) (string, error) {
	return normalizePathFrom(path, "")
}

func normalizeProtectedPaths(rules []ProtectedPath) []ProtectedPath {
	normalized := make([]ProtectedPath, 0, len(rules))
	seen := make(map[string]bool, len(rules))
	for _, rule := range rules {
		if rule.Path == "" {
			continue
		}
		path, err := NormalizePath(rule.Path)
		if err != nil || seen[path] {
			continue
		}
		if rule.RetentionHours < 0 {
			rule.RetentionHours = 0
		}
		rule.Path = path
		normalized = append(normalized, rule)
		seen[path] = true
	}
	return normalized
}

func normalizePathFrom(path, cwd string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path is required")
	}
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		path = filepath.Join(home, path[2:])
	}
	if !filepath.IsAbs(path) {
		base := cwd
		if base == "" {
			var err error
			base, err = os.Getwd()
			if err != nil {
				return "", err
			}
		}
		path = filepath.Join(base, path)
	}
	return filepath.Clean(path), nil
}

func pathContains(root, target string) bool {
	root = filepath.Clean(root)
	target = filepath.Clean(target)
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) && !filepath.IsAbs(rel))
}

func (cfg Config) RetentionDuration() time.Duration {
	return time.Duration(cfg.RetentionHours) * time.Hour
}

func EnsureDir() error {
	if err := os.MkdirAll(OopsDir(), 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", OopsDir(), err)
	}
	if err := os.MkdirAll(TrashDir(), 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", TrashDir(), err)
	}
	return CheckWritable()
}

func CheckWritable() error {
	for _, dir := range []string{OopsDir(), TrashDir()} {
		if err := checkDirWritable(dir); err != nil {
			return fmt.Errorf("%s is not writable: %w (fix: %s)", dir, err, PermissionFixCommand())
		}
	}
	return nil
}

func PermissionFixCommand() string {
	user := os.Getenv("SUDO_USER")
	if user == "" || user == "root" {
		user = os.Getenv("USER")
	}
	if user == "" {
		user = "$(id -un)"
	}
	return "sudo chown -R " + shellQuote(user) + " " + shellQuote(OopsDir())
}

func checkDirWritable(dir string) error {
	f, err := os.CreateTemp(dir, ".write-test-*")
	if err != nil {
		return err
	}
	name := f.Name()
	if err := f.Close(); err != nil {
		_ = os.Remove(name)
		return err
	}
	return os.Remove(name)
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func ShouldCleanup() bool {
	data, err := os.ReadFile(LastCleanupPath())
	if err != nil {
		return true
	}
	t, err := time.Parse(time.RFC3339, string(data))
	if err != nil {
		return true
	}
	return time.Since(t) > time.Hour
}

func MarkCleanup() {
	_ = os.WriteFile(LastCleanupPath(), []byte(time.Now().Format(time.RFC3339)), 0o644)
}

// CatchCount returns how many destructive commands oops has caught for this user,
// used to fade the onboarding hint. Returns 0 when unreadable.
func CatchCount() int {
	data, err := os.ReadFile(CatchCountPath())
	if err != nil {
		return 0
	}
	n, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || n < 0 {
		return 0
	}
	return n
}

// IncrementCatchCount bumps the catch counter and returns the new value. Failures
// are tolerated silently (the counter is best-effort onboarding state, not data).
func IncrementCatchCount() int {
	n := CatchCount() + 1
	if err := os.MkdirAll(OopsDir(), 0o755); err == nil {
		_ = os.WriteFile(CatchCountPath(), []byte(strconv.Itoa(n)), 0o644)
	}
	return n
}

// HasSeenUndo reports whether the user has ever successfully run an undo. Once
// true, onboarding hints stop entirely and the catch path takes a read-only fast
// path that performs no writes.
func HasSeenUndo() bool {
	_, err := os.Stat(SeenUndoPath())
	return err == nil
}

// MarkSeenUndo records that the user has completed an undo. Best-effort.
func MarkSeenUndo() {
	if err := os.MkdirAll(OopsDir(), 0o755); err == nil {
		_ = os.WriteFile(SeenUndoPath(), []byte(time.Now().Format(time.RFC3339)), 0o644)
	}
}
