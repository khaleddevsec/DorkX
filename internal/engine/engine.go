package engine

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// Result is a single scraped candidate URL before filtering
type Result struct {
	RawURL string
	Source string // which engine returned it
}

// Engine defines what any search engine scraper must implement
type Engine interface {
	Name() string
	Search(dork string, page int, proxyURL *url.URL, timeout int) ([]Result, error)
}

// ---- User-Agent Pool --------------------------------------------------------
var userAgents = []string{
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",
	"Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:125.0) Gecko/20100101 Firefox/125.0",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
	"Mozilla/5.0 (X11; Fedora; Linux x86_64; rv:124.0) Gecko/20100101 Firefox/124.0",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",
	"Mozilla/5.0 (X11; Linux x86_64) rv:123.0 Gecko/20100101 Firefox/123.0",
}

func randomUA() string {
	return userAgents[rand.Intn(len(userAgents))]
}

// ---- HTTP Client Builder ----------------------------------------------------

func buildClient(proxyURL *url.URL, timeout int) *http.Client {
	transport := &http.Transport{
		DisableKeepAlives: false,
		MaxIdleConns:      10,
	}

	if proxyURL != nil {
		transport.Proxy = http.ProxyURL(proxyURL)
	}

	effectiveTimeout := timeout
	if effectiveTimeout <= 0 {
		effectiveTimeout = 30
	}

	return &http.Client{
		Transport: transport,
		Timeout:   time.Duration(effectiveTimeout) * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func buildHeaders(referer string) map[string]string {
	return map[string]string{
		"User-Agent":                randomUA(),
		"Accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8",
		"Accept-Language":           "en-US,en;q=0.9",
		"Connection":                "keep-alive",
		"Upgrade-Insecure-Requests": "1",
		"Sec-Fetch-Dest":            "document",
		"Sec-Fetch-Mode":            "navigate",
		"Sec-Fetch-Site":            "none",
		"Sec-Fetch-User":            "?1",
		"Cache-Control":             "max-age=0",
		"Referer":                   referer,
	}
}

func fetchPage(client *http.Client, targetURL string, headers map[string]string) (string, error) {
	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return "", err
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 || resp.StatusCode == 503 {
		return "", fmt.Errorf("rate limited (HTTP %d)", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// ---- Google HTML Engine -----------------------------------------------------

type GoogleEngine struct{}

func NewGoogleEngine() *GoogleEngine {
	return &GoogleEngine{}
}

func (g *GoogleEngine) Name() string { return "Google (HTML Scraper)" }

var hrefRe = regexp.MustCompile(`href=["'](/url\?q=[^"'&]+|https?://[^"'\s>]+)["' >]`)

func (g *GoogleEngine) Search(dork string, page int, proxyURL *url.URL, timeout int) ([]Result, error) {
	start := page * 10
	encoded := url.QueryEscape(dork)

	searchURL := fmt.Sprintf(
		"https://www.google.com/search?q=%s&hl=en&start=%d&filter=0",
		encoded, start,
	)

	client := buildClient(proxyURL, timeout)
	headers := buildHeaders("https://www.google.com/")

	body, err := fetchPage(client, searchURL, headers)
	if err != nil {
		return nil, err
	}

	lowerBody := strings.ToLower(body)
	if strings.Contains(lowerBody, "detected unusual traffic") ||
		strings.Contains(lowerBody, "recaptcha") ||
		strings.Contains(lowerBody, "sorry/index") {
		return nil, fmt.Errorf("CAPTCHA or rate-limit detected on page %d", page+1)
	}

	var results []Result
	seen := make(map[string]bool)

	matches := hrefRe.FindAllStringSubmatch(body, -1)
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		raw := match[1]

		if strings.HasPrefix(raw, "/url?q=") {
			raw = "/url?q=" + raw[7:]
		}

		if !strings.HasPrefix(raw, "http") && !strings.HasPrefix(raw, "/url") {
			continue
		}

		if !seen[raw] {
			seen[raw] = true
			results = append(results, Result{RawURL: raw, Source: g.Name()})
		}
	}

	return results, nil
}

// ---- Google Custom Search (CSE) API Engine -----------------------------------

type GoogleCSEEngine struct {
	APIKey string
	CX     string
}

func NewGoogleCSEEngine(apiKey, cx string) *GoogleCSEEngine {
	return &GoogleCSEEngine{
		APIKey: apiKey,
		CX:     cx,
	}
}

func (g *GoogleCSEEngine) Name() string { return "Google (Official Custom Search API)" }

type googleCSEJSONResponse struct {
	Items []struct {
		Link  string `json:"link"`
		Title string `json:"title"`
	} `json:"items"`
	Error *struct {
		Message string `json:"message"`
		Code    int    `json:"code"`
	} `json:"error"`
}

func (g *GoogleCSEEngine) Search(dork string, page int, proxyURL *url.URL, timeout int) ([]Result, error) {
	// Google CSE uses 1-based start index (1, 11, 21...)
	start := (page * 10) + 1
	apiURL := fmt.Sprintf(
		"https://www.googleapis.com/customsearch/v1?key=%s&cx=%s&q=%s&start=%d",
		url.QueryEscape(g.APIKey),
		url.QueryEscape(g.CX),
		url.QueryEscape(dork),
		start,
	)

	client := buildClient(proxyURL, timeout)
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var jsonRes googleCSEJSONResponse
	if err := json.Unmarshal(body, &jsonRes); err != nil {
		return nil, fmt.Errorf("JSON parse error: %v", err)
	}

	if jsonRes.Error != nil {
		return nil, fmt.Errorf("Google API Error (%d): %s", jsonRes.Error.Code, jsonRes.Error.Message)
	}

	var results []Result
	for _, item := range jsonRes.Items {
		if item.Link != "" {
			results = append(results, Result{RawURL: item.Link, Source: g.Name()})
		}
	}

	return results, nil
}
