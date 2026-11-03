package updater

import (
	"strings"
	"testing"
)

func TestFormatVersionInfo(t *testing.T) {
	latest := FormatVersionInfo("v3.1.3", "v3.1.3")
	if !strings.Contains(latest, "latest") {
		t.Errorf("Expected latest, got: %s", latest)
	}

	outdated := FormatVersionInfo("v3.1.2", "v3.1.3")
	if !strings.Contains(outdated, "outdated") {
		t.Errorf("Expected outdated, got: %s", outdated)
	}

	empty := FormatVersionInfo("v3.1.3", "")
	if empty != "v3.1.3" {
		t.Errorf("Expected v3.1.3, got: %s", empty)
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
