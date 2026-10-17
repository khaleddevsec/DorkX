package runner

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/khaleddevsec/DorkX/internal/engine"
	"github.com/khaleddevsec/DorkX/internal/filter"
	"github.com/khaleddevsec/DorkX/internal/output"
	"github.com/khaleddevsec/DorkX/internal/proxy"
)

// Config holds all runtime configuration from CLI flags
type Config struct {
	Dorks        []string
	Limit        int    // Target number of unique results per dork (e.g: 50)
	DelayRange   string // "min-max" seconds e.g. "1-2"
	Proxy        string
	ProxyList    string
	Silent       bool
	UniqueDomain bool
	Threads      int
	Timeout      int
	OutputFile   string
	NoColor      bool
	StatusCode   bool   // Probe and display HTTP status code
	MatchCode    string // Filter by HTTP status code (e.g. "200" or "200,301")
	InsecureTLS  bool   // Disable TLS verification for probed target URLs
	Engine       engine.Engine
}

// Runner orchestrates the entire search lifecycle
type Runner struct {
	cfg         *Config
	proxyPool   *proxy.Pool
	mu          sync.Mutex
	seenURLs    map[string]bool
	seenDomains map[string]bool
	matchCodes  map[int]bool
	outWriter   *bufio.Writer
	outFile     *os.File
	totalFound  int
	delayMin    float64
	delayMax    float64
	probeClient *http.Client
}

// New constructs a Runner from a Config
func New(cfg *Config) *Runner {
	output.NoColor = cfg.NoColor

	if cfg.Limit <= 0 {
		cfg.Limit = 50
	}

	r := &Runner{
		cfg:         cfg,
		seenURLs:    make(map[string]bool),
		seenDomains: make(map[string]bool),
		matchCodes:  make(map[int]bool),
	}

	if cfg.MatchCode != "" {
		for _, part := range strings.Split(cfg.MatchCode, ",") {
			part = strings.TrimSpace(part)
			if code, err := strconv.Atoi(part); err == nil {
				r.matchCodes[code] = true
			}
		}
	}

	r.delayMin, r.delayMax = parseDelay(cfg.DelayRange)

	r.probeClient = &http.Client{
		Timeout: 6 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig:   &tls.Config{InsecureSkipVerify: cfg.InsecureTLS},
			DisableKeepAlives: false,
			MaxIdleConns:      100,
			IdleConnTimeout:   30 * time.Second,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}

	return r
}

// probeURL makes a quick GET request to check HTTP status code
func (r *Runner) probeURL(target string, proxyURL *url.URL) int {
	req, err := http.NewRequest("GET", target, nil)
	if err != nil {
		return 0
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "*/*")

	client := r.probeClient
	if proxyURL != nil {
		tr := &http.Transport{
			Proxy:             http.ProxyURL(proxyURL),
			TLSClientConfig:   &tls.Config{InsecureSkipVerify: r.cfg.InsecureTLS},
			DisableKeepAlives: false,
			MaxIdleConns:      100,
			IdleConnTimeout:   30 * time.Second,
		}
		client = &http.Client{
			Transport: tr,
			Timeout:   7 * time.Second,
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()

	return resp.StatusCode
}

// Run executes all dorks, respects concurrency, writes results
func (r *Runner) Run() {
	if r.cfg.OutputFile != "" {
		f, err := os.Create(r.cfg.OutputFile)
		if err != nil {
			output.Errorf("Cannot create output file: %v\n", err)
			os.Exit(1)
		}
		r.outFile = f
		r.outWriter = bufio.NewWriter(f)
		defer func() {
			r.mu.Lock()
			if r.outWriter != nil {
				_ = r.outWriter.Flush()
			}
			f.Close()
			r.mu.Unlock()
		}()
		if !r.cfg.Silent {
			output.Infof("Saving results to: %s\n", r.cfg.OutputFile)
		}
	}

	// Graceful shutdown on Interrupt (SIGINT / Ctrl+C)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		r.mu.Lock()
		if r.outWriter != nil {
			_ = r.outWriter.Flush()
		}
		if r.outFile != nil {
			_ = r.outFile.Close()
		}
		total := r.totalFound
		r.mu.Unlock()
		if !r.cfg.Silent {
			output.Warnf("\n[!] Interrupted by user. Collected %d unique URLs before exit.\n", total)
		}
		os.Exit(0)
	}()

	if r.cfg.ProxyList != "" {
		pool, err := proxy.NewPoolFromFile(r.cfg.ProxyList)
		if err != nil {
			output.Warnf("Could not load proxy list: %v\n", err)
		} else {
			r.proxyPool = pool
			if !r.cfg.Silent {
				output.Infof("Loaded %d proxies\n", pool.Len())
			}
		}
	} else if r.cfg.Proxy != "" {
		parsed, err := url.Parse(r.cfg.Proxy)
		if err != nil {
			output.Warnf("Invalid proxy URL: %v\n", err)
		} else {
			pool, err := proxy.NewPool([]string{parsed.String()})
			if err != nil {
				output.Warnf("Invalid proxy: %v\n", err)
			} else {
				r.proxyPool = pool
				if !r.cfg.Silent {
					output.Infof("Using proxy: %s\n", r.cfg.Proxy)
				}
			}
		}
	}

	if !r.cfg.Silent {
		output.Infof("Engine: %s | Target Limit: %d | Threads: %d | Delay: %s\n",
			r.cfg.Engine.Name(), r.cfg.Limit, r.cfg.Threads, r.cfg.DelayRange)
		if r.cfg.StatusCode {
			output.Infof("Live Probe: Enabled (-status-code)\n")
		}
		if len(r.matchCodes) > 0 {
			output.Infof("Match Status Code: %s\n", r.cfg.MatchCode)
		}
		output.Infof("Total dorks to process: %d\n\n", len(r.cfg.Dorks))
	}

	sem := make(chan struct{}, r.cfg.Threads)
	var wg sync.WaitGroup

	for i, dork := range r.cfg.Dorks {
		sem <- struct{}{}
		wg.Add(1)

		go func(idx int, d string) {
			defer wg.Done()
			defer func() { <-sem }()

			r.searchDork(idx+1, len(r.cfg.Dorks), d)
		}(i, dork)
	}

	wg.Wait()

	if !r.cfg.Silent {
		fmt.Fprintln(os.Stderr, "")
		output.Successf("Done! Total strictly matched unique URLs: %d\n", r.totalFound)
		if r.cfg.OutputFile != "" {
			output.Successf("Results saved to: %s\n", r.cfg.OutputFile)
		}
	}
}

// searchDork continuously queries until Target Limit is reached or index is exhausted
func (r *Runner) searchDork(idx, total int, dork string) {
	rule := filter.ExtractRulesFromDork(dork)
	clusterDorks := filter.GenerateClusterDorks(dork)

	if !r.cfg.Silent {
		output.Dorkf("[%d/%d] %s (Target: %d URLs)\n", idx, total, dork, r.cfg.Limit)
		if len(rule.InURLReqs) > 0 {
			var patterns []string
			for _, req := range rule.InURLReqs {
				patterns = append(patterns, req.RawPattern)
			}
			output.Grayf("  ↳ Strict inurl: %v\n", patterns)
		}
		if len(rule.SiteSuffixes) > 0 {
			output.Grayf("  ↳ Site filter: %v\n", rule.SiteSuffixes)
		}
		if len(rule.FileTypes) > 0 {
			output.Grayf("  ↳ File types: %v\n", rule.FileTypes)
		}
		if len(rule.ExcludedTerms) > 0 || len(rule.ExcludedSites) > 0 || len(rule.ExcludedInURLs) > 0 {
			output.Grayf("  ↳ Exclusions: %v %v %v\n", rule.ExcludedTerms, rule.ExcludedSites, rule.ExcludedInURLs)
		}
	}

	foundForDork := 0

	for _, activeDork := range clusterDorks {
		if foundForDork >= r.cfg.Limit {
			break
		}

		page := 0
		consecutiveEmpty := 0
		maxConsecutiveEmpty := 5
		maxPagesSafetyLimit := 40

		for foundForDork < r.cfg.Limit && page < maxPagesSafetyLimit && consecutiveEmpty < maxConsecutiveEmpty {
			if page > 0 {
				r.sleep()
			}

			proxyURL := r.nextProxy()

			results, err := r.cfg.Engine.Search(activeDork, page, proxyURL, r.cfg.Timeout)
			if err != nil {
				if strings.Contains(err.Error(), "CAPTCHA") {
					break
				}
				page++
				consecutiveEmpty++
				continue
			}

			if len(results) == 0 {
				consecutiveEmpty++
				page++
				continue
			}

			pageNew := 0
			for _, res := range results {
				if foundForDork >= r.cfg.Limit {
					break
				}

				cleaned := filter.ExtractRealURL(res.RawURL)

				if !filter.IsValidURL(cleaned, rule) {
					continue
				}

				domainKey := filter.RootDomain(cleaned)

				// Atomically check and reserve URL so concurrent threads never probe or output duplicates
				r.mu.Lock()
				if r.seenURLs[cleaned] {
					r.mu.Unlock()
					continue
				}
				if r.cfg.UniqueDomain && r.seenDomains[domainKey] {
					r.mu.Unlock()
					continue
				}
				r.seenURLs[cleaned] = true
				if r.cfg.UniqueDomain {
					r.seenDomains[domainKey] = true
				}
				r.mu.Unlock()

				// Live Status Code Probe (if -status-code or -mc is set)
				statusCode := 0
				if r.cfg.StatusCode || len(r.matchCodes) > 0 {
					statusCode = r.probeURL(cleaned, proxyURL)

					// Filter by match code if specified (e.g. -mc 200)
					if len(r.matchCodes) > 0 && !r.matchCodes[statusCode] {
						continue
					}
				}

				r.mu.Lock()
				r.totalFound++
				foundForDork++

				// Write to output file with immediate flush to prevent data loss
				if r.outWriter != nil {
					if r.cfg.StatusCode && statusCode > 0 {
						fmt.Fprintf(r.outWriter, "%s [%d]\n", cleaned, statusCode)
					} else {
						fmt.Fprintln(r.outWriter, cleaned)
					}
					_ = r.outWriter.Flush()
				}
				r.mu.Unlock()

				// Print to stdout (clean and thread-safe for pipes)
				if r.cfg.StatusCode && statusCode > 0 {
					output.ResultWithStatus(cleaned, statusCode)
				} else {
					output.Result(cleaned)
				}

				pageNew++
			}

			if pageNew > 0 {
				consecutiveEmpty = 0
			} else {
				consecutiveEmpty++
			}

			page++
		}
	}

	if !r.cfg.Silent {
		output.Successf("Dork complete: %d matched URLs found.\n\n", foundForDork)
	}
}

func (r *Runner) nextProxy() *url.URL {
	if r.proxyPool == nil {
		return nil
	}
	return r.proxyPool.Next()
}

func (r *Runner) sleep() {
	if r.delayMin <= 0 {
		return
	}
	spread := r.delayMax - r.delayMin
	secs := r.delayMin + rand.Float64()*spread
	time.Sleep(time.Duration(secs*1000) * time.Millisecond)
}

func parseDelay(s string) (float64, float64) {
	parts := strings.Split(s, "-")
	if len(parts) != 2 {
		return 1.0, 2.0
	}
	min, err1 := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	max, err2 := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err1 != nil || err2 != nil {
		return 1.0, 2.0
	}
	if min > max {
		min, max = max, min
	}
	return min, max
}
