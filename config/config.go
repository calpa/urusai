package config

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
)

//go:embed default_config.json
var defaultConfigJSON embed.FS

// Config represents the configuration for the crawler
type Config struct {
	MaxDepth        int                 `json:"max_depth"`
	MinSleep        int                 `json:"min_sleep"`
	MaxSleep        int                 `json:"max_sleep"`
	Timeout         int                 `json:"timeout"`
	RootURLs        []string            `json:"root_urls"`
	BlacklistedURLs []string            `json:"blacklisted_urls"`
	Blacklist       map[string]struct{} `json:"-"`
	UserAgents      []string            `json:"user_agents"`
}

// initBlacklist populates the Blacklist map from the BlacklistedURLs slice.
func (c *Config) initBlacklist() {
	c.Blacklist = make(map[string]struct{}, len(c.BlacklistedURLs))
	for _, u := range c.BlacklistedURLs {
		c.Blacklist[u] = struct{}{}
	}
}

// Validate checks that all required config fields have valid values.
func (c *Config) Validate() error {
	if c.MaxDepth <= 0 {
		return fmt.Errorf("config: max_depth must be > 0, got %d", c.MaxDepth)
	}
	if c.MinSleep <= 0 {
		return fmt.Errorf("config: min_sleep must be > 0, got %d", c.MinSleep)
	}
	if c.MaxSleep <= 0 {
		return fmt.Errorf("config: max_sleep must be > 0, got %d", c.MaxSleep)
	}
	if c.MinSleep > c.MaxSleep {
		return fmt.Errorf("config: min_sleep (%d) must be <= max_sleep (%d)", c.MinSleep, c.MaxSleep)
	}
	if len(c.RootURLs) == 0 {
		return fmt.Errorf("config: root_urls must not be empty")
	}
	if len(c.UserAgents) == 0 {
		return fmt.Errorf("config: user_agents must not be empty")
	}
	return nil
}

// LoadFromFile loads configuration from a JSON file
func LoadFromFile(filePath string) (*Config, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	config := &Config{}
	err = decoder.Decode(config)
	if err != nil {
		return nil, err
	}

	if config.Timeout < 0 {
		config.Timeout = 0
	}

	config.initBlacklist()
	return config, nil
}

// LoadDefaultConfig loads the default configuration embedded in the binary
func LoadDefaultConfig() (*Config, error) {
	data, err := defaultConfigJSON.ReadFile("default_config.json")
	if err != nil {
		return nil, err
	}

	config := &Config{}
	err = json.Unmarshal(data, config)
	if err != nil {
		return nil, err
	}

	config.initBlacklist()
	return config, nil
}
