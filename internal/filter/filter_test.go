package filter

import (
	"testing"
)

func TestGarbageFiltering(t *testing.T) {
	dork := "inurl:admin/login.php"
	rule := ExtractRulesFromDork(dork)

	garbageURLs := []string{
		"https://admin.cloud.microsoft/Adminportal/Home?source=applauncher",
		"https://login.gov/admin/",
		"https://api.data.gov/admin/login",
		"https://www.firstrespondertraining.gov/frts/admin",
		"https://www.nps.gov/commonspot/help/content/administrators_reference/administrator_login.htm",
		"https://property.uintacountywy.gov/AdminLogin.aspx",
		"https://dhhodisha.in/index.php/admin",
		"http://carboncalculator.americanpistachios.org/admin/index.php",
	}

	for _, u := range garbageURLs {
		if IsValidURL(u, rule) {
			t.Errorf("Garbage URL [%s] should have been REJECTED for dork [%s]", u, dork)
		}
	}

	validURLs := []string{
		"https://www.example.com/admin/login.php",
		"https://portal.site.org/app/admin/login.php?redirect=1",
		"https://www.bamesschool.com.ng/secondary/admin/login.php",
		"https://www.mithras-europe.eu/admin/login.php",
		"https://www.hamch.in/admin/login.php",
	}

	for _, u := range validURLs {
		if !IsValidURL(u, rule) {
			t.Errorf("Valid URL [%s] should have been ACCEPTED for dork [%s]", u, dork)
		}
	}
}

func TestPhpIdFiltering(t *testing.T) {
	dork := "inurl:php?id="
	rule := ExtractRulesFromDork(dork)

	validURLs := []string{
		"https://www.post.japanpost.jp/cgi-shiten_search/shiten.php?id=5210",
		"https://www.wtcpu.org.in/page.php?id=1001",
		"https://target.com/news/page.php?cat=articles&id=105",
	}

	for _, u := range validURLs {
		if !IsValidURL(u, rule) {
			t.Errorf("Valid URL [%s] should have been ACCEPTED for dork [%s]", u, dork)
		}
	}

	garbageURLs := []string{
		"https://sjcem.edu.in/aiml-eng/wp-content/uploads/sites/2/2026/03/BOS-Syllbus-AIML-29-11-25-.pdf",
		"https://www.gravitysplash.com/b/regenesis-roofing-restoration-2/",
		"https://example.com/page.html",
	}

	for _, u := range garbageURLs {
		if IsValidURL(u, rule) {
			t.Errorf("Garbage URL [%s] should have been REJECTED for dork [%s]", u, dork)
		}
	}
}

func TestRootDomain(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{"https://www.example.com/page", "example.com"},
		{"https://sub.portal.example.com/test", "example.com"},
		{"https://portal.site.gov.bd/login", "site.gov.bd"},
		{"https://api.app.co.uk/index.php", "app.co.uk"},
		{"http://test.org.in/admin", "test.org.in"},
		{"http://192.168.1.1:8080/index.php", "192.168.1.1"},
		{"http://[::1]/index.php", "::1"},
	}

	for _, tc := range tests {
		got := RootDomain(tc.url)
		if got != tc.expected {
			t.Errorf("RootDomain(%s) = %s; want %s", tc.url, got, tc.expected)
		}
	}
}

func TestNegativeExclusions(t *testing.T) {
	dork := `inurl:php?id= -github -site:excluded.com -inurl:test`
	rule := ExtractRulesFromDork(dork)

	validURL := "https://target.com/page.php?id=123"
	if !IsValidURL(validURL, rule) {
		t.Errorf("Expected %s to be valid", validURL)
	}

	rejectedURLs := []string{
		"https://target.com/page.php?id=123&ref=github",
		"https://excluded.com/page.php?id=123",
		"https://sub.excluded.com/page.php?id=123",
		"https://target.com/test/page.php?id=123",
	}

	for _, u := range rejectedURLs {
		if IsValidURL(u, rule) {
			t.Errorf("Expected %s to be rejected by negative exclusions", u)
		}
	}
}

func TestFileTypeFiltering(t *testing.T) {
	dork := `ext:sql`
	rule := ExtractRulesFromDork(dork)

	validURL := "https://example.com/backup/database.sql"
	if !IsValidURL(validURL, rule) {
		t.Errorf("Expected %s to match ext:sql", validURL)
	}

	// Should not match an HTML article discussing sql
	falsePositive := "https://example.com/articles/how-to-sql-guide.html"
	if IsValidURL(falsePositive, rule) {
		t.Errorf("Expected %s to NOT match ext:sql", falsePositive)
	}
}

func TestGenerateClusterDorks(t *testing.T) {
	dork := "inurl:admin/login.php -site:github.com"
	cluster := GenerateClusterDorks(dork)
	if len(cluster) == 0 {
		t.Fatal("Expected cluster to produce dork variants")
	}

	foundCombo := false
	for _, cd := range cluster {
		if cd == "inurl:admin inurl:login.php -site:github.com" {
			foundCombo = true
			break
		}
	}
	if !foundCombo {
		t.Errorf("Expected decomposed cluster dork 'inurl:admin inurl:login.php -site:github.com', got: %v", cluster)
	}
}

func TestExtractRealURL(t *testing.T) {
	raw := "https://www.google.com/url?q=https%3A%2F%2Ftarget.com%2Fadmin%2Flogin.php%3Fid%3D1&sa=U"
	expected := "https://target.com/admin/login.php?id=1"
	got := ExtractRealURL(raw)
	if got != expected {
		t.Errorf("ExtractRealURL(%s) = %s; want %s", raw, got, expected)
	}
}
