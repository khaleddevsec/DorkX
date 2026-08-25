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

const banner = `
     _            _    __  __
  __| | ___  _ __| | _\ \/ /
 / _' |/ _ \| '__| |/ /\  /
| (_| | (_) | |  |   < /  \
 \__,_|\___/|_|  |_|\_/_/\_\

     Google Dork CLI Tool
     by Fr0z3n_1c3 | v1.0.0
`

func main() {
	// --- CLI Flags ---
	dork        := flag.String("d", "", "Single dork query (e.g: \"inurl:php?id=\")")
	list        := flag.String("l", "", "Path to file containing dorks (one per line)")
	outputFile  := flag.String("o", "", "Save results to file (e.g: -o results.txt)")
	pages       := flag.Int("p", 3, "Number of Google result pages to scrape per dork (default: 3)")
	delay       := flag.String("delay", "2-5", "Delay range between requests in seconds (e.g: 2-5)")
	proxy       := flag.String("proxy", "", "HTTP/SOCKS5 proxy URL (e.g: http://127.0.0.1:8080)")
	proxyList   := flag.String("proxy-list", "", "Path to file with proxy list (round-robin rotation)")
	silent      := flag.Bool("silent", false, "Silent mode: only print matched URLs (no banner/logs)")
	uniqueDomain := flag.Bool("u", false, "Only output 1 URL per unique root domain")
	threads     := flag.Int("t", 1, "Number of concurrent dork searches (default: 1)")
	timeout     := flag.Int("timeout", 15, "HTTP request timeout in seconds (default: 15)")
	noColor     := flag.Bool("nc", false, "Disable colored output")
	version     := flag.Bool("version", false, "Print version and exit")

	flag.Usage = printUsage
	flag.Parse()

	if *version {
		fmt.Println("dorkx v1.0.0")
		os.Exit(0)
	}

	if !*silent {
		fmt.Println(banner)
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

	// --- Build Config ---
	cfg := &runner.Config{
		Dorks:        dorks,
		Pages:        *pages,
		DelayRange:   *delay,
		Proxy:        *proxy,
		ProxyList:    *proxyList,
		Silent:       *silent,
		UniqueDomain: *uniqueDomain,
		Threads:      *threads,
		Timeout:      *timeout,
		OutputFile:   *outputFile,
		NoColor:      *noColor,
		Engine:       engine.NewGoogleEngine(),
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
dorkx - Google Dork CLI Tool by Fr0z3n_1c3

USAGE:
  dorkx [flags]

INPUT FLAGS:
  -d "query"         Single dork query string
  -l file.txt        File containing dorks, one per line
                     (Also accepts STDIN pipe: cat dorks.txt | dorkx)

SEARCH FLAGS:
  -p  int            Number of Google result pages per dork (default: 3)
  -t  int            Concurrent dork searches (default: 1)
  -delay "min-max"   Delay range in seconds between requests (default: "2-5")
  -timeout int       HTTP timeout per request in seconds (default: 15)

FILTER FLAGS:
  -u                 Only output 1 URL per unique root domain

PROXY FLAGS:
  -proxy   url       Single HTTP/SOCKS5 proxy (e.g: http://127.0.0.1:8080)
  -proxy-list file   File of proxies for round-robin rotation

OUTPUT FLAGS:
  -o  file.txt       Save results to output file
  -silent            Only print matching URLs (no banner, no logs)
  -nc                Disable colored output
  -version           Print version and exit

EXAMPLES:
  dorkx -d "inurl:index.php?id=" -p 5 -silent
  dorkx -d "inurl:page.php?id=" -u -o results.txt
  dorkx -l dorks.txt -p 3 -t 2 -silent | httpx -silent -mc 200
  cat dorks.txt | dorkx -silent | tee targets.txt
  dorkx -d "site:.gov.bd inurl:page.php?id=" -p 10 -delay 3-7 -o gov_targets.txt

`)
}
