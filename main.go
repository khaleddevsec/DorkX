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
)

// ANSI color codes for the banner
const (
	ansiCyan   = "\033[36m"
	ansiBold   = "\033[1m"
	ansiYellow = "\033[33m"
	ansiGray   = "\033[90m"
	ansiReset  = "\033[0m"
)

func printBanner() {
	art := ansiCyan + ansiBold + `
     _            _    __  __
  __| | ___  _ __| | _\ \/ /
 / _' |/ _ \| '__| |/ /\  /
| (_| | (_) | |  |   < /  \
 \__,_|\___/|_|  |_|\_/_/\_\
` + ansiReset

	tagline := ansiBold + "         Google Dork CLI Tool" + ansiReset
	meta := ansiGray + "         by " + ansiReset +
		ansiYellow + "Fr0z3n_1c3" + ansiReset +
		ansiGray + " | " + ansiReset +
		ansiCyan + "v2.0.0" + ansiReset +
		ansiGray + " | " + ansiReset +
		"https://github.com/frozeniceee/DorkX"

	fmt.Fprintln(os.Stderr, art)
	fmt.Fprintln(os.Stderr, tagline)
	fmt.Fprintln(os.Stderr, meta)
	fmt.Fprintln(os.Stderr, "")
}

func main() {
	// --- CLI Flags ---
	dork         := flag.String("d", "", "Single dork query (e.g: \"inurl:php?id=\")")
	list         := flag.String("l", "", "Path to file containing dorks (one per line)")
	outputFile   := flag.String("o", "", "Save results to file (e.g: -o results.txt)")
	limit        := flag.Int("limit", 50, "Target number of strictly-matched unique URLs to collect (default: 50)")
	engineName   := flag.String("e", "hybrid", "Search engine: hybrid (default: Yahoo + Bing), yahoo, bing, google, google-api, or all")
	delay        := flag.String("delay", "1-2", "Delay range between requests in seconds (default: 1-2)")
	proxy        := flag.String("proxy", "", "HTTP/SOCKS5 proxy URL (e.g: socks5://127.0.0.1:9050 or http://proxy:8080)")
	proxyList    := flag.String("proxy-list", "", "Path to file with proxy list (round-robin rotation)")
	silent       := flag.Bool("silent", false, "Silent mode: only print matched URLs (no banner/logs)")
	uniqueDomain := flag.Bool("u", false, "Only output 1 URL per unique root domain")
	threads      := flag.Int("t", 1, "Number of concurrent dork searches (default: 1)")
	timeout      := flag.Int("timeout", 25, "HTTP request timeout in seconds (default: 25)")
	apiKey       := flag.String("api-key", os.Getenv("GOOGLE_API_KEY"), "Google Custom Search JSON API Key (for google-api engine)")
	cx           := flag.String("cx", os.Getenv("GOOGLE_CX"), "Google Custom Search Engine ID / CX (for google-api engine)")
	noColor      := flag.Bool("nc", false, "Disable colored output")
	version      := flag.Bool("version", false, "Print version and exit")

	flag.Usage = printUsage
	flag.Parse()

	if *version {
		fmt.Println("dorkx v2.0.0")
		os.Exit(0)
	}

	if !*silent {
		printBanner()
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
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		stdinDorks, err := readLines("-")
		if err == nil {
			dorks = append(dorks, stdinDorks...)
		}
	}

	if len(dorks) == 0 {
		if !*silent {
			output.Errorf("No dork provided. Use -d \"dork\" or -l dorks.txt or pipe via stdin.\n")
			fmt.Println("\nRun 'dorkx -h' for help.")
		}
		os.Exit(1)
	}

	// Select Search Engine
	var searchEngine engine.Engine
	switch strings.ToLower(*engineName) {
	case "yahoo":
		searchEngine = engine.NewYahooEngine()
	case "bing":
		searchEngine = engine.NewBingEngine()
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
			searchEngine = engine.NewHybridEngine(engine.NewYahooEngine(), engine.NewBingEngine(), engine.NewGoogleCSEEngine(*apiKey, *cx))
		} else {
			searchEngine = engine.NewHybridEngine(engine.NewYahooEngine(), engine.NewBingEngine(), engine.NewGoogleEngine())
		}
	case "hybrid":
		fallthrough
	default:
		searchEngine = engine.NewHybridEngine(engine.NewYahooEngine(), engine.NewBingEngine())
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

INPUT FLAGS:
  -d "query"         Single dork query string (e.g: -d "inurl:php?id=")
  -l file.txt        File containing dorks, one per line
                     (Also accepts STDIN pipe: cat dorks.txt | dorkx)

COLLECTION & FILTER FLAGS:
  -limit int         Target number of unique matched URLs to collect (default: 50)
  -u                 Only output 1 URL per unique root domain
  -t int             Concurrent dork searches (default: 1)
  -delay "min-max"   Delay range in seconds between requests (default: "1-2")
  -timeout int       HTTP timeout per request in seconds (default: 25)

ENGINE FLAGS:
  -e name            Engine: hybrid (default: Yahoo + Bing), yahoo, bing, google, google-api, or all
  -api-key key       Google Custom Search API Key (for google-api engine)
  -cx id             Google Search Engine CX ID (for google-api engine)

PROXY FLAGS:
  -proxy   url       Single HTTP/SOCKS5 proxy (e.g: socks5://127.0.0.1:9050 or http://127.0.0.1:8080)
  -proxy-list file   File of proxies for round-robin rotation

OUTPUT FLAGS:
  -o  file.txt       Save results to output file
  -silent            Only print matching URLs (no banner, no logs)
  -nc                Disable colored output
  -version           Print version and exit

EXAMPLES:
  # Collect 50 unique targets instantly (Zero CAPTCHA, High Volume):
  dorkx -d "inurl:php?id=" -o results.txt

  # Collect 100 targets with specific dork:
  dorkx -d "inurl:index.php?id=" -limit 100 -o targets.txt

  # Target specific TLD with unique root domains:
  dorkx -d "site:.gov.bd inurl:page.php?id=" -limit 50 -u -o gov_targets.txt

  # Pipe directly into httpx & nuclei:
  dorkx -d "inurl:view.php?id=" -limit 50 -silent | httpx -silent -mc 200 | nuclei -t sqli/
`)
}
