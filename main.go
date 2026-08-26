package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/frozeniceee/DorkX/internal/engine"
	"github.com/frozeniceee/DorkX/internal/output"
	"github.com/frozeniceee/DorkX/internal/runner"
	"github.com/frozeniceee/DorkX/internal/updater"
)

// ANSI color codes
const (
	ansiWhite  = "\033[97m"
	ansiBold   = "\033[1m"
	ansiYellow = "\033[33m"
	ansiCyan   = "\033[36m"
	ansiGray   = "\033[90m"
	ansiReset  = "\033[0m"
)

const AppVersion = "v3.1.0"

func printBanner() {
	art := ansiWhite + ansiBold + `
     _            _    __  __
  __| | ___  _ __| | _\ \/ /
 / _' |/ _ \| '__| |/ /\  /
| (_| | (_) | |  |   < /  \
 \__,_|\___/|_|  |_|\_/_/\_\
` + ansiReset

	tagline := ansiBold + "              Dork CLI Tool" + ansiReset
	meta := ansiGray + "         by " + ansiReset +
		ansiYellow + "Fr0z3n_1c3" + ansiReset +
		ansiGray + " | " + ansiReset +
		ansiCyan + AppVersion + ansiReset +
		ansiGray + " | " + ansiReset +
		"https://github.com/frozeniceee/DorkX"

	fmt.Fprintln(os.Stderr, art)
	fmt.Fprintln(os.Stderr, tagline)
	fmt.Fprintln(os.Stderr, meta)
	fmt.Fprintln(os.Stderr, "")
}

func main() {
	// --- CLI Flags ---
	dork         := flag.String("d", "", "Single dork query string (e.g: -d \"inurl:php?id= -github\")")
	list         := flag.String("l", "", "Path to file containing dorks (one per line)")
	outputFile   := flag.String("o", "", "Save results to file (e.g: -o results.txt)")
	limit        := flag.Int("limit", 50, "Target number of unique matched URLs to collect (default: 50)")
	engineName   := flag.String("e", "hybrid", "Search engine: hybrid (default: Yahoo + Bing + DuckDuckGo + AOL), yahoo, bing, duckduckgo, google, google-api, or all")
	delay        := flag.String("delay", "1-2", "Delay range between requests in seconds (default: 1-2)")
	proxy        := flag.String("proxy", "", "HTTP/SOCKS5 proxy URL (e.g: socks5://127.0.0.1:9050 or http://proxy:8080)")
	proxyList    := flag.String("proxy-list", "", "Path to file with proxy list (round-robin rotation)")
	silent       := flag.Bool("silent", false, "Silent mode: only print matched URLs (no banner/logs)")
	uniqueDomain := flag.Bool("u", false, "Only output 1 URL per unique root domain")
	statusCode   := flag.Bool("status-code", false, "Probe and display HTTP status code (e.g: [200])")
	scShort      := flag.Bool("sc", false, "Alias for -status-code")
	matchCode    := flag.String("mc", "", "Match HTTP status code filter (e.g: -mc 200 or -mc 200,301)")
	threads      := flag.Int("t", 1, "Number of concurrent dork searches (default: 1)")
	timeout      := flag.Int("timeout", 25, "HTTP request timeout in seconds (default: 25)")
	apiKey       := flag.String("api-key", os.Getenv("GOOGLE_API_KEY"), "Google Custom Search JSON API Key (for google-api engine)")
	cx           := flag.String("cx", os.Getenv("GOOGLE_CX"), "Google Custom Search Engine ID / CX (for google-api engine)")
	noColor      := flag.Bool("nc", false, "Disable colored output")
	doUpdate     := flag.Bool("update", false, "Update DorkX to the latest version directly from GitHub")
	upShort      := flag.Bool("up", false, "Alias for -update")
	version      := flag.Bool("version", false, "Print version and exit")

	flag.Usage = printUsage
	flag.Parse()

	if *version {
		fmt.Println("dorkx " + AppVersion)
		os.Exit(0)
	}

	if *doUpdate || *upShort {
		if !*silent {
			printBanner()
		}
		updater.Update()
		os.Exit(0)
	}

	// --- Collect dorks from all sources ---
	var dorks []string

	// 1. From -d flag
	if *dork != "" {
		dorks = append(dorks, strings.TrimSpace(*dork))
	}

	// 2. From -l file
	if *list != "" {
		fileDorks, err := readLines(*list)
		if err != nil {
			output.Errorf("Could not read dork list file: %v\n", err)
			os.Exit(1)
		}
		dorks = append(dorks, fileDorks...)
	}

	// 3. From STDIN (pipe: cat dorks.txt | dorkx -silent)
	stat, err := os.Stdin.Stat()
	if err == nil && (stat.Mode()&os.ModeNamedPipe != 0) {
		stdinDorks, err := readLines("-")
		if err == nil {
			dorks = append(dorks, stdinDorks...)
		}
	}

	// If no dorks provided, print clean banner and usage guide
	if len(dorks) == 0 {
		if !*silent {
			printBanner()
			printUsage()
		}
		os.Exit(0)
	}

	if !*silent {
		printBanner()
	}

	// Select Search Engine
	var searchEngine engine.Engine
	switch strings.ToLower(*engineName) {
	case "yahoo":
		searchEngine = engine.NewYahooEngine()
	case "bing":
		searchEngine = engine.NewBingEngine()
	case "duckduckgo", "ddg":
		searchEngine = engine.NewDuckDuckGoEngine()
	case "aol":
		searchEngine = engine.NewAOLEngine()
	case "google":
		searchEngine = engine.NewGoogleEngine()
	case "google-api", "cse":
		if *apiKey == "" || *cx == "" {
			output.Errorf("Google API requires -api-key and -cx flags (or GOOGLE_API_KEY & GOOGLE_CX env vars)\n")
			os.Exit(1)
		}
		searchEngine = engine.NewGoogleCSEEngine(*apiKey, *cx)
	case "all":
		if *apiKey != "" && *cx != "" {
			searchEngine = engine.NewHybridEngine(
				engine.NewYahooEngine(),
				engine.NewBingEngine(),
				engine.NewDuckDuckGoEngine(),
				engine.NewAOLEngine(),
				engine.NewGoogleCSEEngine(*apiKey, *cx),
			)
		} else {
			searchEngine = engine.NewHybridEngine(
				engine.NewYahooEngine(),
				engine.NewBingEngine(),
				engine.NewDuckDuckGoEngine(),
				engine.NewAOLEngine(),
				engine.NewGoogleEngine(),
			)
		}
	case "hybrid":
		fallthrough
	default:
		searchEngine = engine.NewHybridEngine(
			engine.NewYahooEngine(),
			engine.NewBingEngine(),
			engine.NewDuckDuckGoEngine(),
			engine.NewAOLEngine(),
		)
	}

	// --- Build Config ---
	cfg := &runner.Config{
		Dorks:        dorks,
		Limit:        *limit,
		DelayRange:   *delay,
		Proxy:        *proxy,
		ProxyList:    *proxyList,
		Silent:       *silent,
		UniqueDomain: *uniqueDomain,
		StatusCode:   *statusCode || *scShort,
		MatchCode:    *matchCode,
		Threads:      *threads,
		Timeout:      *timeout,
		OutputFile:   *outputFile,
		NoColor:      *noColor,
		Engine:       searchEngine,
	}

	// --- Run ---
	r := runner.New(cfg)
	r.Run()
}

// readLines reads line-by-line from a file path or stdin ("-")
func readLines(path string) ([]string, error) {
	var scanner *bufio.Scanner

	if path == "-" {
		scanner = bufio.NewScanner(os.Stdin)
	} else {
		f, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		scanner = bufio.NewScanner(f)
	}

	var lines []string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			lines = append(lines, line)
		}
	}
	return lines, scanner.Err()
}

func printUsage() {
	fmt.Print(`
DorkX - Dork Recon & Target URL Collector CLI by Fr0z3n_1c3

USAGE:
  dorkx [flags]

INPUT OPTIONS:
  -d string          Single dork query string (supports inurl:, site:, filetype:, -exclusions)
  -l string          Path to file containing dorks, one per line
  stdin              Pipe dorks via standard input: cat dorks.txt | dorkx

COLLECTION & FILTERING:
  -limit int         Target number of unique matched URLs to collect (default: 50)
  -u                 Unique root domain mode: only output 1 URL per root domain
  -status-code, -sc  Live probe: verify and display HTTP status code (e.g: [200])
  -mc string         Match HTTP status codes only (e.g: -mc 200 or -mc 200,301)
  -t int             Number of concurrent dork searches (default: 1)
  -delay string      Delay range between requests in seconds (default: "1-2")
  -timeout int       HTTP request timeout in seconds (default: 25)

ENGINE SELECTION:
  -e string          Search engine: hybrid (default: Yahoo+Bing+DuckDuckGo+AOL), yahoo, bing, ddg, google, google-api, or all
  -api-key string    Google Custom Search API Key (for google-api engine)
  -cx string         Google Search Engine CX ID (for google-api engine)

PROXY CONFIGURATION:
  -proxy string      Single HTTP/SOCKS5 proxy (e.g: socks5://127.0.0.1:9050 or http://127.0.0.1:8080)
  -proxy-list string Path to file containing list of proxies (round-robin rotation)

UPDATE & MAINTENANCE:
  -update, -up       Update DorkX directly to the latest version from GitHub
  -version           Display version and exit

OUTPUT & DISPLAY:
  -o string          Save collected URLs to output file
  -silent            Silent mode: print clean URLs only to stdout (suppresses banner & logs)
  -nc                Disable colored output

DORK SYNTAX SUPPORTED:
  inurl:pattern      URL path or query must contain pattern (e.g: inurl:php?id=, inurl:/admin/)
  site:domain        Target specific domain or TLD (e.g: site:.gov.bd, site:.edu)
  filetype:ext       Target specific file extension (e.g: filetype:pdf, ext:sql, ext:env)
  intitle:"text"     Search engine title targeting
  "exact phrase"     Quotes for exact phrase matching
  -term              Exclude keywords from URLs or domains (e.g: -github -stackoverflow)
  -site:domain       Exclude specific domains (e.g: -site:facebook.com -site:.gov.bd)
  -inurl:pattern     Exclude specific path/param patterns (e.g: -inurl:login -inurl:test)

EXAMPLES:
  # 1. Basic Single Dork (Collects 50 unique targets):
  dorkx -d "Dork" -o results.txt

  # 2. Collect only live URLs responding with HTTP 200 OK:
  dorkx -d "Dork" -status-code -mc 200 -limit 50 -o live_targets.txt

  # 3. Collect 100 targets while excluding noise domains:
  dorkx -d "Dork -github -stackoverflow -.gov.bd" -limit 100 -o targets.txt

  # 4. Target government domains with 1 URL per unique domain:
  dorkx -d "site:.gov.bd Dork" -limit 30 -u -o gov_targets.txt

  # 5. Recon Pipeline (Dork -> HTTP probe -> SQLi vulnerability scan):
  dorkx -d "Dork" -limit 50 -silent | httpx -silent -mc 200 | nuclei -t sqli/

  # 6. Update DorkX to the latest version:
  dorkx -update
`)
}
