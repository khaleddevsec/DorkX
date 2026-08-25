# DorkX

> **Fast, strict Google dorking CLI tool for Linux — built for bug bounty recon pipelines.**
> By **Fr0z3n_1c3** | [github.com/frozeniceee/DorkX](https://github.com/frozeniceee/DorkX)

---

## What is dorkx?

`dorkx` is a CLI Google dorking tool designed to work exactly like `httpx`, `nuclei`, and `subfinder` from ProjectDiscovery.

Key design principles:
- **Strict URL matching** — if you search `inurl:php?id=`, you *only* get URLs that actually contain `php?id=` in the path/query. Zero garbage.
- **Zero duplicates** — same URL or same domain never appears twice.
- **Pipeline-ready** — all matched URLs go to `stdout`, all logs go to `stderr`, so you can pipe directly into `httpx`, `nuclei`, or `tee`.
- **No heavy dependencies** — single compiled binary, no Python, no browser drivers, no Electron.

---

## Install

### Option 1: Build from source (requires Go 1.21+)

```bash
git clone https://github.com/frozeniceee/DorkX.git
cd DorkX
go build -o dorkx .
sudo mv dorkx /usr/local/bin/
```

### Option 2: Direct install with `go install`

```bash
go install github.com/frozeniceee/DorkX@latest
```

---

## Usage

```
dorkx [flags]
```

### Input Flags

| Flag | Description |
|------|-------------|
| `-d "query"` | Single dork query |
| `-l file.txt` | File with dorks, one per line |
| `stdin` | Pipe dorks via stdin (see examples) |

### Engine Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-e name` | `hybrid` | Search engine: `hybrid` (default: Yahoo + Bing parallel), `yahoo`, `bing`, `google`, `google-api`, or `all` |
| `-api-key key` | `""` | Google Custom Search API Key (for `google-api` engine) |
| `-cx id` | `""` | Google Search Engine CX ID (for `google-api` engine) |

### Collection & Filter Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-limit int` | `50` | Target number of unique strictly-matched URLs to collect |
| `-u` | `false` | Only output 1 URL per unique root domain |
| `-t int` | `1` | Concurrent dork searches |
| `-delay "min-max"` | `"1-2"` | Random delay between requests (seconds) |
| `-timeout int` | `25` | HTTP timeout per request in seconds |

### Proxy Flags

| Flag | Description |
|------|-------------|
| `-proxy url` | Single HTTP/SOCKS5 proxy (`socks5://127.0.0.1:9050` or `http://127.0.0.1:8080`) |
| `-proxy-list file` | File with proxy list (round-robin rotation) |

### Output Flags

| Flag | Description |
|------|-------------|
| `-o file.txt` | Save matched URLs to file |
| `-silent` | Only print URLs (no banner/logs) — best for piping |
| `-nc` | Disable colored output |
| `-version` | Print version and exit |

---

## Examples

### Collect 50 Unique Targets Instantly (Default)

```bash
dorkx -d "inurl:php?id=" -o results.txt
```

### Collect Exact Target Count (e.g. 100 targets)

```bash
dorkx -d "inurl:index.php?id=" -limit 100 -o targets.txt
```

### Silent Mode — Pipe to `httpx` & `nuclei`

```bash
dorkx -d "inurl:page.php?id=" -limit 50 -silent | httpx -silent -mc 200 | nuclei -t sqli/
```

### Target Specific Country / TLD (.gov.bd) with Unique Domains

```bash
dorkx -d "site:.gov.bd inurl:page.php?id=" -limit 30 -u -o gov_targets.txt
```

### Bulk Dorks from File

```bash
dorkx -l dorks.txt -limit 50 -o all_results.txt
```

### Pipe dorks from stdin

```bash
cat dorks.txt | dorkx -silent > targets.txt
```

### Full recon pipeline (httpx + nuclei)

```bash
dorkx -d "inurl:index.php?id=" -silent | httpx -silent -mc 200 | nuclei -t sqli/
```

### With proxy for anonymity

```bash
dorkx -d "inurl:php?id=" -proxy http://127.0.0.1:8080 -p 5
```

### With proxy rotation list

```bash
dorkx -l dorks.txt -proxy-list proxies.txt -delay 3-8 -silent | tee targets.txt
```

### Slower, stealthy scraping with higher delay

```bash
dorkx -d "inurl:php?id=" -p 10 -delay 5-12 -o results.txt
```

---

## Strict Matching — How It Works

When you run:

```bash
dorkx -d "inurl:php?id="
```

dorkx extracts the pattern `php?id=` and checks **every candidate URL** from Google against it.

| URL | Result |
|-----|--------|
| `https://example.com/page.php?id=13` | ✅ MATCH |
| `http://csi-india.org/news/index.php?id=18` | ✅ MATCH |
| `https://bethaneyschool.com/gallery.php?id=2` | ✅ MATCH |
| `https://example.com/about-us` | ❌ DISCARDED |
| `https://google.com/search?q=php?id=` | ❌ DISCARDED (Google internal) |
| `https://example.com/admin` (duplicate domain) | ❌ DISCARDED (with `-u` flag) |

---

## Dork Syntax Reference

`dorkx` understands standard Google dork operators:

| Operator | Example | Effect |
|----------|---------|--------|
| `inurl:` | `inurl:admin.php?id=` | URL must contain this pattern in path/query |
| `site:` | `site:.gov.bd` | Domain must match this suffix |
| Combined | `site:.edu inurl:login.php?user=` | Both constraints must be satisfied |

---

## Tips for Best Results

1. **Use `-delay 3-8`** on large runs to avoid Google rate limits.
2. **Use `-proxy-list proxies.txt`** with residential proxies for large-scale recon.
3. **Use `-u`** to get maximum domain coverage instead of multiple URLs from the same site.
4. **Keep `-p` between 3–10**. Beyond page 10, Google usually stops returning relevant results anyway.
5. **Always use `-silent`** when piping to other tools.

---

## Dorks File Format (`-l`)

One dork per line. Lines starting with `#` are ignored (comments).

```
# SQL injection parameter hunting
inurl:index.php?id=
inurl:page.php?id=
inurl:view.php?id=
inurl:item.php?id=

# Admin panels
inurl:admin/login.php
inurl:administrator/index.php

# Specific TLD
site:.gov.bd inurl:page.php?id=
```

---

## Output Format

All matched URLs are printed **one per line** to `stdout` — compatible with any UNIX pipeline tool.

```
https://www.rangamaticollege.gov.bd/page.php?id=13
http://csi-india.org/news/index.php?id=18
https://bethaneyschoolsulantu.com/gallery.php?id=2
```

---

## License

MIT License — free to use, modify, and distribute.
