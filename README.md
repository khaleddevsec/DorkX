<h1 align="center">
  <br>
  <pre>
     _            _    __  __
  __| | ___  _ __| | _\ \/ /
 / _' |/ _ \| '__| |/ /\  / 
| (_| | (_) | |  |   < /  \ 
 \__,_|\___/|_|  |_|\_/_/\_\
  </pre>
  <b>DorkX - Dork Recon & Target URL Collector CLI</b>
</h1>

<p align="center">
  <b>Fast, zero-captcha, strict multi-engine dorking tool built for security researchers and bug bounty hunters.</b>
</p>

<p align="center">
  <a href="https://golang.org"><img src="https://img.shields.io/badge/Language-Go%201.21+-00ADD8?style=flat&logo=go" alt="Go"></a>
  <a href="https://github.com/frozeniceee/DorkX/releases"><img src="https://img.shields.io/badge/Version-v2.0.0-brightgreen?style=flat" alt="Version"></a>
  <a href="https://opensource.org/licenses/MIT"><img src="https://img.shields.io/badge/License-MIT-blue.svg" alt="License"></a>
  <a href="https://github.com/frozeniceee/DorkX"><img src="https://img.shields.io/badge/Platform-Linux%20%7C%20macOS%20%7C%20Windows-lightgrey?style=flat" alt="Platform"></a>
</p>

---

## ⚡ What is DorkX?

**DorkX** is a high-performance reconnaissance CLI tool designed to find and collect strictly-matching target URLs using search engine dorks. Built to seamlessly integrate into UNIX security pipelines alongside tools like `httpx`, `nuclei`, `subfinder`, and `katana`.

### 🔥 Key Features

- **🎯 Limit-Based Target Collection (`-limit 50`)**: Tell DorkX how many unique target URLs you want (e.g. 50, 100, 500) and it collects continuously until your target is met.
- **⚡ Hybrid Multi-Engine Engine (`-e hybrid`)**: Queries **Yahoo and Bing in parallel** out of the box — **Zero CAPTCHAs, Zero Tor, Zero Proxies needed**.
- **🧠 Smart Strict Matching**: When you search `inurl:php?id=`, DorkX strictly enforces that candidate URLs contain `.php` and an `id=` parameter (e.g. `page.php?cat=1&id=10` or `view.php?id=2`), discarding search engine noise, ads, and irrelevant pages.
- **🚫 Full Exclusion & Negative Operators**: Supports `-term`, `-site:domain`, `-inurl:pattern`, and `filetype:` directly in queries (e.g. `inurl:php?id= -github -stackoverflow -.gov.bd`).
- **🔗 Pipeline-First Design**: URLs are printed to `stdout` and logs to `stderr`, making it 100% pipe-compatible with `httpx` and `nuclei`.
- **📦 Zero Dependencies**: Single compiled standalone binary with no runtime dependencies.

---

## 📥 Installation

### Method 1: Build from Source (Linux / WSL / macOS)

```bash
# 1. Clone repository
git clone https://github.com/frozeniceee/DorkX.git
cd DorkX

# 2. Build the binary
go build -o dorkx .

# 3. Move to your system bin path
sudo mv dorkx /usr/local/bin/
```

### Method 2: Direct Install with Go

```bash
go install github.com/frozeniceee/DorkX@latest
```

---

## 🚀 Quick Start & Usage

```bash
dorkx [flags]
```

### 1. Basic Single Dork (Collects 50 unique targets by default)
```bash
dorkx -d "inurl:php?id=" -o results.txt
```

### 2. Collect only Live URLs Responding with HTTP 200 OK
```bash
dorkx -d "inurl:index.php?id=" -status-code -mc 200 -limit 50 -o live_targets.txt
```

### 3. Collect an Exact Number of Targets (e.g. 100 targets)
```bash
dorkx -d "inurl:index.php?id=" -limit 100 -o targets.txt
```

### 4. Exclude Noise Domains & Keywords
```bash
dorkx -d "inurl:php?id= -github -stackoverflow -.gov.bd -demo" -limit 50 -o clean_targets.txt
```

### 5. Target Specific Country / TLD with 1 URL per Domain (`-u`)
```bash
dorkx -d "site:.gov.bd inurl:page.php?id=" -limit 30 -u -o gov_targets.txt
```

### 6. Pipe into `httpx` and `nuclei` (Bug Bounty Pipeline)
```bash
dorkx -d "inurl:index.php?id=" -limit 50 -silent | httpx -silent -mc 200 | nuclei -t sqli/
```

### 7. Bulk Dorking from File
```bash
dorkx -l dorks.txt -limit 50 -t 2 -o all_results.txt
```

---

## 📖 Flag Reference

| Flag | Type | Default | Description |
| :--- | :--- | :--- | :--- |
| **Input Options** | | | |
| `-d` | `string` | `""` | Single dork query string |
| `-l` | `string` | `""` | File containing dorks, one per line |
| `stdin` | `pipe` | `-` | Pipe dorks via standard input (`cat dorks.txt \| dorkx`) |
| **Collection & Filtering** | | | |
| `-limit` | `int` | `50` | Target number of unique strictly-matched URLs to collect |
| `-u` | `bool` | `false` | Unique domain mode: only output 1 URL per root domain |
| `-status-code`, `-sc` | `bool` | `false` | Live probe: verify and display HTTP status code (e.g. `[200]`) |
| `-mc` | `string` | `""` | Match status code filter (e.g. `-mc 200` or `-mc 200,301`) |
| `-t` | `int` | `1` | Number of concurrent dork searches |
| `-delay` | `string` | `"1-2"` | Delay range in seconds between batch requests |
| `-timeout` | `int` | `25` | HTTP request timeout in seconds |
| **Engine Selection** | | | |
| `-e` | `string` | `hybrid` | Search engine: `hybrid` (Yahoo+Bing), `yahoo`, `bing`, `google`, `google-api`, or `all` |
| `-api-key` | `string` | `""` | Google Custom Search API Key (for `google-api` engine) |
| `-cx` | `string` | `""` | Google Search Engine CX ID (for `google-api` engine) |
| **Proxy Options** | | | |
| `-proxy` | `string` | `""` | Single HTTP/SOCKS5 proxy (`socks5://127.0.0.1:9050` or `http://127.0.0.1:8080`) |
| `-proxy-list` | `string` | `""` | Path to file containing list of proxies for rotation |
| **Output & Formatting** | | | |
| `-o` | `string` | `""` | File path to save collected URLs |
| `-silent` | `bool` | `false` | Silent mode: outputs clean URLs only to stdout |
| `-nc` | `bool` | `false` | Disable colored terminal output |
| `-version` | `bool` | `false` | Print version and exit |

---

## 🎯 Supported Dork Operators

DorkX supports full search engine operator syntax and enforces them strictly on every returned candidate URL:

| Operator | Syntax Example | Meaning |
| :--- | :--- | :--- |
| **inurl:** | `inurl:php?id=` | URL path or query parameters must contain this pattern |
| **site:** | `site:.gov.bd` or `site:example.com` | Target URL domain must match or end with this domain/TLD |
| **filetype: / ext:** | `filetype:php` or `ext:sql` | Target URL must end with this file extension |
| **Negative Term (-)** | `-github -stackoverflow` | Exclude any URL containing these keywords |
| **Negative Site (-site:)** | `-site:facebook.com -site:.gov` | Exclude specific domains or TLDs |
| **Negative inurl (-inurl:)** | `-inurl:login -inurl:test` | Exclude specific path/parameter patterns |

### Combined Query Example:
```bash
dorkx -d "inurl:item.php?id= -site:amazon.com -site:ebay.com -stackoverflow filetype:php" -limit 50
```

---

## 🛠️ Recon Pipeline Workflows

### 1. Dorking $\rightarrow$ Live Host Probing with `httpx`
```bash
dorkx -d "inurl:page.php?id=" -limit 50 -silent | httpx -silent -status-code -title
```

### 2. Dorking $\rightarrow$ Vulnerability Scanning with `nuclei`
```bash
dorkx -d "inurl:index.php?id=" -limit 50 -silent | httpx -silent -mc 200 | nuclei -tags sqli -o sqli_vulnerable.txt
```

### 3. Save Targets and Pipe Simultaneously with `tee`
```bash
dorkx -d "inurl:view.php?id=" -limit 100 -silent | tee targets.txt | httpx -silent
```

---

## 📄 Dork List File Format (`-l`)

Create a text file (e.g. `dorks.txt`) with one dork per line. Lines starting with `#` are ignored as comments:

```text
# SQL Injection Dorks
inurl:index.php?id= -github
inurl:page.php?id= -stackoverflow
inurl:product.php?id=

# LFI / Path Traversal Dorks
inurl:include.php?file=
inurl:load.php?page=

# Specific TLD Targeting
site:.gov.bd inurl:page.php?id=
site:.edu inurl:index.php?id=
```

Run in bulk:
```bash
dorkx -l dorks.txt -limit 30 -o bulk_targets.txt
```

---

## 🤝 Contributing

Contributions, issues, and feature requests are welcome! Feel free to check the [issues page](https://github.com/frozeniceee/DorkX/issues).

---

## 📜 License

This project is licensed under the **MIT License** — see the [LICENSE](LICENSE) file for details.

---

<p align="center">
  Made with ❤️ by <b>Fr0z3n_1c3</b>
</p>
