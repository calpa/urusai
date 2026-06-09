package crawler

import (
	"context"
	"fmt"
	"io"
	"log"
	"math/rand/v2"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/calpa/urusai/config"
)

var (
	validURLRegex = regexp.MustCompile(
		`(?i)^(?:http|https)s?://`+
			`(?:(?:[A-Z0-9](?:[A-Z0-9-]{0,61}[A-Z0-9])?\.)+(?:[A-Z]{2,6}\.?|[A-Z0-9-]{2,}\.?)|`+
			`localhost|`+
			`\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})`+
			`(?:\d+)?`+
			`(?:/?|[/?]\S+)$`)
	hrefRegex = regexp.MustCompile(`href=["'](.*?)["']`)
)

// Crawler represents a web crawler that generates random HTTP traffic
type Crawler struct {
	config     *config.Config
	mu         sync.Mutex
	links      []string
	startTime  time.Time
	httpClient *http.Client
}

// NewCrawler creates a new crawler with the given configuration
func NewCrawler(cfg *config.Config) *Crawler {
	return &Crawler{
		config: cfg,
		links:  []string{},
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 5 {
					return fmt.Errorf("too many redirects")
				}
				if isPrivateURL(req.URL.String()) {
					return fmt.Errorf("redirect to private address blocked")
				}
				return nil
			},
		},
	}
}

// Crawl starts the crawling process with concurrent workers.
func (c *Crawler) Crawl(ctx context.Context) {
	c.startTime = time.Now()

	var wg sync.WaitGroup
	for i := 0; i < c.config.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.worker(ctx)
		}()
	}
	wg.Wait()
}

// worker is the main loop for a single concurrent crawler.
func (c *Crawler) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if c.isTimeoutReached() {
			return
		}

		c.mu.Lock()
		rootURL := c.config.RootURLs[rand.N(len(c.config.RootURLs))]
		c.mu.Unlock()

		log.Printf("Starting with root URL: %s", rootURL)

		body, err := c.request(ctx, rootURL)
		if err != nil {
			log.Printf("Error connecting to root URL %s: %v", rootURL, err)
			continue
		}

		c.mu.Lock()
		c.links = c.extractURLs(body, rootURL)
		linkCount := len(c.links)
		c.mu.Unlock()
		log.Printf("Found %d links from %s", linkCount, rootURL)

		if linkCount > 0 {
			c.browseFromLinks(ctx, c.config.MaxDepth)
		}
	}
}

// request sends an HTTP request to the given URL with a random user agent
func (c *Crawler) request(ctx context.Context, urlStr string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return "", err
	}

	userAgent := c.config.UserAgents[rand.N(len(c.config.UserAgents))]
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Read the response body, capped at 1MB
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1*1024*1024))
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// normalizeLink converts relative URLs to absolute URLs
func (c *Crawler) normalizeLink(link, rootURL string) string {
	if strings.HasPrefix(link, "//") {
		parsedRoot, err := url.Parse(rootURL)
		if err != nil {
			return ""
		}
		return parsedRoot.Scheme + ":" + link
	}

	parsedURL, err := url.Parse(link)
	if err != nil {
		return ""
	}

	if parsedURL.Scheme != "" {
		return link
	}

	base, err := url.Parse(rootURL)
	if err != nil {
		return ""
	}

	return base.ResolveReference(parsedURL).String()
}

// isValidURL checks if a URL is valid
func (c *Crawler) isValidURL(urlStr string) bool {
	return validURLRegex.MatchString(urlStr)
}

// isPrivateURL checks if a URL resolves to a private, loopback, link-local, or metadata address.
func isPrivateURL(urlStr string) bool {
	u, err := url.Parse(urlStr)
	if err != nil {
		return true
	}
	host := u.Hostname()
	ip := net.ParseIP(host)
	if ip != nil {
		return isPrivateIP(ip)
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		return true
	}
	for _, resolved := range ips {
		if isPrivateIP(resolved) {
			return true
		}
	}
	return false
}

// isPrivateIP reports whether an IP falls into any blocked range.
func isPrivateIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() {
		return true
	}
	if ipv4 := ip.To4(); ipv4 != nil {
		if ipv4[0] == 169 && ipv4[1] == 254 {
			return true
		}
	}
	if ip.To4() == nil && len(ip) >= 1 && (ip[0]&0xfe) == 0xfc {
		return true
	}
	return false
}

// isBlacklisted checks if a URL is blacklisted using the map for O(1) lookups.
func (c *Crawler) isBlacklisted(urlStr string) bool {
	for suffix := range c.config.Blacklist {
		if strings.Contains(urlStr, suffix) {
			return true
		}
	}
	return false
}

// shouldAcceptURL checks if a URL should be accepted for crawling
func (c *Crawler) shouldAcceptURL(urlStr string) bool {
	return urlStr != "" && c.isValidURL(urlStr) && !c.isBlacklisted(urlStr) && !isPrivateURL(urlStr)
}

// extractURLs extracts URLs from an HTML body
func (c *Crawler) extractURLs(body, rootURL string) []string {
	matches := hrefRegex.FindAllStringSubmatch(body, -1)

	var urls []string
	for _, match := range matches {
		if len(match) > 1 {
			if strings.HasPrefix(match[1], "#") {
				continue
			}

			normalizedURL := c.normalizeLink(match[1], rootURL)

			if c.shouldAcceptURL(normalizedURL) {
				urls = append(urls, normalizedURL)
			}
		}
	}

	return urls
}

// removeAndBlacklist removes a link and adds to blacklist. Caller must hold c.mu.
func (c *Crawler) removeAndBlacklist(link string) {
	c.config.Blacklist[link] = struct{}{}
	for i, l := range c.links {
		if l == link {
			c.links = append(c.links[:i], c.links[i+1:]...)
			break
		}
	}
}
// browseFromLinks browses from the available links iteratively.
func (c *Crawler) browseFromLinks(ctx context.Context, maxDepth int) {
	for depth := 0; depth < maxDepth; depth++ {
		select {
		case <-ctx.Done():
			return
		default:
		}

		c.mu.Lock()
		linkCount := len(c.links)
		c.mu.Unlock()
		if linkCount == 0 {
			return
		}

		if c.isTimeoutReached() {
			return
		}

		c.mu.Lock()
		randomIndex := rand.N(len(c.links))
		randomLink := c.links[randomIndex]
		c.mu.Unlock()

		log.Printf("Visiting %s (depth: %d)", randomLink, depth)

		body, err := c.request(ctx, randomLink)
		if err != nil {
			log.Printf("Error visiting %s: %v", randomLink, err)
			c.mu.Lock()
			c.removeAndBlacklist(randomLink)
			c.mu.Unlock()
			continue
		}

		subLinks := c.extractURLs(body, randomLink)
		log.Printf("Found %d links from %s", len(subLinks), randomLink)

		sleepTime := time.Duration(rand.IntN(c.config.MaxSleep-c.config.MinSleep+1)+c.config.MinSleep) * time.Second
		log.Printf("Sleeping for %v", sleepTime)
		select {
		case <-time.After(sleepTime):
		case <-ctx.Done():
			return
		}

		c.mu.Lock()
		if len(subLinks) > 1 {
			c.links = subLinks
		} else {
			c.removeAndBlacklist(randomLink)
		}
		c.mu.Unlock()
	}

}
// isTimeoutReached checks if the timeout has been reached
func (c *Crawler) isTimeoutReached() bool {
	if c.config.Timeout == 0 {
		return false
	}

	timeoutDuration := time.Duration(c.config.Timeout) * time.Second
	return time.Since(c.startTime) > timeoutDuration
}
