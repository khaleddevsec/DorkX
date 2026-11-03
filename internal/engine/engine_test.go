package engine

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestEngineNames(t *testing.T) {
	bing := NewBingEngine()
	if bing.Name() != "Bing" {
		t.Errorf("Expected Bing, got %s", bing.Name())
	}

	yahoo := NewYahooEngine()
	if yahoo.Name() != "Yahoo" {
		t.Errorf("Expected Yahoo, got %s", yahoo.Name())
	}

	ddg := NewDuckDuckGoEngine()
	if ddg.Name() != "DuckDuckGo" {
		t.Errorf("Expected DuckDuckGo, got %s", ddg.Name())
	}

	google := NewGoogleEngine()
	if !strings.Contains(google.Name(), "Google") {
		t.Errorf("Expected Google, got %s", google.Name())
	}

	cse := NewGoogleCSEEngine("dummy_key", "dummy_cx")
	if !strings.Contains(cse.Name(), "Custom Search") {
		t.Errorf("Expected Custom Search, got %s", cse.Name())
	}

	hybrid := NewHybridEngine(bing, yahoo)
	if hybrid.Name() != "Bing + Yahoo" {
		t.Errorf("Expected 'Bing + Yahoo', got %s", hybrid.Name())
	}
}

func TestDecodeBingRedirect(t *testing.T) {
	targetURL := "https://example.com/admin/login.php?id=1"
	encoded := base64.StdEncoding.EncodeToString([]byte(targetURL))

	rawBingLink := "https://www.bing.com/ck/a?!&&p=123&u=a1" + encoded + "&ntb=1"

	got := DecodeBingRedirect(rawBingLink)
	if got != targetURL {
		t.Errorf("DecodeBingRedirect failed: got %s, want %s", got, targetURL)
	}

	// Non-redirect string should be returned as is
	plain := "https://example.com/test"
	if DecodeBingRedirect(plain) != plain {
		t.Errorf("Expected plain URL to return unchanged")
	}
}

func TestBuildClient(t *testing.T) {
	c1 := buildClient(nil, 25)
	c2 := buildClient(nil, 25)
	if c1 != c2 {
		t.Error("Expected client caching to return the same client instance for identical parameters")
	}
}
