package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	KerberosMock bool      `yaml:"kerberos_mock"`
	ServersMock  bool      `yaml:"servers_mock"`
	API          APIConfig `yaml:"api"`
	SSH          SSHConfig `yaml:"ssh"`
	CacheDir     string    `yaml:"cache_dir"`
}

type APIConfig struct {
	BaseURL  string   `yaml:"base_url"`
	ESMCodes []string `yaml:"esm_codes"`
}

type SSHConfig struct {
	DefaultUser string `yaml:"default_user"`
}

func DefaultConfig() *Config {
	return &Config{
		API: APIConfig{
			BaseURL: "https://baseurl",
		},
		SSH: SSHConfig{
			DefaultUser: "irteam",
		},
	}
}

func DefaultConfigPath() string {
	dir, err := configDir()
	if err != nil {
		return "~/.karazhan/config.yaml"
	}
	return filepath.Join(dir, "config.yaml")
}

func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".karazhan"), nil
}

// Load loads config from the default path.
func Load() (*Config, error) {
	return LoadFrom(DefaultConfigPath())
}

// LoadFrom loads config from a specific path.
func LoadFrom(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, err
	}

	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", path, err)
	}

	if cfg.CacheDir == "" {
		dir := filepath.Dir(path)
		cfg.CacheDir = filepath.Join(dir, "cache")
	}

	return cfg, nil
}

func (c *Config) Save() error {
	dir, err := configDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(dir, "config.yaml"), data, 0600)
}

func (c *Config) ServerCachePath() string {
	if c.CacheDir == "" {
		dir, _ := configDir()
		c.CacheDir = filepath.Join(dir, "cache")
	}
	return filepath.Join(c.CacheDir, "servers.json")
}

// Dump returns the current config as YAML string.
func (c *Config) Dump() (string, error) {
	data, err := yaml.Marshal(c)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

const SampleConfig = `# Karazhan Configuration
# =====================

# Kerberos authentication mock (skip kinit for development)
kerberos_mock: false

# Server list mock (use built-in mock data instead of API)
servers_mock: false

# TIPS API configuration
api:
  # Base URL for TIPS server API
  base_url: "https://karazhan.com/api/external"

  # ESM codes to fetch server lists from (supports multiple)
  esm_codes:
    - "NE032765"
    # - "NE026503"
    # - "NE026504"

# SSH connection settings
ssh:
  # Default SSH user (irteam or irteamsu, selectable at connection time)
  default_user: "irteam"

# Cache directory for server list (default: ~/.karazhan/cache)
# cache_dir: ""
`
