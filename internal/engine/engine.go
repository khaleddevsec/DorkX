package engine

import (
	"crypto/tls"
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
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",
}

func randomUA() string {
	return userAgents[rand.Intn(len(userAgents))]
}

// ---- HTTP Client Builder ----------------------------------------------------

func buildClient(proxyURL *url.URL, timeout int) *http.Client {
	transport := &http.Transport{
		DisableKeepAlives: false,
		MaxIdleConns:      10,
		TLSClientConfig:   &tls.Config{InsecureSkipVerify: true},
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

// ---- Yahoo Engine (Default & Zero-Captcha) -----------------------------------

type YahooEngine struct{}

func NewYahooEngine() *YahooEngine {
	return &YahooEngine{}
}

func (y *YahooEngine) Name() string { return "Yahoo" }

var yahooRe = regexp.MustCompile(`/RU=([^/]+)/RK=`)

func (y *YahooEngine) Search(dork string, page int, proxyURL *url.URL, timeout int) ([]Result, error) {
	// Yahoo pagination: b=1, 51, 101...
	start := (page * 50) + 1
	searchURL := fmt.Sprintf(
		"https://search.yahoo.com/search?p=%s&n=50&b=%d",
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

	var results []Result
	seen := make(map[string]bool)

	matches := yahooRe.FindAllStringSubmatch(string(body), -1)
	for _, m := range matches {
		decoded, err := url.QueryUnescape(m[1])
		if err != nil || !strings.HasPrefix(decoded, "http") {
			continue
		}

		if !seen[decoded] {
			seen[decoded] = true
			results = append(results, Result{RawURL: decoded, Source: y.Name()})
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
		return nil, fmt.Errorf("CAPTCHA or rate-limit detected on page %d", page+1)
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
	return &GoogleCSEEngine{
		APIKey: apiKey,
		CX:     cx,
	}
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

// ---- Multi-Engine Provider ---------------------------------------------------

type MultiEngine struct {
	engines []Engine
}

func NewMultiEngine(engines ...Engine) *MultiEngine {
	return &MultiEngine{engines: engines}
}

func (m *MultiEngine) Name() string {
	var names []string
	for _, e := range m.engines {
		names = append(names, e.Name())
	}
	return strings.Join(names, " + ")
}

func (m *MultiEngine) Search(dork string, page int, proxyURL *url.URL, timeout int) ([]Result, error) {
	var allResults []Result
	seen := make(map[string]bool)

	for _, eng := range m.engines {
		res, err := eng.Search(dork, page, proxyURL, timeout)
		if err != nil {
			continue // fallback smoothly to next engine
		}
		for _, r := range res {
			if !seen[r.RawURL] {
				seen[r.RawURL] = true
				allResults = append(allResults, r)
			}
		}
	}

	return allResults, nil
}
