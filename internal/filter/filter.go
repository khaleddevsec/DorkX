package filter

import (
	"net/url"
	"strings"
)

// defaultExcludedDomains are generic portals we always discard unless explicitly asked by the user
var defaultExcludedDomains = []string{
	"google.com", "google.co", "googleapis.com", "gstatic.com",
	"googleusercontent.com", "youtube.com", "youtu.be",
	"facebook.com", "twitter.com", "x.com", "instagram.com", "linkedin.com",
	"pinterest.com", "reddit.com", "wikipedia.org", "wikimedia.org",
	"amazon.com", "ebay.com", "bing.com", "yahoo.com", "yimg.com",
	"duckduckgo.com", "aol.com", "w3.org", "schema.org", "archive.org",
	"maps.google", "accounts.google", "policies.google",
	"support.google", "translate.google", "books.google",
}

// defaultExcludedExtensions are static browser assets & media only (NEVER data/docs/logs)
var defaultExcludedExtensions = []string{
	".jpg", ".jpeg", ".png", ".gif", ".bmp", ".svg", ".webp", ".ico",
	".mp4", ".mp3", ".avi", ".mkv", ".mov", ".webm",
	".css", ".woff", ".woff2", ".ttf", ".eot",
}

// InURLRequirement stores path and parameter constraints
type InURLRequirement struct {
	RawPattern string
	PathPart   string // e.g. "php" or "page.php" or "/admin/"
	ParamPart  string // e.g. "id=" or "id"
}

// DorkRule holds the complete set of strict positive and negative constraints
type DorkRule struct {
	InURLReqs      []InURLRequirement
	SiteSuffixes   []string
	FileTypes      []string
	ExcludedTerms  []string // e.g. -github, -stackoverflow, -.gov.bd
	ExcludedSites  []string // e.g. -site:github.com, -site:.gov.bd
	ExcludedInURLs []string // e.g. -inurl:test, -inurl:demo
	RawDork        string
}

// ExtractRulesFromDork parses any search engine dork query with all operators.
func ExtractRulesFromDork(dork string) *DorkRule {
	rule := &DorkRule{RawDork: dork}

	tokens := parseTokens(dork)
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

// GenerateClusterDorks expands a dork into smart clusterbomb permutations compatible with search engines
func GenerateClusterDorks(dork string) []string {
	var cluster []string
	dork = strings.TrimSpace(dork)
	dorkLower := strings.ToLower(dork)

	if strings.Contains(dorkLower, "inurl:") {
		tokens := parseTokens(dork)
		var nonInurls []string
		var inurlPatterns []string

		for _, t := range tokens {
			if strings.HasPrefix(strings.ToLower(t), "inurl:") {
				val := strings.Trim(strings.TrimPrefix(t, "inurl:"), `"'`)
				inurlPatterns = append(inurlPatterns, val)
			} else {
				nonInurls = append(nonInurls, t)
			}
		}

		suffix := strings.Join(nonInurls, " ")

		for _, pat := range inurlPatterns {
			if strings.Contains(pat, "/") {
				segments := strings.Split(pat, "/")
				var validSegs []string
				for _, s := range segments {
					s = strings.TrimSpace(s)
					if s != "" {
						validSegs = append(validSegs, s)
					}
				}

				if len(validSegs) >= 2 {
					var inurlCombo []string
					for _, s := range validSegs {
						inurlCombo = append(inurlCombo, "inurl:"+s)
					}
					cluster = append(cluster, strings.TrimSpace(strings.Join(inurlCombo, " ")+" "+suffix))
					cluster = append(cluster, strings.TrimSpace("inurl:"+validSegs[0]+" \""+validSegs[len(validSegs)-1]+"\" "+suffix))
					cluster = append(cluster, strings.TrimSpace("\""+pat+"\" "+suffix))

					if strings.Contains(pat, "admin") && strings.Contains(pat, "login") {
						cluster = append(cluster,
							strings.TrimSpace("inurl:admin inurl:index.php "+suffix),
							strings.TrimSpace("inurl:index.php inurl:admin "+suffix),
							strings.TrimSpace("inurl:login.php inurl:admin "+suffix),
						)
					}
				}
			}

			if strings.Contains(pat, "php?id=") || strings.Contains(pat, ".php?id=") {
				cluster = append(cluster,
					strings.TrimSpace("inurl:index.php?id= "+suffix),
					strings.TrimSpace("inurl:page.php?id= "+suffix),
					strings.TrimSpace("inurl:view.php?id= "+suffix),
					strings.TrimSpace("inurl:item.php?id= "+suffix),
					strings.TrimSpace("inurl:article.php?id= "+suffix),
				)
			}
		}
	}

	cluster = append(cluster, dork)

	// Deduplicate cluster
	seen := make(map[string]bool)
	var uniqueCluster []string
	for _, d := range cluster {
		d = strings.TrimSpace(d)
		if d != "" && !seen[d] {
			seen[d] = true
			uniqueCluster = append(uniqueCluster, d)
		}
	}

	return uniqueCluster
}

// parseTokens splits dork queries preserving quoted strings ("index of", "admin login")
func parseTokens(query string) []string {
	var tokens []string
	var current strings.Builder
	inQuotes := false

	for i := 0; i < len(query); i++ {
		ch := query[i]
		if ch == '"' || ch == '\'' {
			inQuotes = !inQuotes
			current.WriteByte(ch)
		} else if (ch == ' ' || ch == '\t') && !inQuotes {
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
		} else {
			current.WriteByte(ch)
		}
	}

	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens
}

// IsValidURL checks a URL against all positive and negative dork rules with strict semantic accuracy
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

	// --- 1. DEFAULT PORTAL & ENGINE EXCLUSIONS ---
	for _, excluded := range defaultExcludedDomains {
		if strings.Contains(host, excluded) {
			explicitlyTargeted := false
			for _, site := range rule.SiteSuffixes {
				if strings.Contains(site, excluded) {
					explicitlyTargeted = true
					break
				}
			}
			if !explicitlyTargeted {
				return false
			}
		}
	}

	for _, ext := range defaultExcludedExtensions {
		if strings.HasSuffix(path, ext) {
			explicitlyRequested := false
			for _, ft := range rule.FileTypes {
				if ft == ext {
					explicitlyRequested = true
					break
				}
			}
			if !explicitlyRequested {
				return false
			}
		}
	}

	if strings.Contains(fullURL, "/url?q=") ||
		strings.Contains(fullURL, "/search?") ||
		strings.Contains(fullURL, "search.yahoo") ||
		strings.Contains(fullURL, "bing.com/ck/") ||
		strings.Contains(fullURL, "duckduckgo.com/l/") {
		return false
	}

	// --- 2. USER NEGATIVE EXCLUSIONS (-) ---
	for _, term := range rule.ExcludedTerms {
		if strings.Contains(fullURL, term) || strings.Contains(host, term) {
			return false
		}
	}

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

	for _, exInurl := range rule.ExcludedInURLs {
		if strings.Contains(path, exInurl) || strings.Contains(query, exInurl) {
			return false
		}
	}

	// --- 3. POSITIVE CONSTRAINTS ---

	// Enforce filetype: / ext:
	if len(rule.FileTypes) > 0 {
		matchedFileType := false
		for _, ext := range rule.FileTypes {
			if strings.HasSuffix(path, ext) || strings.Contains(path, ext+"?") || strings.Contains(path, ext+"/") || strings.Contains(path, ext) {
				matchedFileType = true
				break
			}
		}
		if !matchedFileType {
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

	// Enforce inurl: constraint strictly
	for _, req := range rule.InURLReqs {
		pathAndQuery := path
		if query != "" {
			pathAndQuery += "?" + query
		}

		if !matchesInURLRequirement(pathAndQuery, path, query, host, req) {
			return false
		}
	}

	return true
}

func matchesInURLRequirement(pathAndQuery, path, query, host string, req InURLRequirement) bool {
	// 1. Direct exact substring match (e.g. /admin/login.php or php?id=)
	if strings.Contains(pathAndQuery, req.RawPattern) {
		return true
	}

	// 2. Multi-part query parameter match (e.g. php?id= or page.php?id=)
	if req.PathPart != "" && req.ParamPart != "" {
		pathMatches := strings.Contains(path, req.PathPart)
		queryMatches := strings.Contains(query, req.ParamPart) || strings.Contains(query, strings.TrimSuffix(req.ParamPart, "=")+"=")
		if pathMatches && queryMatches {
			return true
		}
	}

	// 3. Multi-segment inurl matching (e.g. admin/login.php or admin/index.php)
	if strings.Contains(req.RawPattern, "/") {
		// If dork specifies a file extension like .php or .asp
		requiredExt := ""
		if strings.Contains(req.RawPattern, ".") {
			parts := strings.Split(req.RawPattern, ".")
			requiredExt = "." + parts[len(parts)-1]
		}

		// If dork explicitly requires an extension (.php), strictly reject non-matching extensions (.aspx, .htm, .html)
		if requiredExt != "" && !strings.Contains(path, requiredExt) && !strings.Contains(query, requiredExt) {
			return false
		}

		// Intent recognition for admin and login portals (must have requiredExt)
		if strings.Contains(req.RawPattern, "admin") && (strings.Contains(req.RawPattern, "login") || strings.Contains(req.RawPattern, "index")) {
			if strings.Contains(pathAndQuery, "admin") && (strings.Contains(pathAndQuery, "login") || strings.Contains(pathAndQuery, "index") || strings.Contains(pathAndQuery, "php")) {
				return true
			}
			if strings.Contains(pathAndQuery, "login.php") || strings.Contains(pathAndQuery, "index.php/admin") {
				return true
			}
		}

		// Check presence of all core path keywords
		segments := strings.Split(req.RawPattern, "/")
		allSegmentsPresent := true
		for _, seg := range segments {
			seg = strings.TrimSpace(seg)
			if seg == "" {
				continue
			}
			baseName := seg
			if requiredExt != "" {
				baseName = strings.TrimSuffix(seg, requiredExt)
			}
			if !strings.Contains(pathAndQuery, seg) && !strings.Contains(pathAndQuery, baseName) {
				allSegmentsPresent = false
				break
			}
		}

		if allSegmentsPresent {
			return true
		}
	}

	return false
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
	if strings.Contains(raw, "uddg=") {
		parts := strings.SplitN(raw, "uddg=", 2)
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
