package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type Config struct {
	RetentionDays int    `json:"retention_days"`
	MaxTrashBytes int64  `json:"max_trash_bytes"`
	RiskWarning   bool   `json:"risk_warning"`
	ConfirmMode   string `json:"confirm_mode"` // "off", "high", "all"
}

var Default = Config{
	RetentionDays: 7,
	MaxTrashBytes: 5 * 1024 * 1024 * 1024, // 5GB
	RiskWarning:   true,
	ConfirmMode:   "off",
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

func Load() Config {
	cfg := Default
	data, err := os.ReadFile(ConfigPath())
	if err != nil {
		return cfg
	}
	_ = json.Unmarshal(data, &cfg)
	if cfg.RetentionDays <= 0 {
		cfg.RetentionDays = Default.RetentionDays
	}
	if cfg.MaxTrashBytes <= 0 {
		cfg.MaxTrashBytes = Default.MaxTrashBytes
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
	case "retention_days":
		return strconv.Itoa(cfg.RetentionDays)
	case "max_trash_bytes":
		return strconv.FormatInt(cfg.MaxTrashBytes, 10)
	case "risk_warning":
		return strconv.FormatBool(cfg.RiskWarning)
	case "confirm_mode":
		return cfg.ConfirmMode
	default:
		return ""
	}
}

func Set(key, value string) error {
	cfg := Load()
	switch key {
	case "retention_days":
		n, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid value for retention_days: %s", value)
		}
		cfg.RetentionDays = n
	case "max_trash_bytes":
		n, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
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
	}
	return Save(cfg)
}

func EnsureDir() error {
	if err := os.MkdirAll(OopsDir(), 0o755); err != nil {
		return err
	}
	return os.MkdirAll(TrashDir(), 0o755)
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

