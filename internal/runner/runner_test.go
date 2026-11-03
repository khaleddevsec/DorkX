package runner

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/khaleddevsec/DorkX/internal/engine"
)

// mockEngine implements engine.Engine for fast deterministic testing
type mockEngine struct {
	results []engine.Result
}

func (m *mockEngine) Name() string { return "MockEngine" }
func (m *mockEngine) Search(dork string, page int, proxyURL *url.URL, timeout int) ([]engine.Result, error) {
	if page > 0 {
		return nil, nil
	}
	return m.results, nil
}

func TestParseDelay(t *testing.T) {
	tests := []struct {
		input   string
		wantMin float64
		wantMax float64
	}{
		{"1-2", 1.0, 2.0},
		{"0.5-1.5", 0.5, 1.5},
		{"2-1", 1.0, 2.0}, // auto-swap if min > max
		{"invalid", 1.0, 2.0},
	}

	for _, tc := range tests {
		min, max := parseDelay(tc.input)
		if min != tc.wantMin || max != tc.wantMax {
			t.Errorf("parseDelay(%s) = (%v, %v); want (%v, %v)", tc.input, min, max, tc.wantMin, tc.wantMax)
		}
	}
}

func TestConfigSanitization(t *testing.T) {
	cfg := &Config{
		Limit:     -10,
		Threads:   0,
		Timeout:   -5,
		MatchCode: "200, 301, invalid, 404",
		Engine:    &mockEngine{},
	}

	r := New(cfg)
	if r.cfg.Limit != 50 {
		t.Errorf("Expected Limit to default to 50, got %d", r.cfg.Limit)
	}
	if r.cfg.Threads != 1 {
		t.Errorf("Expected Threads to sanitize to 1, got %d", r.cfg.Threads)
	}
	if r.cfg.Timeout != 25 {
		t.Errorf("Expected Timeout to sanitize to 25, got %d", r.cfg.Timeout)
	}
	if !r.matchCodes[200] || !r.matchCodes[301] || !r.matchCodes[404] || r.matchCodes[500] {
		t.Errorf("matchCodes improperly parsed: %v", r.matchCodes)
	}
}

func TestRunContextCancellation(t *testing.T) {
	cfg := &Config{
		Dorks:   []string{"inurl:php?id="},
		Limit:   10,
		Threads: 1,
		Silent:  true,
		Engine: &mockEngine{
			results: []engine.Result{
				{RawURL: "https://example.com/item.php?id=1", Source: "Mock"},
			},
		},
	}

	r := New(cfg)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := r.RunContext(ctx)
	if err == nil {
		t.Error("Expected RunContext to return context cancellation error")
	}
}

func TestRunContextCollectsResults(t *testing.T) {
	cfg := &Config{
		Dorks:        []string{"inurl:php?id="},
		Limit:        5,
		Threads:      1,
		DelayRange:   "0-0",
		Silent:       true,
		UniqueDomain: true,
		Engine: &mockEngine{
			results: []engine.Result{
				{RawURL: "https://site1.com/item.php?id=1", Source: "Mock"},
				{RawURL: "https://site1.com/product.php?id=2", Source: "Mock"}, // should be filtered by UniqueDomain
				{RawURL: "https://site2.com/item.php?id=3", Source: "Mock"},
			},
		},
	}

	r := New(cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := r.RunContext(ctx)
	if err != nil {
		t.Fatalf("Unexpected error from RunContext: %v", err)
	}

	if r.totalFound != 2 {
		t.Errorf("Expected exactly 2 unique domain targets, got %d", r.totalFound)
	}
}
