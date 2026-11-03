package updater

import (
	"strings"
	"testing"
)

func TestVersionInfo(t *testing.T) {
	// Test when version matches or has string format
	info := VersionInfo("v3.1.1")
	if !strings.Contains(info, "v3.1.1") {
		t.Errorf("Expected version info to contain current version, got: %s", info)
	}
}

func TestVersionRegex(t *testing.T) {
	sampleSource := `
package main

const AppVersion = "v3.1.1"
`
	matches := versionRe.FindStringSubmatch(sampleSource)
	if len(matches) < 2 || matches[1] != "v3.1.1" {
		t.Fatalf("Failed to extract version from source with regex: %v", matches)
	}
}

func TestCompareSemver(t *testing.T) {
	tests := []struct {
		v1   string
		v2   string
		want int
	}{
		{"v3.1.1", "v3.1.0", 1},
		{"v3.1.0", "v3.1.1", -1},
		{"v3.1.1", "v3.1.1", 0},
		{"3.2.0", "3.1.9", 1},
		{"v4.0.0", "v3.9.9", 1},
	}

	for _, tc := range tests {
		got := compareSemver(tc.v1, tc.v2)
		if got != tc.want {
			t.Errorf("compareSemver(%s, %s) = %d; want %d", tc.v1, tc.v2, got, tc.want)
		}
	}
}
