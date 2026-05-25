package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"auto_ballchasing/config"
)

func withTempConfig(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	orig, _ := os.UserConfigDir()
	os.Setenv("APPDATA", dir)
	t.Cleanup(func() { os.Setenv("APPDATA", orig) })
	return filepath.Join(dir, "auto_ballchasing", "config.json")
}

func TestLoadReturnsDefaultWhenMissing(t *testing.T) {
	withTempConfig(t)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if cfg.APIKey != "" {
		t.Errorf("expected empty API key, got: %s", cfg.APIKey)
	}
	if cfg.Visibility != "public" {
		t.Errorf("expected default visibility 'public', got: %s", cfg.Visibility)
	}
}

func TestSaveAndLoad(t *testing.T) {
	withTempConfig(t)

	original := &config.Config{
		APIKey:     "test-api-key-123",
		Visibility: "unlisted",
	}

	if err := original.Save(); err != nil {
		t.Fatalf("could not save config: %v", err)
	}

	loaded, err := config.Load()
	if err != nil {
		t.Fatalf("could not load config: %v", err)
	}

	if loaded.APIKey != original.APIKey {
		t.Errorf("expected API key '%s', got '%s'", original.APIKey, loaded.APIKey)
	}
	if loaded.Visibility != original.Visibility {
		t.Errorf("expected visibility '%s', got '%s'", original.Visibility, loaded.Visibility)
	}
}

func TestIsValid(t *testing.T) {
	empty := &config.Config{}
	if empty.IsValid() {
		t.Error("expected empty config to be invalid")
	}

	valid := &config.Config{APIKey: "some-key"}
	if !valid.IsValid() {
		t.Error("expected config with API key to be valid")
	}
}

func TestSaveCreatesDirectory(t *testing.T) {
	configPath := withTempConfig(t)

	cfg := &config.Config{APIKey: "test-key", Visibility: "public"}
	if err := cfg.Save(); err != nil {
		t.Fatalf("could not save: %v", err)
	}

	if _, err := os.Stat(configPath); err != nil {
		t.Errorf("expected config file to exist at %s: %v", configPath, err)
	}
}
