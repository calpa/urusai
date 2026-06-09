package config

import (
	"embed"
	"encoding/json"
	"os"
)

//go:embed default_config.json
var defaultConfigJSON embed.FS

// Config represents the configuration for the crawler
type Config struct {
	MaxDepth        int      `json:"max_depth"`
	MinSleep        int      `json:"min_sleep"`
	MaxSleep        int      `json:"max_sleep"`
	Timeout         int      `json:"timeout"`
	Concurrency     int      `json:"concurrency"`
	RootURLs        []string `json:"root_urls"`
	BlacklistedURLs []string `json:"blacklisted_urls"`
	UserAgents      []string `json:"user_agents"`
}

// Validate checks configuration values and applies defaults.
func (c *Config) Validate() {
	if c.Concurrency <= 0 {
		c.Concurrency = 1
	}
	if c.Timeout < 0 {
		c.Timeout = 0
	}
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

	config.Validate()
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

	config.Validate()
	return config, nil
}
