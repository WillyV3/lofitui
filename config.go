package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Preset represents a single lofi stream
type Preset struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// Config represents the application configuration
type Config struct {
	Presets []Preset `json:"presets"`
}

// getConfigDir returns the config directory path following XDG spec
func getConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	// Use XDG_CONFIG_HOME if set, otherwise use ~/.config
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		configHome = filepath.Join(homeDir, ".config")
	}

	configDir := filepath.Join(configHome, "lofitui")
	return configDir, nil
}

// getConfigPath returns the full path to the config file
func getConfigPath() (string, error) {
	configDir, err := getConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "config.json"), nil
}

// loadConfig loads configuration from disk or returns defaults
func loadConfig() (*Config, error) {
	configPath, err := getConfigPath()
	if err != nil {
		return nil, err
	}

	// If config doesn't exist, return defaults
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return getDefaultConfig(), nil
	}

	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	// Parse JSON
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &config, nil
}

// saveConfig saves configuration to disk
func saveConfig(config *Config) error {
	configDir, err := getConfigDir()
	if err != nil {
		return err
	}

	// Create config directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	configPath, err := getConfigPath()
	if err != nil {
		return err
	}

	// Marshal config to JSON with indentation
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// getDefaultConfig returns the default configuration
func getDefaultConfig() *Config {
	return &Config{
		Presets: []Preset{
			{Name: "Lofi Girl - Study", URL: "https://www.youtube.com/watch?v=jfKfPfyJRdk"},
			{Name: "Lofi Girl - Sleep", URL: "https://www.youtube.com/watch?v=DWcJFNfaw9c"},
			{Name: "Lofi Girl - Jazz", URL: "https://www.youtube.com/watch?v=HuFYqnbVbzY"},
			{Name: "Synthwave Radio", URL: "https://www.youtube.com/watch?v=4xDzrJKXOOY"},
			{Name: "Chillhop Music", URL: "https://www.youtube.com/watch?v=5yx6BWlEVcY"},
			{Name: "The Bootleg Boy", URL: "https://www.youtube.com/watch?v=FWjZ0x2M8og"},
			{Name: "Dreamhop Music", URL: "https://www.youtube.com/live/D5bqo8lcny4"},
			{Name: "Lofi Geek", URL: "https://www.youtube.com/watch?v=1tJ8sc8I4z0"},
			{Name: "STEEZYASFUCK", URL: "https://www.youtube.com/watch?v=S_MOd40zlYU"},
			{Name: "Homework Radio", URL: "https://www.youtube.com/watch?v=lTRiuFIWV54"},
		},
	}
}
