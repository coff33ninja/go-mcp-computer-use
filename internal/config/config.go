package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	LogLevel       string `json:"log_level"`
	MouseSpeed     int    `json:"mouse_speed"`
	ClickDelay     int    `json:"click_delay_ms"`
	VerifyBounds   bool   `json:"verify_bounds"`
	ActionTimeoutMs int   `json:"action_timeout_ms"`
	UIAWarmup      bool   `json:"uia_warmup"`
}

func Default() *Config {
	return &Config{
		LogLevel:        "info",
		MouseSpeed:      500,
		ClickDelay:      100,
		VerifyBounds:    true,
		ActionTimeoutMs: 30000,
		UIAWarmup:       true,
	}
}

func (c *Config) LogLevelSlog() int {
	switch c.LogLevel {
	case "debug":
		return -4
	case "info":
		return 0
	case "warn":
		return 4
	case "error":
		return 8
	default:
		return 0
	}
}

func Load() (*Config, error) {
	cfg := Default()

	home, err := os.UserHomeDir()
	if err != nil {
		return cfg, nil
	}

	configDir := filepath.Join(home, ".config", "go-mcp-computer-use")
	configPath := filepath.Join(configDir, "config.json")

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("read config: %w", err)
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return cfg, fmt.Errorf("parse config: %w", err)
	}

	return cfg, nil
}

func (c *Config) Save() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("home dir: %w", err)
	}

	configDir := filepath.Join(home, ".config", "go-mcp-computer-use")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	configPath := filepath.Join(configDir, "config.json")
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return nil
}
