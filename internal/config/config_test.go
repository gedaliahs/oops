package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupTestConfig(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
}

func TestLoadDefaultRetentionHours(t *testing.T) {
	setupTestConfig(t)

	cfg := Load()
	if cfg.RetentionHours != 2 {
		t.Fatalf("expected default retention to be 2 hours, got %d", cfg.RetentionHours)
	}
}

func TestLoadMigratesLegacyRetentionDays(t *testing.T) {
	setupTestConfig(t)
	if err := os.MkdirAll(OopsDir(), 0o755); err != nil {
		t.Fatal(err)
	}
	data := []byte(`{"retention_days":3,"max_trash_bytes":1024,"risk_warning":false,"confirm_mode":"all"}`)
	if err := os.WriteFile(ConfigPath(), data, 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := Load()
	if cfg.RetentionHours != 72 {
		t.Fatalf("expected legacy retention_days to migrate to 72 hours, got %d", cfg.RetentionHours)
	}
	if cfg.MaxTrashBytes != 1024 {
		t.Fatalf("expected max_trash_bytes=1024, got %d", cfg.MaxTrashBytes)
	}
	if cfg.RiskWarning {
		t.Fatal("expected risk_warning=false to be preserved")
	}
	if cfg.ConfirmMode != "all" {
		t.Fatalf("expected confirm_mode=all, got %q", cfg.ConfirmMode)
	}
}

func TestSetRetentionHoursSavesNewKey(t *testing.T) {
	setupTestConfig(t)

	if err := Set("retention_hours", "6"); err != nil {
		t.Fatal(err)
	}

	cfg := Load()
	if cfg.RetentionHours != 6 {
		t.Fatalf("expected retention_hours=6, got %d", cfg.RetentionHours)
	}

	data, err := os.ReadFile(ConfigPath())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "retention_days") {
		t.Fatalf("config should not save legacy retention_days key: %s", data)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	if _, ok := raw["retention_hours"]; !ok {
		t.Fatalf("config missing retention_hours key: %s", data)
	}
}

func TestApplyPresetAgent(t *testing.T) {
	setupTestConfig(t)

	cfg, err := ApplyPreset("agent")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.RetentionHours != 6 || !cfg.RiskWarning || cfg.ConfirmMode != "all" {
		t.Fatalf("unexpected agent preset: %+v", cfg)
	}
}

func TestApplyPresetNormal(t *testing.T) {
	setupTestConfig(t)

	if _, err := ApplyPreset("agent"); err != nil {
		t.Fatal(err)
	}
	cfg, err := ApplyPreset("normal")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.RetentionHours != Default.RetentionHours || !cfg.RiskWarning || cfg.ConfirmMode != Default.ConfirmMode {
		t.Fatalf("unexpected normal preset: %+v", cfg)
	}
}

func TestProtectedPathMatchesChildren(t *testing.T) {
	setupTestConfig(t)
	root := filepath.Join(t.TempDir(), "project")
	child := filepath.Join(root, "src", "file.txt")
	if err := os.MkdirAll(filepath.Dir(child), 0o755); err != nil {
		t.Fatal(err)
	}

	rule, err := AddProtectedPath(root, true, 24)
	if err != nil {
		t.Fatal(err)
	}
	if !rule.AlwaysConfirm || rule.RetentionHours != 24 {
		t.Fatalf("unexpected rule: %+v", rule)
	}

	cfg := Load()
	matched, ok := cfg.MatchProtectedPath([]string{child}, "")
	if !ok {
		t.Fatal("expected child path to match protected path")
	}
	if matched.Path != filepath.Clean(root) {
		t.Fatalf("expected %s, got %s", root, matched.Path)
	}
}

func TestRemoveProtectedPath(t *testing.T) {
	setupTestConfig(t)
	root := t.TempDir()
	if _, err := AddProtectedPath(root, false, 0); err != nil {
		t.Fatal(err)
	}

	removed, err := RemoveProtectedPath(root)
	if err != nil {
		t.Fatal(err)
	}
	if !removed {
		t.Fatal("expected rule to be removed")
	}
	if len(Load().ProtectedPaths) != 0 {
		t.Fatalf("expected no protected paths, got %+v", Load().ProtectedPaths)
	}
}
