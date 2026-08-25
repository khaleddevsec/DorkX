package runner

import (
	"bufio"
	"fmt"
	"math/rand"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/frozeniceee/DorkX/internal/engine"
	"github.com/frozeniceee/DorkX/internal/filter"
	"github.com/frozeniceee/DorkX/internal/output"
	"github.com/frozeniceee/DorkX/internal/proxy"
)

// Config holds all runtime configuration from CLI flags
type Config struct {
	Dorks        []string
	Pages        int
	DelayRange   string // "min-max" seconds e.g. "2-5"
	Proxy        string
	ProxyList    string
	Silent       bool
	UniqueDomain bool
	Threads      int
	Timeout      int
	OutputFile   string
	NoColor      bool
	Engine       engine.Engine
}

// Runner orchestrates the entire search lifecycle
type Runner struct {
	cfg        *Config
	proxyPool  *proxy.Pool
	mu         sync.Mutex
	seenURLs   map[string]bool
	seenDomains map[string]bool
	outWriter  *bufio.Writer
	outFile    *os.File
	totalFound int
	delayMin   float64
	delayMax   float64
}

// New constructs a Runner from a Config
func New(cfg *Config) *Runner {
	output.NoColor = cfg.NoColor

	r := &Runner{
		cfg:         cfg,
		seenURLs:    make(map[string]bool),
		seenDomains: make(map[string]bool),
	}

	// Parse delay range
	r.delayMin, r.delayMax = parseDelay(cfg.DelayRange)

	return r
}

// Run executes all dorks, respects concurrency, writes results
func (r *Runner) Run() {
	// Open output file if specified
	if r.cfg.OutputFile != "" {
		f, err := os.Create(r.cfg.OutputFile)
		if err != nil {
			output.Errorf("Cannot create output file: %v\n", err)
			os.Exit(1)
		}
		r.outFile = f
		r.outWriter = bufio.NewWriter(f)
		defer func() {
			r.outWriter.Flush()
			f.Close()
		}()
		if !r.cfg.Silent {
			output.Infof("Saving results to: %s\n", r.cfg.OutputFile)
		}
	}

	// Build proxy pool
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
			r.proxyPool, _ = proxy.NewPool([]string{parsed.String()})
			if !r.cfg.Silent {
				output.Infof("Using proxy: %s\n", r.cfg.Proxy)
			}
		}
	}

	if !r.cfg.Silent {
		output.Infof("Engine: %s | Pages: %d | Threads: %d | Delay: %s\n",
			r.cfg.Engine.Name(), r.cfg.Pages, r.cfg.Threads, r.cfg.DelayRange)
		output.Infof("Total dorks to process: %d\n\n", len(r.cfg.Dorks))
	}

	// Semaphore channel to limit concurrency
	sem := make(chan struct{}, r.cfg.Threads)
	var wg sync.WaitGroup

	for i, dork := range r.cfg.Dorks {
		sem <- struct{}{} // acquire slot
		wg.Add(1)

		go func(idx int, d string) {
			defer wg.Done()
			defer func() { <-sem }() // release slot

			r.searchDork(idx+1, len(r.cfg.Dorks), d)
		}(i, dork)
	}

	wg.Wait()

	// Final summary
	if !r.cfg.Silent {
		fmt.Fprintln(os.Stderr, "")
		output.Successf("Done! Total matched URLs: %d\n", r.totalFound)
		if r.cfg.OutputFile != "" {
			output.Successf("Results saved to: %s\n", r.cfg.OutputFile)
		}
	}
}

// searchDork runs a full paginated search for a single dork query
func (r *Runner) searchDork(idx, total int, dork string) {
	rule := filter.ExtractRulesFromDork(dork)

	if !r.cfg.Silent {
		output.Dorkf("[%d/%d] %s\n", idx, total, dork)
		if len(rule.InURLPatterns) > 0 {
			output.Grayf("  ↳ Strict patterns: %v\n", rule.InURLPatterns)
		}
		if len(rule.SiteSuffixes) > 0 {
			output.Grayf("  ↳ Site filter: %v\n", rule.SiteSuffixes)
		}
	}

	foundForDork := 0

	for page := 0; page < r.cfg.Pages; page++ {
		// Delay before each page request (not before page 0 on first dork)
		if page > 0 {
			r.sleep()
		}

		proxyURL := r.nextProxy()

		if !r.cfg.Silent {
			output.Grayf("  ↳ Page %d/%d ...\n", page+1, r.cfg.Pages)
		}

		results, err := r.cfg.Engine.Search(dork, page, proxyURL, r.cfg.Timeout)
		if err != nil {
			if strings.Contains(err.Error(), "CAPTCHA") {
				output.Warnf("CAPTCHA detected on dork [%s] page %d. Try adding -proxy or increasing -delay.\n", dork, page+1)
				break
			}
			output.Warnf("Page %d error for dork [%s]: %v\n", page+1, dork, err)
			// Short pause before retrying next page
			time.Sleep(2 * time.Second)
			continue
		}

		pageNew := 0
		for _, res := range results {
			// Unwrap Google redirects
			cleaned := filter.ExtractRealURL(res.RawURL)

			// Apply strict filter
			if !filter.IsValidURL(cleaned, rule) {
				continue
			}

			// Deduplication
			r.mu.Lock()
			alreadySeen := r.seenURLs[cleaned]
			domainKey := filter.RootDomain(cleaned)
			domainSeen := r.cfg.UniqueDomain && r.seenDomains[domainKey]
			r.mu.Unlock()

			if alreadySeen || domainSeen {
				continue
			}

			r.mu.Lock()
			r.seenURLs[cleaned] = true
			r.seenDomains[domainKey] = true
			r.totalFound++
			r.mu.Unlock()

			// Write to stdout (always — this is what gets piped)
			output.Result(cleaned)

			// Also write to file if configured
			if r.outWriter != nil {
				r.mu.Lock()
				fmt.Fprintln(r.outWriter, cleaned)
				r.mu.Unlock()
			}

			pageNew++
			foundForDork++
		}

		if !r.cfg.Silent {
			output.Grayf("  ↳ Page %d: +%d new URLs\n", page+1, pageNew)
		}

		// If a page returned zero raw results, we've likely hit the end
		if len(results) == 0 {
			if !r.cfg.Silent {
				output.Grayf("  ↳ No more results from engine. Stopping pagination.\n")
			}
			break
		}

		// Delay before next page
		r.sleep()
	}

	if !r.cfg.Silent {
		output.Successf("Dork complete: %d matched URLs found.\n\n", foundForDork)
	}
}

// nextProxy returns the next proxy from the pool (nil if none configured)
func (r *Runner) nextProxy() *url.URL {
	if r.proxyPool == nil {
		return nil
	}
	return r.proxyPool.Next()
}

// sleep waits for a random duration within the configured delay range
func (r *Runner) sleep() {
	if r.delayMin <= 0 {
		return
	}
	spread := r.delayMax - r.delayMin
	secs := r.delayMin + rand.Float64()*spread
	time.Sleep(time.Duration(secs*1000) * time.Millisecond)
}

// parseDelay parses "min-max" into two float64 values (seconds)
func parseDelay(s string) (float64, float64) {
	parts := strings.Split(s, "-")
	if len(parts) != 2 {
		return 2.0, 5.0
	}
	min, err1 := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	max, err2 := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err1 != nil || err2 != nil {
		return 2.0, 5.0
	}
	if min > max {
		min, max = max, min
	}
	return min, max
}
