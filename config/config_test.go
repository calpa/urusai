package config

import (
	"testing"
)

func TestLoadDefaultConfig(t *testing.T) {
	cfg, err := LoadDefaultConfig()
	if err != nil {
		t.Fatalf("Failed to load default config: %v", err)
	}

	if cfg.MaxDepth <= 0 {
		t.Errorf("Expected MaxDepth > 0, got %d", cfg.MaxDepth)
	}

	if cfg.MinSleep <= 0 {
		t.Errorf("Expected MinSleep > 0, got %d", cfg.MinSleep)
	}

	if cfg.MaxSleep <= 0 {
		t.Errorf("Expected MaxSleep > 0, got %d", cfg.MaxSleep)
	}

	if len(cfg.RootURLs) == 0 {
		t.Error("Expected RootURLs to have at least one URL")
	}

	if len(cfg.UserAgents) == 0 {
		t.Error("Expected UserAgents to have at least one user agent")
	}

	if cfg.Concurrency != 3 {
		t.Errorf("Expected default Concurrency=3, got %d", cfg.Concurrency)
	}
}

func TestValidateConcurrency(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected int
	}{
		{"zero defaults to 1", 0, 1},
		{"negative defaults to 1", -5, 1},
		{"positive preserved", 4, 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{Concurrency: tt.input}
			c.Validate()
			if c.Concurrency != tt.expected {
				t.Errorf("Validate(): Concurrency=%d, want %d", c.Concurrency, tt.expected)
			}
		})
	}
}

func TestValidateTimeout(t *testing.T) {
	c := &Config{Timeout: -1, Concurrency: 2}
	c.Validate()
	if c.Timeout != 0 {
		t.Errorf("Validate(): Timeout=%d, want 0", c.Timeout)
	}
	if c.Concurrency != 2 {
		t.Errorf("Validate(): Concurrency=%d, want 2", c.Concurrency)
	}
}
