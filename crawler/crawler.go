package crawler

import (
	"context"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"regexp"
	"strings"
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

// Crawl starts the crawling process
func (c *Crawler) Crawl(ctx context.Context) {
	c.startTime = time.Now()

	for {
		select {
		case <-ctx.Done():
			log.Println("Shutdown signal received")
			return
		default:
		}

		if c.isTimeoutReached() {
			log.Println("Timeout has been reached, exiting")
			return
		}

		// Select a random root URL
		rootURL := c.config.RootURLs[rand.Intn(len(c.config.RootURLs))]
		log.Printf("Starting with root URL: %s", rootURL)

		try := func() bool {
			body, err := c.request(ctx, rootURL)
			if err != nil {
				log.Printf("Error connecting to root URL %s: %v", rootURL, err)
				return false
			}

			c.links = c.extractURLs(body, rootURL)
			log.Printf("Found %d links from %s", len(c.links), rootURL)

			if len(c.links) > 0 {
				c.browseFromLinks(ctx, c.config.MaxDepth)
				return true
			}
			return false
		}

		// Try to crawl, if failed, try another root URL
		if !try() {
			continue
		}
	}
}

// request sends an HTTP request to the given URL with a random user agent
func (c *Crawler) request(ctx context.Context, urlStr string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return "", err
	}

	// Set a random user agent
	userAgent := c.config.UserAgents[rand.Intn(len(c.config.UserAgents))]
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
	// Handle URLs that start with //
	if strings.HasPrefix(link, "//") {
		parsedRoot, err := url.Parse(rootURL)
		if err != nil {
			return ""
		}
		return parsedRoot.Scheme + ":" + link
	}

	// Parse the URL
	parsedURL, err := url.Parse(link)
	if err != nil {
		return ""
	}

	// If it's already an absolute URL, return it
	if parsedURL.Scheme != "" {
		return link
	}

	// Join the root URL with the relative URL
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

// isBlacklisted checks if a URL is blacklisted
func (c *Crawler) isBlacklisted(urlStr string) bool {
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
	matches := hrefRegex.FindAllStringSubmatch(body, -1)

	var urls []string
	for _, match := range matches {
		if len(match) > 1 {
			// Ignore fragment links (links starting with #)
			if strings.HasPrefix(match[1], "#") {
				continue
			}

			// Normalize the link
			normalizedURL := c.normalizeLink(match[1], rootURL)

			// Check if the URL should be accepted
			if c.shouldAcceptURL(normalizedURL) {
				urls = append(urls, normalizedURL)
			}
		}
	}

	return urls
}

// removeAndBlacklist removes a link from the links list and adds it to the blacklist
func (c *Crawler) removeAndBlacklist(link string) {
	// Add to blacklist
	c.config.BlacklistedURLs = append(c.config.BlacklistedURLs, link)

	// Remove from links
	for i, l := range c.links {
		if l == link {
			c.links = append(c.links[:i], c.links[i+1:]...)
			break
		}
	}
}

// browseFromLinks browses from the available links iteratively
func (c *Crawler) browseFromLinks(ctx context.Context, maxDepth int) {
	for depth := 0; depth < maxDepth; depth++ {
		// Check context cancellation
		select {
		case <-ctx.Done():
			log.Println("Shutdown signal received")
			return
		default:
		}

		// Check if we have any links to browse
		if len(c.links) == 0 {
			log.Println("No links to browse, moving to next root URL")
			return
		}

		// Check if timeout has been reached
		if c.isTimeoutReached() {
			log.Println("Timeout has been reached, exiting")
			return
		}

		// Select a random link
		randomIndex := rand.Intn(len(c.links))
		randomLink := c.links[randomIndex]

		log.Printf("Visiting %s (depth: %d)", randomLink, depth)

		// Visit the link
		body, err := c.request(ctx, randomLink)
		if err != nil {
			log.Printf("Error visiting %s: %v", randomLink, err)
			c.removeAndBlacklist(randomLink)
			continue
		}

		// Extract links from the page
		subLinks := c.extractURLs(body, randomLink)
		log.Printf("Found %d links from %s", len(subLinks), randomLink)

		// Sleep for a random amount of time
		sleepTime := time.Duration(rand.Intn(c.config.MaxSleep-c.config.MinSleep+1)+c.config.MinSleep) * time.Second
		log.Printf("Sleeping for %v", sleepTime)
		time.Sleep(sleepTime)

		// If we found more than one link, update our links list
		if len(subLinks) > 1 {
			c.links = subLinks
		} else {
			// Otherwise, remove the current link from our list
			c.removeAndBlacklist(randomLink)
		}
	}
}

// isTimeoutReached checks if the timeout has been reached
func (c *Crawler) isTimeoutReached() bool {
	// If timeout is 0, it means no timeout
	if c.config.Timeout == 0 {
		return false
	}

	// Check if the current time exceeds the start time plus the timeout
	timeoutDuration := time.Duration(c.config.Timeout) * time.Second
	return time.Since(c.startTime) > timeoutDuration
}
