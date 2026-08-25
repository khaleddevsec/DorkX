package filter

import (
	"net/url"
	"regexp"
	"strings"
)

// excludedDomains are domains we always discard (Google internals, social media, etc.)
var excludedDomains = []string{
	"google.com", "google.co", "googleapis.com", "gstatic.com",
	"googleusercontent.com", "youtube.com", "youtu.be",
	"facebook.com", "twitter.com", "instagram.com", "linkedin.com",
	"pinterest.com", "reddit.com", "wikipedia.org", "wikimedia.org",
	"amazon.com", "ebay.com", "bing.com", "yahoo.com",
	"w3.org", "schema.org", "archive.org",
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

// DorkRule holds the strict constraints extracted from a dork query
type DorkRule struct {
	// InURLPatterns are strings that MUST appear in the URL path+query
	InURLPatterns []string
	// SiteSuffixes are domain suffixes like .gov.bd, .edu that the host must end with
	SiteSuffixes []string
	// HasParams means the URL must contain at least one query parameter
	HasParams bool
}

// ExtractRulesFromDork parses a dork query string and extracts strict matching rules.
// Example: "site:.gov.bd inurl:page.php?id=" =>
//   InURLPatterns: ["page.php?id="]
//   SiteSuffixes:  [".gov.bd"]
func ExtractRulesFromDork(dork string) *DorkRule {
	rule := &DorkRule{}

	// Extract inurl: patterns
	inurlRe := regexp.MustCompile(`(?i)inurl:["']?([^\s"']+)["']?`)
	for _, match := range inurlRe.FindAllStringSubmatch(dork, -1) {
		if len(match) > 1 {
			pattern := strings.Trim(match[1], `"'`)
			if pattern != "" {
				rule.InURLPatterns = append(rule.InURLPatterns, strings.ToLower(pattern))
				// If any inurl pattern contains "?", that means we need query params
				if strings.Contains(pattern, "?") || strings.Contains(pattern, "=") {
					rule.HasParams = true
				}
			}
		}
	}

	// Extract site: constraints
	siteRe := regexp.MustCompile(`(?i)site:["']?([^\s"']+)["']?`)
	for _, match := range siteRe.FindAllStringSubmatch(dork, -1) {
		if len(match) > 1 {
			site := strings.Trim(match[1], `"'`)
			if site != "" {
				rule.SiteSuffixes = append(rule.SiteSuffixes, strings.ToLower(site))
			}
		}
	}

	return rule
}

// IsValidURL returns true if the URL passes all strict dork rules.
// This is the core filter. Any URL that doesn't satisfy ALL rules is discarded.
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
	fullURL := strings.ToLower(rawURL)

	// 1. Reject known garbage domains
	for _, excluded := range excludedDomains {
		if strings.Contains(host, excluded) {
			return false
		}
	}

	// 2. Reject static asset extensions
	path := strings.ToLower(parsed.Path)
	for _, ext := range excludedExtensions {
		if strings.HasSuffix(path, ext) {
			return false
		}
	}

	// 3. Reject Google tracking/redirect URLs
	if strings.Contains(fullURL, "/url?q=") || strings.Contains(fullURL, "google.com/search") {
		return false
	}

	// 4. Enforce site: constraint — host must end with or equal the site pattern
	for _, suffix := range rule.SiteSuffixes {
		// site:example.com means exact host match
		// site:.gov.bd means host must end with .gov.bd
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

	// 5. Enforce inurl: pattern — the pattern must appear in the path+query of the URL
	//    We reconstruct path+query so we check where the pattern sits, NOT in the domain
	pathAndQuery := strings.ToLower(parsed.Path)
	if parsed.RawQuery != "" {
		pathAndQuery += "?" + strings.ToLower(parsed.RawQuery)
	}

	for _, pattern := range rule.InURLPatterns {
		if !strings.Contains(pathAndQuery, pattern) {
			return false
		}
	}

	return true
}

// ExtractRealURL unwraps Google redirect URLs like /url?q=https://target.com/...
func ExtractRealURL(raw string) string {
	if strings.Contains(raw, "/url?q=") {
		parts := strings.SplitN(raw, "/url?q=", 2)
		if len(parts) == 2 {
			// Strip everything after & (Google tracking params)
			target := strings.SplitN(parts[1], "&", 2)[0]
			// URL-decode basic escapes
			target = strings.ReplaceAll(target, "%3A", ":")
			target = strings.ReplaceAll(target, "%2F", "/")
			target = strings.ReplaceAll(target, "%3F", "?")
			target = strings.ReplaceAll(target, "%3D", "=")
			target = strings.ReplaceAll(target, "%26", "&")
			return target
		}
	}
	return raw
}

// RootDomain extracts the root domain (host) from a URL for deduplication.
func RootDomain(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	host := strings.ToLower(parsed.Hostname())
	// Strip www.
	host = strings.TrimPrefix(host, "www.")
	return host
}
