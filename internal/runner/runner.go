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
	Limit        int    // Target number of unique results per dork (e.g: 50)
	DelayRange   string // "min-max" seconds e.g. "1-3"
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
	cfg         *Config
	proxyPool   *proxy.Pool
	mu          sync.Mutex
	seenURLs    map[string]bool
	seenDomains map[string]bool
	outWriter   *bufio.Writer
	outFile     *os.File
	totalFound  int
	delayMin    float64
	delayMax    float64
}

// New constructs a Runner from a Config
func New(cfg *Config) *Runner {
	output.NoColor = cfg.NoColor

	if cfg.Limit <= 0 {
		cfg.Limit = 50 // default to 50 results
	}

	r := &Runner{
		cfg:         cfg,
		seenURLs:    make(map[string]bool),
		seenDomains: make(map[string]bool),
	}

	r.delayMin, r.delayMax = parseDelay(cfg.DelayRange)
	return r
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
			r.outWriter.Flush()
			f.Close()
		}()
		if !r.cfg.Silent {
			output.Infof("Saving results to: %s\n", r.cfg.OutputFile)
		}
	}

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
		output.Infof("Engine: %s | Target Limit: %d | Threads: %d | Delay: %s\n",
			r.cfg.Engine.Name(), r.cfg.Limit, r.cfg.Threads, r.cfg.DelayRange)
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

	if !r.cfg.Silent {
		output.Dorkf("[%d/%d] %s (Target: %d URLs)\n", idx, total, dork, r.cfg.Limit)
		if len(rule.InURLReqs) > 0 {
			var patterns []string
			for _, req := range rule.InURLReqs {
				patterns = append(patterns, req.RawPattern)
			}
			output.Grayf("  ↳ Strict patterns: %v\n", patterns)
		}
		if len(rule.SiteSuffixes) > 0 {
			output.Grayf("  ↳ Site filter: %v\n", rule.SiteSuffixes)
		}
	}

	foundForDork := 0
	page := 0
	consecutiveEmpty := 0
	maxConsecutiveEmpty := 4
	maxPagesSafetyLimit := 40

	for foundForDork < r.cfg.Limit && page < maxPagesSafetyLimit && consecutiveEmpty < maxConsecutiveEmpty {
		if page > 0 {
			r.sleep()
		}

		proxyURL := r.nextProxy()

		if !r.cfg.Silent {
			output.Grayf("  ↳ Fetching batch %d (Collected: %d/%d) ...\n", page+1, foundForDork, r.cfg.Limit)
		}

		results, err := r.cfg.Engine.Search(dork, page, proxyURL, r.cfg.Timeout)
		if err != nil {
			if strings.Contains(err.Error(), "CAPTCHA") {
				output.Warnf("CAPTCHA detected on dork [%s]. Try adding -proxy or using hybrid engine.\n", dork)
				break
			}
			output.Warnf("Batch %d error for dork [%s]: %v\n", page+1, dork, err)
			time.Sleep(1 * time.Second)
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

			// Print to stdout (clean for piping)
			output.Result(cleaned)

			if r.outWriter != nil {
				r.mu.Lock()
				fmt.Fprintln(r.outWriter, cleaned)
				r.mu.Unlock()
			}

			pageNew++
			foundForDork++
		}

		if pageNew > 0 {
			consecutiveEmpty = 0
		} else {
			consecutiveEmpty++
		}

		page++

		if foundForDork >= r.cfg.Limit {
			if !r.cfg.Silent {
				output.Successf("Target limit of %d URLs reached for this dork!\n", r.cfg.Limit)
			}
			break
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
		return 1.0, 3.0
	}
	min, err1 := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	max, err2 := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err1 != nil || err2 != nil {
		return 1.0, 3.0
	}
	if min > max {
		min, max = max, min
	}
	return min, max
}
