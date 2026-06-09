package crawler

import (
	"context"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/calpa/urusai/config"
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

// worker independently picks random root URLs and follows links until ctx is cancelled or timeout is reached.
func (c *Crawler) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if c.isTimeoutReached() {
			log.Println("Timeout has been reached, exiting")
			return
		}

		c.mu.Lock()
		rootURL := c.config.RootURLs[rand.Intn(len(c.config.RootURLs))]
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
			c.browseFromLinks(ctx, 0)
		}
	}
}

// request sends an HTTP request to the given URL with a random user agent.
func (c *Crawler) request(ctx context.Context, urlStr string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return "", err
	}

	c.mu.Lock()
	userAgent := c.config.UserAgents[rand.Intn(len(c.config.UserAgents))]
	c.mu.Unlock()
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	buf := make([]byte, 1024*1024)
	n, _ := resp.Body.Read(buf)
	return string(buf[:n]), nil
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
	regex := regexp.MustCompile(
		`(?i)^(?:http|https)s?://` +
			`(?:(?:[A-Z0-9](?:[A-Z0-9-]{0,61}[A-Z0-9])?\.)+(?:[A-Z]{2,6}\.?|[A-Z0-9-]{2,}\.?)|` +
			`localhost|` +
			`\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})` +
			`(?:\d+)?` +
			`(?:/?|[/?]\S+)$`)
	return regex.MatchString(urlStr)
}

// isBlacklisted checks if a URL is blacklisted
func (c *Crawler) isBlacklisted(urlStr string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, blacklisted := range c.config.BlacklistedURLs {
		if strings.Contains(urlStr, blacklisted) {
			return true
		}
	}
	return false
}

// shouldAcceptURL checks if a URL should be accepted for crawling
func (c *Crawler) shouldAcceptURL(urlStr string) bool {
	return urlStr != "" && c.isValidURL(urlStr) && !c.isBlacklisted(urlStr)
}

// extractURLs extracts URLs from an HTML body
func (c *Crawler) extractURLs(body, rootURL string) []string {
	pattern := `href=["'](.*?)["']`
	regex := regexp.MustCompile(pattern)
	matches := regex.FindAllStringSubmatch(body, -1)

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

// removeAndBlacklist removes a link from the links list and adds it to the blacklist
func (c *Crawler) removeAndBlacklist(link string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.config.BlacklistedURLs = append(c.config.BlacklistedURLs, link)

	for i, l := range c.links {
		if l == link {
			c.links = append(c.links[:i], c.links[i+1:]...)
			break
		}
	}
}

// browseFromLinks browses from the available links iteratively.
func (c *Crawler) browseFromLinks(ctx context.Context, depth int) {
	for depth < c.config.MaxDepth {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if c.isTimeoutReached() {
			log.Println("Timeout has been reached, exiting")
			return
		}

		c.mu.Lock()
		if len(c.links) == 0 {
			c.mu.Unlock()
			log.Println("No links to browse, moving to next root URL")
			return
		}

		randomIndex := rand.Intn(len(c.links))
		randomLink := c.links[randomIndex]
		c.mu.Unlock()

		log.Printf("Visiting %s (depth: %d)", randomLink, depth)

		body, err := c.request(ctx, randomLink)
		if err != nil {
			log.Printf("Error visiting %s: %v", randomLink, err)
			c.removeAndBlacklist(randomLink)
			continue
		}

		subLinks := c.extractURLs(body, randomLink)
		log.Printf("Found %d links from %s", len(subLinks), randomLink)

		c.mu.Lock()
		sleepTime := time.Duration(rand.Intn(c.config.MaxSleep-c.config.MinSleep+1)+c.config.MinSleep) * time.Second
		c.mu.Unlock()
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

		depth++
	}
	log.Println("Maximum depth reached, moving to next root URL")
}

// isTimeoutReached checks if the timeout has been reached
func (c *Crawler) isTimeoutReached() bool {
	if c.config.Timeout == 0 {
		return false
	}

	timeoutDuration := time.Duration(c.config.Timeout) * time.Second
	return time.Since(c.startTime) > timeoutDuration
}
