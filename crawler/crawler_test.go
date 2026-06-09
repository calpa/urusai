package crawler

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/calpa/urusai/config"
)

func testConfig() *config.Config {
	cfg := &config.Config{
		RootURLs:        []string{"http://example.com"},
		BlacklistedURLs: []string{},
		UserAgents:      []string{"test-bot"},
		MinSleep:        100,
		MaxSleep:        200,
		MaxDepth:        3,
		Timeout:         60,
	}
	cfg.Blacklist = make(map[string]struct{})
	return cfg
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
		{"http://example.com:8080", false},
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
	cfg.Blacklist = map[string]struct{}{
		"/login": {},
		"/admin": {},
	}
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

func TestIsPrivateIP(t *testing.T) {
	cases := []struct {
		ip    net.IP
		want  bool
		label string
	}{
		{net.ParseIP("127.0.0.1"), true, "IPv4 loopback"},
		{net.ParseIP("127.0.0.2"), true, "IPv4 loopback alt"},
		{net.ParseIP("::1"), true, "IPv6 loopback"},
		{net.ParseIP("10.0.0.1"), true, "10.0.0.0/8"},
		{net.ParseIP("172.16.0.1"), true, "172.16.0.0/12"},
		{net.ParseIP("192.168.1.1"), true, "192.168.0.0/16"},
		{net.ParseIP("169.254.169.254"), true, "AWS metadata"},
		{net.ParseIP("169.254.0.1"), true, "link-local IPv4"},
		{net.ParseIP("fe80::1"), true, "IPv6 link-local"},
		{net.ParseIP("fc00::1"), true, "IPv6 ULA fc00"},
		{net.ParseIP("fd00::1"), true, "IPv6 ULA fd00"},
		{net.ParseIP("8.8.8.8"), false, "Google DNS"},
		{net.ParseIP("1.1.1.1"), false, "Cloudflare DNS"},
		{net.ParseIP("203.0.113.1"), false, "public IPv4"},
		{net.ParseIP("2606:4700:4700::1111"), false, "public IPv6"},
	}
	for _, tc := range cases {
		t.Run(tc.label, func(t *testing.T) {
			got := isPrivateIP(tc.ip)
			if got != tc.want {
				t.Errorf("isPrivateIP(%v) = %v, want %v", tc.ip, got, tc.want)
			}
		})
	}
}

func TestIsPrivateURL_LiteralIPs(t *testing.T) {
	cases := []struct {
		url   string
		want  bool
		label string
	}{
		{"http://127.0.0.1", true, "loopback"},
		{"http://10.0.0.1/path", true, "private 10.x"},
		{"http://172.16.0.1", true, "private 172.16.x"},
		{"http://192.168.1.1", true, "private 192.168.x"},
		{"http://169.254.169.254/latest", true, "AWS metadata"},
		{"http://[::1]", true, "IPv6 loopback"},
		{"http://[fc00::1]", true, "IPv6 ULA"},
		{"http://[fe80::1]", true, "IPv6 link-local"},
		{"http://8.8.8.8", false, "public IP"},
		{"http://1.1.1.1:8080", false, "public IP with port"},
		{"http://[2606:4700::1111]", false, "public IPv6"},
	}
	for _, tc := range cases {
		t.Run(tc.label, func(t *testing.T) {
			got := isPrivateURL(tc.url)
			if got != tc.want {
				t.Errorf("isPrivateURL(%q) = %v, want %v", tc.url, got, tc.want)
			}
		})
	}
}

func TestIsPrivateURL_Unparseable(t *testing.T) {
	if !isPrivateURL("://not-a-url") {
		t.Error("unparseable URL should be treated as private")
	}
}

func TestIsPrivateIP_Nil(t *testing.T) {
	if isPrivateIP(nil) {
		t.Error("nil IP should not be private")
	}
}
