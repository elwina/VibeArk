package httpclient

import (
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"time"
	"vibeark/internal/config"
)

// Client wraps http.Client with proxy support
type Client struct {
	HTTPClient *http.Client
	ProxyPort  string
}

// NewClient creates a new HTTP client with optional proxy and 30s timeout.
func NewClient(proxyPort string) *Client {
	return newClient(proxyPort, 30*time.Second)
}

// NewClientNoTimeout creates a new HTTP client for downloads (no body read timeout).
func NewClientNoTimeout(proxyPort string) *Client {
	return newClient(proxyPort, 0)
}

func newClient(proxyPort string, timeout time.Duration) *Client {
	transport := &http.Transport{
		Proxy: nil,
	}

	if proxyPort != "" {
		proxyURL := config.GetProxyURL(proxyPort)
		if proxyURL != "" {
			transport.Proxy = http.ProxyURL(&url.URL{
				Scheme: "http",
				Host:   fmt.Sprintf("127.0.0.1:%s", proxyPort),
			})
		}
	}

	return &Client{
		HTTPClient: &http.Client{
			Transport: transport,
			Timeout:   timeout,
			Jar:       mustCookieJar(),
		},
		ProxyPort: proxyPort,
	}
}

// Get performs a GET request and returns the response body as string
func (c *Client) Get(url string) (string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	// Don't set User-Agent - let Go use its default

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// GetWithRedirect follows redirects and returns the final URL and body
func (c *Client) GetWithRedirect(url string) (string, string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", "", err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}

	return resp.Request.URL.String(), string(body), nil
}

// Head performs a HEAD request and returns status and content length
func (c *Client) Head(url string) (int, int64, error) {
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return 0, 0, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()

	return resp.StatusCode, resp.ContentLength, nil
}

// ExtractVersion extracts version from HTML content using regex pattern
func ExtractVersion(content, pattern string) (string, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", fmt.Errorf("invalid regex pattern: %v", err)
	}

	matches := re.FindStringSubmatch(content)
	if len(matches) < 2 {
		return "", fmt.Errorf("version pattern not found")
	}

	return matches[1], nil
}

// ReplaceVersion replaces {version} placeholder in URL
func ReplaceVersion(urlTemplate, version string) string {
	return strings.ReplaceAll(urlTemplate, "{version}", version)
}

func mustCookieJar() http.CookieJar {
	jar, _ := cookiejar.New(nil)
	return jar
}
