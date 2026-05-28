package config

import (
	"encoding/json"
	"os"
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
