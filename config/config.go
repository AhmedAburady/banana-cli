package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config holds the application configuration
type Config struct {
	APIKey string `json:"api_key"`
}

// DefaultConfigDir returns the default config directory
func DefaultConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "banana")
}

// DefaultConfigPath returns the default config file path
func DefaultConfigPath() string {
	return filepath.Join(DefaultConfigDir(), "config.json")
}

// Load reads the config from the default location
func Load() (*Config, error) {
	path := DefaultConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Save writes the config to the default location
func Save(cfg *Config) error {
	dir := DefaultConfigDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(DefaultConfigPath(), data, 0600)
}

// SaveAPIKey saves just the API key
func SaveAPIKey(key string) error {
	cfg, err := Load()
	if err != nil {
		cfg = &Config{}
	}
	cfg.APIKey = key
	return Save(cfg)
}

// GetAPIKey returns the API key from env vars or config file
// Priority: GEMINI_API_KEY > GOOGLE_API_KEY > config file
func GetAPIKey() string {
	// Check environment variables first
	if key := os.Getenv("GEMINI_API_KEY"); key != "" {
		return key
	}
	if key := os.Getenv("GOOGLE_API_KEY"); key != "" {
		return key
	}

	// Fall back to config file
	cfg, err := Load()
	if err != nil {
		return ""
	}
	return cfg.APIKey
}
