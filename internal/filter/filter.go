package filter

import (
	"net/url"
	"regexp"
	"strings"
)

// excludedDomains are domains we always discard (search engines, portals, social media, etc.)
var excludedDomains = []string{
	"google.com", "google.co", "googleapis.com", "gstatic.com",
	"googleusercontent.com", "youtube.com", "youtu.be",
	"facebook.com", "twitter.com", "instagram.com", "linkedin.com",
	"pinterest.com", "reddit.com", "wikipedia.org", "wikimedia.org",
	"amazon.com", "ebay.com", "bing.com", "yahoo.com", "yimg.com",
	"w3.org", "schema.org", "archive.org", "github.com",
	"maps.google", "accounts.google", "policies.google",
	"support.google", "translate.google", "books.google",
	"drive.google", "docs.google", "play.google",
}

// excludedExtensions are static asset/binary file types we never want
var excludedExtensions = []string{
	".jpg", ".jpeg", ".png", ".gif", ".bmp", ".svg", ".webp", ".ico",
	".mp4", ".mp3", ".avi", ".mkv", ".mov", ".webm",
	".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx",
	".zip", ".rar", ".tar", ".gz", ".7z",
	".exe", ".dmg", ".pkg", ".deb", ".rpm",
	".css", ".woff", ".woff2", ".ttf", ".eot",
}

// InURLRequirement stores path and parameter constraints
type InURLRequirement struct {
	RawPattern string
	PathPart   string // e.g. "php" or "page.php"
	ParamPart  string // e.g. "id=" or "id"
}

// DorkRule holds the strict constraints extracted from a dork query
type DorkRule struct {
	InURLReqs    []InURLRequirement
	SiteSuffixes []string
	RawDork      string
}

// ExtractRulesFromDork parses a dork query string and extracts smart strict matching rules.
func ExtractRulesFromDork(dork string) *DorkRule {
	rule := &DorkRule{RawDork: dork}

	// Extract inurl: patterns
	inurlRe := regexp.MustCompile(`(?i)inurl:["']?([^\s"']+)["']?`)
	for _, match := range inurlRe.FindAllStringSubmatch(dork, -1) {
		if len(match) > 1 {
			rawPattern := strings.ToLower(strings.Trim(match[1], `"'`))
			if rawPattern == "" {
				continue
			}

			req := InURLRequirement{RawPattern: rawPattern}
			if strings.Contains(rawPattern, "?") {
				parts := strings.SplitN(rawPattern, "?", 2)
				req.PathPart = parts[0]
				req.ParamPart = parts[1]
			}
			rule.InURLReqs = append(rule.InURLReqs, req)
		}
	}

	// Extract site: constraints
	siteRe := regexp.MustCompile(`(?i)site:["']?([^\s"']+)["']?`)
	for _, match := range siteRe.FindAllStringSubmatch(dork, -1) {
		if len(match) > 1 {
			site := strings.ToLower(strings.Trim(match[1], `"'`))
			if site != "" {
				rule.SiteSuffixes = append(rule.SiteSuffixes, site)
			}
		}
	}

	return rule
}

// IsValidURL returns true if the URL satisfies all strict dork rules.
func IsValidURL(rawURL string, rule *DorkRule) bool {
	if rawURL == "" {
		return false
	}

	// Must be absolute HTTP(S)
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		return false
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}

	host := strings.ToLower(parsed.Hostname())
	path := strings.ToLower(parsed.Path)
	query := strings.ToLower(parsed.RawQuery)
	fullURL := strings.ToLower(rawURL)

	// 1. Reject known garbage/portal domains
	for _, excluded := range excludedDomains {
		if strings.Contains(host, excluded) {
			return false
		}
	}

	// 2. Reject static asset extensions
	for _, ext := range excludedExtensions {
		if strings.HasSuffix(path, ext) {
			return false
		}
	}

	// 3. Reject search engine loopbacks
	if strings.Contains(fullURL, "/url?q=") ||
		strings.Contains(fullURL, "/search?") ||
		strings.Contains(fullURL, "search.yahoo") ||
		strings.Contains(fullURL, "bing.com") {
		return false
	}

	// 4. Enforce site: constraint
	for _, suffix := range rule.SiteSuffixes {
		if strings.HasPrefix(suffix, ".") {
			if !strings.HasSuffix(host, suffix) {
				return false
			}
		} else {
			if host != suffix && !strings.HasSuffix(host, "."+suffix) {
				return false
			}
		}
	}

	// 5. Enforce inurl: constraint
	for _, req := range rule.InURLReqs {
		pathAndQuery := path
		if query != "" {
			pathAndQuery += "?" + query
		}

		// Direct substring check
		if strings.Contains(pathAndQuery, req.RawPattern) {
			continue
		}

		// Multi-part check (e.g. php?id= matched across path + query params)
		if req.PathPart != "" && req.ParamPart != "" {
			// Path must contain pathPart (e.g. .php or php)
			pathMatches := strings.Contains(path, req.PathPart)
			// Query must contain paramPart (e.g. id= or id)
			queryMatches := strings.Contains(query, req.ParamPart) || strings.Contains(query, strings.TrimSuffix(req.ParamPart, "=")+"=")

			if pathMatches && queryMatches {
				continue
			}
		}

		return false
	}

	return true
}

// ExtractRealURL unwraps search engine redirects (Google, Yahoo, Bing, etc.)
func ExtractRealURL(raw string) string {
	if strings.Contains(raw, "/url?q=") {
		parts := strings.SplitN(raw, "/url?q=", 2)
		if len(parts) == 2 {
			target := strings.SplitN(parts[1], "&", 2)[0]
			return unescapeURL(target)
		}
	}
	return unescapeURL(raw)
}

func unescapeURL(target string) string {
	target = strings.ReplaceAll(target, "%3A", ":")
	target = strings.ReplaceAll(target, "%3a", ":")
	target = strings.ReplaceAll(target, "%2F", "/")
	target = strings.ReplaceAll(target, "%2f", "/")
	target = strings.ReplaceAll(target, "%3F", "?")
	target = strings.ReplaceAll(target, "%3f", "?")
	target = strings.ReplaceAll(target, "%3D", "=")
	target = strings.ReplaceAll(target, "%3d", "=")
	target = strings.ReplaceAll(target, "%26", "&")
	return target
}

// RootDomain extracts the root domain (host) from a URL for deduplication.
func RootDomain(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	host := strings.ToLower(parsed.Hostname())
	return strings.TrimPrefix(host, "www.")
}
