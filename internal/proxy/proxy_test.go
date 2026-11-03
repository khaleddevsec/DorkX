package proxy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewPool(t *testing.T) {
	raw := []string{
		"# Comment line",
		"",
		"http://127.0.0.1:8080",
		"socks5://127.0.0.1:9050",
		"10.0.0.1:3128", // auto-prefix http://
	}

	pool, err := NewPool(raw)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if pool.Len() != 3 {
		t.Errorf("Expected 3 proxies, got %d", pool.Len())
	}

	// Test Round-Robin rotation
	p1 := pool.Next()
	p2 := pool.Next()
	p3 := pool.Next()
	p4 := pool.Next()

	if p1.Host != "127.0.0.1:8080" {
		t.Errorf("Expected first proxy to be 127.0.0.1:8080, got %s", p1.Host)
	}
	if p2.Host != "127.0.0.1:9050" {
		t.Errorf("Expected second proxy to be 127.0.0.1:9050, got %s", p2.Host)
	}
	if p3.Host != "10.0.0.1:3128" {
		t.Errorf("Expected third proxy to be 10.0.0.1:3128, got %s", p3.Host)
	}
	// Wrap around
	if p4.Host != p1.Host {
		t.Errorf("Expected wrapped proxy to match p1, got %s", p4.Host)
	}

	// Test Random returns non-nil
	rnd := pool.Random()
	if rnd == nil {
		t.Error("Expected Random() to return a proxy")
	}
}

func TestNewPoolEmpty(t *testing.T) {
	_, err := NewPool([]string{"", "# only comments"})
	if err == nil {
		t.Error("Expected error when no valid proxies supplied")
	}
}

func TestNewPoolFromFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "proxies.txt")

	content := "http://proxy1:8080\n# comment\nhttp://proxy2:8080\n"
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write temp proxy file: %v", err)
	}

	pool, err := NewPoolFromFile(filePath)
	if err != nil {
		t.Fatalf("NewPoolFromFile failed: %v", err)
	}
	if pool.Len() != 2 {
		t.Errorf("Expected 2 proxies from file, got %d", pool.Len())
	}
}
