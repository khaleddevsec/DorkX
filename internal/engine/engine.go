package engine

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
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
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",
	"Mozilla/5.0 (X11; Fedora; Linux x86_64; rv:124.0) Gecko/20100101 Firefox/124.0",
}

func randomUA() string {
	return userAgents[rand.Intn(len(userAgents))]
}

// ---- HTTP Client Builder ----------------------------------------------------

func buildClient(proxyURL *url.URL, timeout int) *http.Client {
	transport := &http.Transport{
		DisableKeepAlives: false,
		MaxIdleConns:      20,
		TLSClientConfig:   &tls.Config{InsecureSkipVerify: true},
	}

	if proxyURL != nil {
		transport.Proxy = http.ProxyURL(proxyURL)
	}

	effectiveTimeout := timeout
	if effectiveTimeout <= 0 {
		effectiveTimeout = 25
	}

	return &http.Client{
		Transport: transport,
		Timeout:   time.Duration(effectiveTimeout) * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// ---- Yahoo Engine ------------------------------------------------------------

type YahooEngine struct{}

func NewYahooEngine() *YahooEngine {
	return &YahooEngine{}
}

func (y *YahooEngine) Name() string { return "Yahoo" }

var yahooRUMatch = regexp.MustCompile(`/RU=([^/]+)/RK=`)
var yahooDirectHref = regexp.MustCompile(`href=["'](https?://[^"'\s>]+)["']`)

func (y *YahooEngine) Search(dork string, page int, proxyURL *url.URL, timeout int) ([]Result, error) {
	// Yahoo pagination step (1, 11, 21, 31, 41...)
	start := (page * 10) + 1
	searchURL := fmt.Sprintf(
		"https://search.yahoo.com/search?p=%s&n=30&b=%d",
		url.QueryEscape(dork),
		start,
	)

	client := buildClient(proxyURL, timeout)
	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", randomUA())
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 || resp.StatusCode == 503 {
		return nil, fmt.Errorf("rate limited (HTTP %d)", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return nil, err
	}

	bodyStr := string(body)
	var results []Result
	seen := make(map[string]bool)

	// 1. Match /RU=.../RK= tracking links
	matches := yahooRUMatch.FindAllStringSubmatch(bodyStr, -1)
	for _, m := range matches {
		decoded, err := url.QueryUnescape(m[1])
		if err == nil && strings.HasPrefix(decoded, "http") && !seen[decoded] {
			seen[decoded] = true
			results = append(results, Result{RawURL: decoded, Source: y.Name()})
		}
	}

	// 2. Match direct href links
	directMatches := yahooDirectHref.FindAllStringSubmatch(bodyStr, -1)
	for _, m := range directMatches {
		u := m[1]
		if !seen[u] && !strings.Contains(u, "yahoo.com") && !strings.Contains(u, "yimg.com") {
			seen[u] = true
			results = append(results, Result{RawURL: u, Source: y.Name()})
		}
	}

	return results, nil
}

// ---- Bing Engine -------------------------------------------------------------

type BingEngine struct{}

func NewBingEngine() *BingEngine {
	return &BingEngine{}
}

func (b *BingEngine) Name() string { return "Bing" }

var bingCkMatch = regexp.MustCompile(`href=["'](https://www\.bing\.com/ck/a\?[^"'\s>]+)["']`)
var bingDirectHref = regexp.MustCompile(`<h2><a\s+[^>]*href=["'](https?://[^"'\s>]+)["']`)

func decodeBingRedirect(raw string) string {
	if strings.Contains(raw, "&u=a1") || strings.Contains(raw, "&amp;u=a1") {
		var b64 string
		if strings.Contains(raw, "&u=a1") {
			b64 = strings.Split(strings.Split(raw, "&u=a1")[1], "&")[0]
		} else {
			b64 = strings.Split(strings.Split(raw, "&amp;u=a1")[1], "&")[0]
		}
		switch len(b64) % 4 {
		case 2:
			b64 += "=="
		case 3:
			b64 += "="
		}
		b64 = strings.ReplaceAll(strings.ReplaceAll(b64, "-", "+"), "_", "/")
		dec, err := base64.StdEncoding.DecodeString(b64)
		if err == nil && strings.HasPrefix(string(dec), "http") {
			return string(dec)
		}
	}
	return raw
}

func (b *BingEngine) Search(dork string, page int, proxyURL *url.URL, timeout int) ([]Result, error) {
	// Bing pagination: first=1, 11, 21, 31...
	start := (page * 10) + 1
	searchURL := fmt.Sprintf(
		"https://www.bing.com/search?q=%s&first=%d&count=30",
		url.QueryEscape(dork),
		start,
	)

	client := buildClient(proxyURL, timeout)
	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", randomUA())
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 || resp.StatusCode == 503 {
		return nil, fmt.Errorf("rate limited (HTTP %d)", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return nil, err
	}

	bodyStr := string(body)
	var results []Result
	seen := make(map[string]bool)

	// 1. Parse /ck/a tracking links
	matches := bingCkMatch.FindAllStringSubmatch(bodyStr, -1)
	for _, m := range matches {
		decoded := decodeBingRedirect(m[1])
		if !seen[decoded] && !strings.Contains(decoded, "bing.com") && !strings.Contains(decoded, "microsoft.com") {
			seen[decoded] = true
			results = append(results, Result{RawURL: decoded, Source: b.Name()})
		}
	}

	// 2. Parse direct h2/a links
	directMatches := bingDirectHref.FindAllStringSubmatch(bodyStr, -1)
	for _, m := range directMatches {
		u := m[1]
		if !seen[u] && !strings.Contains(u, "bing.com") && !strings.Contains(u, "microsoft.com") {
			seen[u] = true
			results = append(results, Result{RawURL: u, Source: b.Name()})
		}
	}

	return results, nil
}

// ---- Google HTML Engine -----------------------------------------------------

type GoogleEngine struct{}

func NewGoogleEngine() *GoogleEngine {
	return &GoogleEngine{}
}

func (g *GoogleEngine) Name() string { return "Google (HTML Scraper)" }

var googleHrefRe = regexp.MustCompile(`href=["'](/url\?q=[^"'&]+|https?://[^"'\s>]+)["' >]`)

func (g *GoogleEngine) Search(dork string, page int, proxyURL *url.URL, timeout int) ([]Result, error) {
	start := page * 10
	searchURL := fmt.Sprintf(
		"https://www.google.com/search?q=%s&hl=en&start=%d&filter=0",
		url.QueryEscape(dork), start,
	)

	client := buildClient(proxyURL, timeout)
	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", randomUA())
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 || resp.StatusCode == 503 {
		return nil, fmt.Errorf("rate limited (HTTP %d)", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return nil, err
	}

	bodyStr := string(body)
	lowerBody := strings.ToLower(bodyStr)
	if strings.Contains(lowerBody, "detected unusual traffic") ||
		strings.Contains(lowerBody, "recaptcha") ||
		strings.Contains(lowerBody, "sorry/index") {
		return nil, fmt.Errorf("CAPTCHA detected on Google")
	}

	var results []Result
	seen := make(map[string]bool)

	matches := googleHrefRe.FindAllStringSubmatch(bodyStr, -1)
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
	return &GoogleCSEEngine{APIKey: apiKey, CX: cx}
}

func (g *GoogleCSEEngine) Name() string { return "Google (Custom Search API)" }

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

// ---- Hybrid Multi-Engine Pool ------------------------------------------------

type HybridEngine struct {
	engines []Engine
}

func NewHybridEngine(engines ...Engine) *HybridEngine {
	if len(engines) == 0 {
		engines = []Engine{NewYahooEngine(), NewBingEngine()}
	}
	return &HybridEngine{engines: engines}
}

func (h *HybridEngine) Name() string {
	var names []string
	for _, e := range h.engines {
		names = append(names, e.Name())
	}
	return strings.Join(names, " + ")
}

func (h *HybridEngine) Search(dork string, page int, proxyURL *url.URL, timeout int) ([]Result, error) {
	var allResults []Result
	var mu sync.Mutex
	var wg sync.WaitGroup
	seen := make(map[string]bool)

	for _, eng := range h.engines {
		wg.Add(1)
		go func(e Engine) {
			defer wg.Done()
			res, err := e.Search(dork, page, proxyURL, timeout)
			if err != nil {
				return
			}
			mu.Lock()
			for _, r := range res {
				if !seen[r.RawURL] {
					seen[r.RawURL] = true
					allResults = append(allResults, r)
				}
			}
			mu.Unlock()
		}(eng)
	}

	wg.Wait()
	return allResults, nil
}
