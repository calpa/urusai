package crawler

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/calpa/urusai/config"
)

func testConfig() *config.Config {
	return &config.Config{
		RootURLs:        []string{"http://example.com"},
		BlacklistedURLs: []string{},
		UserAgents:      []string{"test-bot"},
		MinSleep:        100,
		MaxSleep:        200,
		MaxDepth:        3,
		Timeout:         60,
	}
}

func TestRequestBodyCompleteness(t *testing.T) {
	wantBody := strings.Repeat("A", 2048)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, wantBody)
	}))
	defer ts.Close()

	c := NewCrawler(testConfig())
	got, err := c.request(context.Background(), ts.URL)
	if err != nil {
		t.Fatalf("request(%q) returned error: %v", ts.URL, err)
	}
	if got != wantBody {
		t.Errorf("body truncated: got %d bytes, want %d bytes", len(got), len(wantBody))
	}
}

func TestRequestLargeBodyCapped(t *testing.T) {
	twoMB := strings.Repeat("X", 2*1024*1024)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, twoMB)
	}))
	defer ts.Close()

	c := NewCrawler(testConfig())
	got, err := c.request(context.Background(), ts.URL)
	if err != nil {
		t.Fatalf("request returned error: %v", err)
	}
	if len(got) > 1*1024*1024 {
		t.Errorf("body exceeds 1MB cap: got %d bytes", len(got))
	}
	if len(got) != 1*1024*1024 {
		t.Errorf("expected exactly 1MB, got %d bytes", len(got))
	}
}

func TestIsValidURL(t *testing.T) {
	tests := []struct {
		url  string
		want bool
	}{
		{"http://example.com", true},
		{"https://example.com", true},
		{"http://example.com/path", true},
		{"http://localhost", true},
		{"http://127.0.0.1", true},
		{"http://example.com:8080", false}, // pre-existing regex doesn't support ports
		{"ftp://example.com", false},
		{"example.com", false},
		{"", false},
		{"javascript:alert(1)", false},
	}
	c := NewCrawler(testConfig())
	for _, tt := range tests {
		got := c.isValidURL(tt.url)
		if got != tt.want {
			t.Errorf("isValidURL(%q) = %v, want %v", tt.url, got, tt.want)
		}
	}
}

func TestExtractURLs(t *testing.T) {
	c := NewCrawler(testConfig())

	body := `<a href="http://example.com/page1">link1</a>` +
		`<a href="http://example.com/page2">link2</a>` +
		`<a href="#fragment">fragment</a>` +
		`<a href='http://example.com/page3'>link3</a>`

	urls := c.extractURLs(body, "http://example.com")
	if len(urls) != 3 {
		t.Fatalf("extractURLs returned %d urls, want 3: %v", len(urls), urls)
	}

	expected := []string{
		"http://example.com/page1",
		"http://example.com/page2",
		"http://example.com/page3",
	}
	for i, exp := range expected {
		if urls[i] != exp {
			t.Errorf("urls[%d] = %q, want %q", i, urls[i], exp)
		}
	}
}

func TestExtractURLsRelativeResolution(t *testing.T) {
	c := NewCrawler(testConfig())
	body := `<a href="/about">about</a>`
	urls := c.extractURLs(body, "http://example.com")
	if len(urls) != 1 || urls[0] != "http://example.com/about" {
		t.Errorf("relative URL not resolved: got %v", urls)
	}
}

func TestIsBlacklisted(t *testing.T) {
	cfg := testConfig()
	cfg.BlacklistedURLs = []string{"/login", "/admin"}
	c := NewCrawler(cfg)

	tests := []struct {
		url  string
		want bool
	}{
		{"http://example.com/login", true},
		{"http://example.com/admin/page", true},
		{"http://example.com/home", false},
		{"http://other.com/login", true},
	}
	for _, tt := range tests {
		got := c.isBlacklisted(tt.url)
		if got != tt.want {
			t.Errorf("isBlacklisted(%q) = %v, want %v", tt.url, got, tt.want)
		}
	}
}
