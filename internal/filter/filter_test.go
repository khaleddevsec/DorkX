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
	}

	for _, u := range garbageURLs {
		if IsValidURL(u, rule) {
			t.Errorf("Garbage URL [%s] should have been REJECTED for dork [%s]", u, dork)
		}
	}

	validURLs := []string{
		"https://dhhodisha.in/index.php/admin",
		"http://carboncalculator.americanpistachios.org/admin/index.php",
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
