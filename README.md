<h1 align="center">
  <br>
  <pre>
     _            _    __  __
  __| | ___  _ __| | _\ \/ /
 / _' |/ _ \| '__| |/ /\  / 
| (_| | (_) | |  |   < /  \ 
 \__,_|\___/|_|  |_|\_/_/\_\
  </pre>
  <b>DorkX - High-Performance Dork Recon & Target URL Collector CLI</b>
</h1>

<p align="center">
  <b>Fast, zero-captcha, multi-engine dorking tool built for security researchers, penetration testers, and bug bounty hunters.</b>
</p>

<p align="center">
  <a href="https://golang.org"><img src="https://img.shields.io/badge/Language-Go%201.21+-00ADD8?style=flat&logo=go" alt="Go"></a>
  <a href="https://github.com/khaleddevsec/DorkX/releases"><img src="https://img.shields.io/badge/Version-v3.1.0-brightgreen?style=flat" alt="Version"></a>
  <a href="https://opensource.org/licenses/MIT"><img src="https://img.shields.io/badge/License-MIT-blue.svg" alt="License"></a>
  <a href="https://github.com/khaleddevsec/DorkX"><img src="https://img.shields.io/badge/Platform-Linux%20%7C%20macOS%20%7C%20Windows-lightgrey?style=flat" alt="Platform"></a>
  <a href="https://github.com/khaleddevsec/DorkX/actions"><img src="https://github.com/khaleddevsec/DorkX/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
</p>

---

## Overview

**DorkX** is a high-performance, concurrent reconnaissance CLI tool designed to find and collect strictly matching target URLs using search engine dorks. Built from the ground up for seamless integration into UNIX security pipelines alongside tools like `httpx`, `nuclei`, `subfinder`, and `katana`.

### Key Features

- **Limit-Based Target Collection (`-limit 50`)**: Specify the exact number of unique target URLs required (e.g. 50, 100, 500), and DorkX collects continuously across search pages until the limit is reached.
- **Hybrid Multi-Engine Pool (`-e hybrid`)**: Queries Yahoo, Bing, and DuckDuckGo concurrently out of the box with zero captchas, zero mandatory API keys, and no Tor setup required.
- **Universal Dork Operator Support**: Supports all standard search engine operators: `inurl:`, `site:`, `filetype:`, `ext:`, `intitle:`, `"exact quotes"`, keyword combos, and negative exclusions (`-github`, `-site:`, `-inurl:`).
- **Live HTTP Status Probe (`-status-code`, `-mc 200`)**: Verify if targets are live and filter out dead or unreachable pages directly within DorkX.
- **Root Domain Deduplication (`-u`)**: Automatically filters results to only keep 1 URL per unique root domain (supports ccTLDs like `.gov.bd`, `.co.uk`, `.com.au`).
- **Pipeline-First Architecture**: Clean target URLs are written to `stdout` and logs/banners to `stderr`, providing 100% compatibility with standard UNIX pipe workflows (`|`).
- **Safe Self-Updating (`-update`)**: Update DorkX directly to the latest release with a single command.
- **Zero Dependencies**: Standalone compiled binary with zero external runtime dependencies.

---

## Installation

### Method 1: Direct Install with Go (Recommended)

```bash
go install github.com/khaleddevsec/DorkX@latest
```

Ensure `$GOPATH/bin` or `$HOME/go/bin` is in your system's `PATH`.

### Method 2: Build from Source (Linux / macOS / Windows)

```bash
git clone https://github.com/khaleddevsec/DorkX.git
cd DorkX
go build -o dorkx .
sudo mv dorkx /usr/local/bin/
```

### Method 3: Pre-Built Binaries

Download pre-compiled binaries for Linux, macOS, and Windows directly from the [Releases](https://github.com/khaleddevsec/DorkX/releases) page.

---

## Self-Updating

Update DorkX directly from GitHub to the latest version anytime:

```bash
dorkx -update
```

---

## Quick Start & Usage

```bash
dorkx [flags]
```

### 1. Basic Single Dork (Collects 50 unique targets by default)
```bash
dorkx -d "inurl:php?id=" -o results.txt
```

### 2. Collect only Live URLs Responding with HTTP 200 OK
```bash
dorkx -d "inurl:php?id=" -status-code -mc 200 -limit 50 -o live_targets.txt
```

### 3. Collect an Exact Number of Targets (e.g. 100 targets)
```bash
dorkx -d "inurl:php?id=" -limit 100 -o targets.txt
```

### 4. Exclude Noise Domains & False Positives
```bash
dorkx -d "inurl:php?id= -github -stackoverflow -.gov.bd" -limit 50 -o clean_targets.txt
```

### 5. Target Specific Country / TLD with 1 URL per Domain (`-u`)
```bash
dorkx -d "site:.gov.bd inurl:index.php" -limit 30 -u -o gov_targets.txt
```

### 6. Pipe into `httpx` and `nuclei` (Bug Bounty Pipeline)
```bash
dorkx -d "inurl:php?id=" -limit 50 -silent | httpx -silent -mc 200 | nuclei -t sqli/
```

### 7. Bulk Dorking from File
```bash
dorkx -l target_dorks.txt -limit 50 -t 2 -o all_results.txt
```

---

## Flag Reference

| Flag | Type | Default | Description |
| :--- | :--- | :--- | :--- |
| **Input Options** | | | |
| `-d` | `string` | `""` | Single dork query string |
| `-l` | `string` | `""` | File containing dorks, one per line |
| `stdin` | `pipe` | `-` | Pipe dorks via standard input (`cat dorks.txt \| dorkx`) |
| **Collection & Filtering** | | | |
| `-limit` | `int` | `50` | Target number of unique strictly-matched URLs to collect |
| `-u` | `bool` | `false` | Unique domain mode: only output 1 URL per unique root domain |
| `-status-code`, `-sc` | `bool` | `false` | Live probe: verify and display HTTP status code (e.g. `[200]`) |
| `-mc` | `string` | `""` | Match status code filter (e.g. `-mc 200` or `-mc 200,301`) |
| `-insecure`, `-k` | `bool` | `false` | Disable TLS certificate verification for target status probing |
| `-t` | `int` | `1` | Number of concurrent dork searches |
| `-delay` | `string` | `"1-2"` | Delay range in seconds between batch requests |
| `-timeout` | `int` | `25` | HTTP request timeout in seconds |
| **Engine Selection** | | | |
| `-e` | `string` | `hybrid` | Search engine: `hybrid` (Yahoo+Bing+DuckDuckGo), `yahoo`, `bing`, `ddg`, `google`, `google-api`, or `all` |
| `-api-key` | `string` | `""` | Google Custom Search API Key (for `google-api` engine) |
| `-cx` | `string` | `""` | Google Search Engine CX ID (for `google-api` engine) |
| **Maintenance** | | | |
| `-update`, `-up` | `bool` | `false` | Update DorkX directly to the latest version from GitHub |
| `-version` | `bool` | `false` | Print version and exit |
| **Proxy Options** | | | |
| `-proxy` | `string` | `""` | Single HTTP/SOCKS5 proxy (`socks5://127.0.0.1:9050` or `http://127.0.0.1:8080`) |
| `-proxy-list` | `string` | `""` | Path to file containing list of proxies for rotation |
| **Output & Formatting** | | | |
| `-o` | `string` | `""` | File path to save collected URLs |
| `-silent` | `bool` | `false` | Silent mode: outputs clean URLs only to stdout |
| `-nc` | `bool` | `false` | Disable colored terminal output |

---

## Supported Dork Operators

DorkX supports native search engine operator syntax:

| Operator | Syntax Example | Meaning |
| :--- | :--- | :--- |
| **inurl:** | `inurl:php?id=`, `inurl:/admin/` | URL path or query parameters must contain this pattern |
| **site:** | `site:.gov.bd` or `site:example.com` | Target URL domain must match or end with this domain/TLD |
| **filetype: / ext:** | `filetype:pdf`, `ext:sql`, `ext:env` | Target URL must end with this file extension |
| **intitle:** | `intitle:"index of"` | Search engine title matching |
| **Quotes ("...")** | `"admin login"` | Exact phrase matching |
| **Negative Term (-)** | `-github -stackoverflow` | Exclude any URL containing these keywords |
| **Negative Site (-site:)** | `-site:facebook.com -site:.gov` | Exclude specific domains or TLDs |
| **Negative inurl (-inurl:)** | `-inurl:login -inurl:test` | Exclude specific path/parameter patterns |

### Combined Query Example:
```bash
dorkx -d "inurl:item.php?id= -site:amazon.com -site:ebay.com -stackoverflow filetype:php" -limit 50
```

---

## Recon Pipeline Workflows

### 1. Dorking to Live Host Probing with `httpx`
```bash
dorkx -d "inurl:php?id=" -limit 50 -silent | httpx -silent -status-code -title
```

### 2. Dorking to Vulnerability Scanning with `nuclei`
```bash
dorkx -d "inurl:php?id=" -limit 50 -silent | httpx -silent -mc 200 | nuclei -tags sqli -o sqli_vulnerable.txt
```

### 3. Save Targets and Pipe Simultaneously with `tee`
```bash
dorkx -d "inurl:php?id=" -limit 100 -silent | tee targets.txt | httpx -silent
```

---

## Bulk Dork List File Format (`-l`)

Pass any text file containing dork queries (one per line). Lines starting with `#` are ignored as comments:

```text
# SQL Injection Dorks
inurl:index.php?id= -github
inurl:page.php?id= -stackoverflow
inurl:product.php?id=

# Sensitive Files
filetype:env intext:DB_PASSWORD
filetype:sql "dump"
filetype:pdf "confidential"

# Specific TLD Targeting
site:.gov.bd inurl:page.php?id=
site:.edu inurl:index.php?id=
```

Run in bulk:
```bash
dorkx -l target_dorks.txt -limit 50 -o bulk_targets.txt
```

---

## Legal & Ethical Disclaimer

> [!WARNING]
> This tool is intended strictly for authorized educational purposes, defensive security auditing, and legitimate bug bounty research on programs where you have explicit permission to test. The author assumes no liability and is not responsible for any misuse or damage caused by this utility. Always comply with applicable local, state, and federal cybersecurity laws and program scopes.

---

## Contributing

Contributions, issues, and feature requests are welcome! Feel free to check the [issues page](https://github.com/khaleddevsec/DorkX/issues) or submit a pull request.

---

## License

This project is licensed under the **MIT License** — see the [LICENSE](LICENSE) file for details.

---

<p align="center">
  Maintained by <b><a href="https://github.com/khaleddevsec">KhaledDevSec</a></b>
</p>
