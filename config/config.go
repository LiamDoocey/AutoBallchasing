package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	APIKey       string `json:"api_key"`
	Visibility   string `json:"visibility"`
	RunOnStartup bool   `json:"run_on_startup"`
}

func path() string {
	dir, _ := os.UserConfigDir()
	return filepath.Join(dir, "auto_ballchasing", "config.json")
}

func Load() (*Config, error) {
	data, err := os.ReadFile(path())
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{Visibility: "public"}, nil
		}
		return nil, fmt.Errorf("could not read config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("could not parse config: %w", err)
	}

	if cfg.Visibility == "" {
		cfg.Visibility = "public"
	}

	return &cfg, nil
}

func (c *Config) Save() error {
	p := path()
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return fmt.Errorf("could not create config dir: %w", err)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("could not encode config: %w", err)
	}

	if err := os.WriteFile(p, data, 0600); err != nil {
		return fmt.Errorf("could not write config: %w", err)
	}

	return nil
}

func (c *Config) IsValid() bool {
	return c.APIKey != ""
}
