package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/goccy/go-yaml"
)

// Config represents the deck-pb configuration file.
type Config struct {
	Progress *ProgressConfig `yaml:"progress,omitempty"`
}

// ProgressConfig holds progress bar settings.
type ProgressConfig struct {
	Position  string `yaml:"position,omitempty"`
	Height    int    `yaml:"height,omitempty"`
	Color     string `yaml:"color,omitempty"`
	StartPage int    `yaml:"startPage,omitempty"`
	EndPage   int    `yaml:"endPage,omitempty"`
}

func (c *ProgressConfig) applyDefaults() {
	if c.Position == "" {
		c.Position = "bottom"
	}
	if c.Height <= 0 {
		c.Height = 10
	}
	if c.Color == "" {
		c.Color = "#4285F4"
	}
	if c.StartPage <= 0 {
		c.StartPage = 1
	}
	// EndPage == 0 means last slide (applied at runtime)
}

func (c *ProgressConfig) validate() error {
	if c.Position != "top" && c.Position != "bottom" {
		return fmt.Errorf("invalid position: %q (must be \"top\" or \"bottom\")", c.Position)
	}
	if c.Height <= 0 {
		return fmt.Errorf("height must be positive, got %d", c.Height)
	}
	if len(c.Color) != 7 || c.Color[0] != '#' {
		return fmt.Errorf("invalid color format: %q (must be \"#RRGGBB\")", c.Color)
	}
	if c.StartPage < 1 {
		return fmt.Errorf("startPage must be >= 1, got %d", c.StartPage)
	}
	if c.EndPage < 0 {
		return fmt.Errorf("endPage must be >= 0, got %d", c.EndPage)
	}
	if c.EndPage > 0 && c.EndPage < c.StartPage {
		return fmt.Errorf("endPage (%d) must be >= startPage (%d)", c.EndPage, c.StartPage)
	}
	return nil
}

// defaultConfigSearchPaths returns config file paths in priority order (highest first).
func defaultConfigSearchPaths() []string {
	paths := []string{
		"deck-pb.yml",
		"deck-pb.yaml",
	}
	if home, err := os.UserHomeDir(); err == nil {
		globalDir := filepath.Join(home, ".config", "deck-pb")
		paths = append(paths, filepath.Join(globalDir, "config.yml"), filepath.Join(globalDir, "config.yaml"))
	}
	return paths
}

// FindConfigFile returns the first existing config file from the search paths.
// Returns empty string if none found.
func FindConfigFile() string {
	for _, p := range defaultConfigSearchPaths() {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// DefaultConfig returns a Config with all default values applied.
func DefaultConfig() *Config {
	cfg := &Config{Progress: &ProgressConfig{}}
	cfg.Progress.applyDefaults()
	return cfg
}

// LoadConfig reads and parses a YAML configuration file.
// If path is empty, returns the default configuration.
func LoadConfig(path string) (*Config, error) {
	if path == "" {
		return DefaultConfig(), nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	cfg := &Config{}
	if err := yaml.Unmarshal(b, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}
	if cfg.Progress == nil {
		cfg.Progress = &ProgressConfig{}
	}
	cfg.Progress.applyDefaults()
	return cfg, nil
}
