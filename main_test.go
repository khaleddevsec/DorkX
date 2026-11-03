package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadLines(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "dorks.txt")

	content := "# Comment\ninurl:php?id=\n\n# Another comment\nsite:.gov.bd\n"
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	lines, err := readLines(filePath)
	if err != nil {
		t.Fatalf("readLines failed: %v", err)
	}

	if len(lines) != 2 {
		t.Fatalf("Expected 2 lines, got %d: %v", len(lines), lines)
	}
	if lines[0] != "inurl:php?id=" {
		t.Errorf("Expected first line to be 'inurl:php?id=', got: %s", lines[0])
	}
	if lines[1] != "site:.gov.bd" {
		t.Errorf("Expected second line to be 'site:.gov.bd', got: %s", lines[1])
	}
}

func TestReadLinesNonExistentFile(t *testing.T) {
	_, err := readLines("non_existent_file_12345.txt")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}
