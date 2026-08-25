package filter

import (
	"net/url"
	"strings"
)

// defaultExcludedDomains are generic portals we always discard unless explicitly asked
var defaultExcludedDomains = []string{
	"google.com", "google.co", "googleapis.com", "gstatic.com",
	"googleusercontent.com", "youtube.com", "youtu.be",
	"facebook.com", "twitter.com", "x.com", "instagram.com", "linkedin.com",
	"pinterest.com", "reddit.com", "wikipedia.org", "wikimedia.org",
	"amazon.com", "ebay.com", "bing.com", "yahoo.com", "yimg.com",
	"w3.org", "schema.org", "archive.org",
	"maps.google", "accounts.google", "policies.google",
	"support.google", "translate.google", "books.google",
}

// defaultExcludedExtensions are static asset/binary file types
var defaultExcludedExtensions = []string{
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

// DorkRule holds the complete set of strict positive and negative constraints
type DorkRule struct {
	InURLReqs       []InURLRequirement
	SiteSuffixes    []string
	FileTypes       []string
	ExcludedTerms   []string // e.g. -github, -stackoverflow, -.gov.bd
	ExcludedSites   []string // e.g. -site:github.com, -site:.gov.bd
	ExcludedInURLs  []string // e.g. -inurl:test, -inurl:demo
	RawDork         string
}

// ExtractRulesFromDork parses complex dork queries with positive and negative operators.
// Example: "inurl:php?id= -github -stackoverflow -.gov.bd filetype:php -site:demo.com"
func ExtractRulesFromDork(dork string) *DorkRule {
	rule := &DorkRule{RawDork: dork}

	tokens := strings.Fields(dork)
	for _, token := range tokens {
		tLower := strings.ToLower(token)

		// 1. Negative inurl: (-inurl:XYZ)
		if strings.HasPrefix(tLower, "-inurl:") {
			val := strings.Trim(strings.TrimPrefix(token, "-inurl:"), `"'`)
			if val != "" {
				rule.ExcludedInURLs = append(rule.ExcludedInURLs, strings.ToLower(val))
			}
			continue
		}

		// 2. Negative site: (-site:XYZ)
		if strings.HasPrefix(tLower, "-site:") {
			val := strings.Trim(strings.TrimPrefix(token, "-site:"), `"'`)
			if val != "" {
				rule.ExcludedSites = append(rule.ExcludedSites, strings.ToLower(val))
			}
			continue
		}

		// 3. Generic Negative Exclusions (-XYZ e.g. -github, -stackoverflow, -.gov.bd)
		if strings.HasPrefix(token, "-") && len(token) > 1 && !strings.Contains(token, ":") {
			val := strings.Trim(strings.TrimPrefix(token, "-"), `"'`)
			if val != "" {
				rule.ExcludedTerms = append(rule.ExcludedTerms, strings.ToLower(val))
			}
			continue
		}

		// 4. Positive filetype: / ext:
		if strings.HasPrefix(tLower, "filetype:") || strings.HasPrefix(tLower, "ext:") {
			val := token
			if strings.HasPrefix(tLower, "filetype:") {
				val = strings.TrimPrefix(token, "filetype:")
			} else {
				val = strings.TrimPrefix(token, "ext:")
			}
			val = strings.Trim(val, `"'`)
			if val != "" {
				if !strings.HasPrefix(val, ".") {
					val = "." + val
				}
				rule.FileTypes = append(rule.FileTypes, strings.ToLower(val))
			}
			continue
		}

		// 5. Positive inurl:
		if strings.HasPrefix(tLower, "inurl:") {
			rawPattern := strings.ToLower(strings.Trim(strings.TrimPrefix(token, "inurl:"), `"'`))
			if rawPattern != "" {
				req := InURLRequirement{RawPattern: rawPattern}
				if strings.Contains(rawPattern, "?") {
					parts := strings.SplitN(rawPattern, "?", 2)
					req.PathPart = parts[0]
					req.ParamPart = parts[1]
				}
				rule.InURLReqs = append(rule.InURLReqs, req)
			}
			continue
		}

		// 6. Positive site:
		if strings.HasPrefix(tLower, "site:") {
			site := strings.ToLower(strings.Trim(strings.TrimPrefix(token, "site:"), `"'`))
			if site != "" {
				rule.SiteSuffixes = append(rule.SiteSuffixes, site)
			}
			continue
		}
	}

	return rule
}

// IsValidURL checks a URL against all positive and negative dork rules
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

	// --- 1. DEFAULT EXCLUSIONS ---
	for _, excluded := range defaultExcludedDomains {
		if strings.Contains(host, excluded) {
			return false
		}
	}

	for _, ext := range defaultExcludedExtensions {
		if strings.HasSuffix(path, ext) {
			return false
		}
	}

	if strings.Contains(fullURL, "/url?q=") ||
		strings.Contains(fullURL, "/search?") ||
		strings.Contains(fullURL, "search.yahoo") ||
		strings.Contains(fullURL, "bing.com") {
		return false
	}

	// --- 2. USER NEGATIVE EXCLUSIONS (-) ---

	// Check generic excluded terms (-github, -stackoverflow, -.gov.bd)
	for _, term := range rule.ExcludedTerms {
		if strings.Contains(fullURL, term) || strings.Contains(host, term) {
			return false
		}
	}

	// Check excluded sites (-site:XYZ)
	for _, exSite := range rule.ExcludedSites {
		if strings.HasPrefix(exSite, ".") {
			if strings.HasSuffix(host, exSite) {
				return false
			}
		} else {
			if host == exSite || strings.HasSuffix(host, "."+exSite) {
				return false
			}
		}
	}

	// Check excluded inurl patterns (-inurl:XYZ)
	for _, exInurl := range rule.ExcludedInURLs {
		if strings.Contains(path, exInurl) || strings.Contains(query, exInurl) {
			return false
		}
	}

	// --- 3. POSITIVE CONSTRAINTS ---

	// Enforce filetype: / ext:
	for _, ext := range rule.FileTypes {
		if !strings.HasSuffix(path, ext) && !strings.Contains(path, ext) {
			return false
		}
	}

	// Enforce site: constraint
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

	// Enforce inurl: constraint
	for _, req := range rule.InURLReqs {
		pathAndQuery := path
		if query != "" {
			pathAndQuery += "?" + query
		}

		if strings.Contains(pathAndQuery, req.RawPattern) {
			continue
		}

		if req.PathPart != "" && req.ParamPart != "" {
			pathMatches := strings.Contains(path, req.PathPart)
			queryMatches := strings.Contains(query, req.ParamPart) || strings.Contains(query, strings.TrimSuffix(req.ParamPart, "=")+"=")

			if pathMatches && queryMatches {
				continue
			}
		}

		return false
	}

	return true
}

// ExtractRealURL unwraps search engine redirects
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

// RootDomain extracts the root domain from a URL for deduplication
func RootDomain(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	host := strings.ToLower(parsed.Hostname())
	return strings.TrimPrefix(host, "www.")
}
