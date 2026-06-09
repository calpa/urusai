package config

import (
	"testing"
)

func TestLoadDefaultConfig(t *testing.T) {
	cfg, err := LoadDefaultConfig()
	if err != nil {
		t.Fatalf("Failed to load default config: %v", err)
	}

	// Verify that the default config has valid values
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
}

func TestValidateValidConfig(t *testing.T) {
	cfg := &Config{
		MaxDepth:   10,
		MinSleep:   1,
		MaxSleep:   5,
		Timeout:    0,
		RootURLs:   []string{"https://example.com"},
		UserAgents: []string{"test-agent"},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected no error for valid config, got: %v", err)
	}
}

func TestValidateMaxDepthZero(t *testing.T) {
	cfg := &Config{
		MaxDepth:   0,
		MinSleep:   1,
		MaxSleep:   5,
		RootURLs:   []string{"https://example.com"},
		UserAgents: []string{"test-agent"},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for MaxDepth=0")
	}
}

func TestValidateMaxDepthNegative(t *testing.T) {
	cfg := &Config{
		MaxDepth:   -3,
		MinSleep:   1,
		MaxSleep:   5,
		RootURLs:   []string{"https://example.com"},
		UserAgents: []string{"test-agent"},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for negative MaxDepth")
	}
}

func TestValidateMinSleepZero(t *testing.T) {
	cfg := &Config{
		MaxDepth:   10,
		MinSleep:   0,
		MaxSleep:   5,
		RootURLs:   []string{"https://example.com"},
		UserAgents: []string{"test-agent"},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for MinSleep=0")
	}
}

func TestValidateMaxSleepZero(t *testing.T) {
	cfg := &Config{
		MaxDepth:   10,
		MinSleep:   1,
		MaxSleep:   0,
		RootURLs:   []string{"https://example.com"},
		UserAgents: []string{"test-agent"},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for MaxSleep=0")
	}
}

func TestValidateMinSleepGreaterThanMaxSleep(t *testing.T) {
	cfg := &Config{
		MaxDepth:   10,
		MinSleep:   10,
		MaxSleep:   3,
		RootURLs:   []string{"https://example.com"},
		UserAgents: []string{"test-agent"},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for MinSleep > MaxSleep")
	}
}

func TestValidateEmptyRootURLs(t *testing.T) {
	cfg := &Config{
		MaxDepth:   10,
		MinSleep:   1,
		MaxSleep:   5,
		RootURLs:   []string{},
		UserAgents: []string{"test-agent"},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for empty RootURLs")
	}
}

func TestValidateNilRootURLs(t *testing.T) {
	cfg := &Config{
		MaxDepth:   10,
		MinSleep:   1,
		MaxSleep:   5,
		RootURLs:   nil,
		UserAgents: []string{"test-agent"},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for nil RootURLs")
	}
}

func TestValidateEmptyUserAgents(t *testing.T) {
	cfg := &Config{
		MaxDepth:   10,
		MinSleep:   1,
		MaxSleep:   5,
		RootURLs:   []string{"https://example.com"},
		UserAgents: []string{},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for empty UserAgents")
	}
}

func TestValidateNilUserAgents(t *testing.T) {
	cfg := &Config{
		MaxDepth:   10,
		MinSleep:   1,
		MaxSleep:   5,
		RootURLs:   []string{"https://example.com"},
		UserAgents: nil,
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for nil UserAgents")
	}
}

func TestValidateDefaultConfig(t *testing.T) {
	cfg, err := LoadDefaultConfig()
	if err != nil {
		t.Fatalf("Failed to load default config: %v", err)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("default config should be valid, got: %v", err)
	}
}
