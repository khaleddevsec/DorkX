// Package updater provides self-updating functionality for DorkX,
// allowing users to check GitHub releases and rebuild or update binaries.
package updater

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/khaleddevsec/DorkX/v3/internal/output"
)

// RepoURL is the module repository path used for go install updates.
const RepoURL = "github.com/khaleddevsec/DorkX/v3@latest"

var versionRe = regexp.MustCompile(`AppVersion\s*=\s*"([^"]+)"`)

// compareSemver compares two semantic versions (e.g. "v3.1.1" vs "v3.1.0").
// Returns 1 if v1 > v2, -1 if v1 < v2, and 0 if v1 == v2.
func compareSemver(v1, v2 string) int {
	clean1 := strings.TrimPrefix(v1, "v")
	clean2 := strings.TrimPrefix(v2, "v")

	parts1 := strings.Split(clean1, ".")
	parts2 := strings.Split(clean2, ".")

	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}

	for i := 0; i < maxLen; i++ {
		var n1, n2 int
		if i < len(parts1) {
			n1, _ = strconv.Atoi(parts1[i])
		}
		if i < len(parts2) {
			n2, _ = strconv.Atoi(parts2[i])
		}
		if n1 > n2 {
			return 1
		}
		if n1 < n2 {
			return -1
		}
	}
	return 0
}

// GetLatestVersion checks GitHub for the newest AppVersion.
func GetLatestVersion() string {
	client := &http.Client{Timeout: 4 * time.Second}
	resp, err := client.Get("https://raw.githubusercontent.com/khaleddevsec/DorkX/main/main.go")
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return ""
	}

	matches := versionRe.FindStringSubmatch(string(body))
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// FormatVersionInfo formats version text based on current and latest versions.
func FormatVersionInfo(currentVersion, latest string) string {
	if latest == "" {
		return currentVersion
	}

	if compareSemver(currentVersion, latest) >= 0 {
		return fmt.Sprintf("%s (%s)", currentVersion, "\033[32mlatest\033[0m")
	}

	return fmt.Sprintf("%s (%s -> %s available, run: dorkx -update)", currentVersion, "\033[33moutdated\033[0m", latest)
}

// VersionInfo returns formatted version string with (latest) or (outdated).
func VersionInfo(currentVersion string) string {
	return FormatVersionInfo(currentVersion, GetLatestVersion())
}

// Update pulls the latest version from GitHub, builds, and replaces active binaries
func Update(currentVersion string) {
	output.Infof("Checking for DorkX updates from GitHub...\n")

	latest := GetLatestVersion()
	if latest != "" && compareSemver(currentVersion, latest) >= 0 {
		output.Successf("DorkX is already at the latest version (%s)!\n", currentVersion)
		return
	}

	if latest != "" {
		output.Infof("Updating DorkX to %s...\n", latest)
	}

	homeDir, _ := os.UserHomeDir()
	binName := "dorkx"
	if runtime.GOOS == "windows" {
		binName = "dorkx.exe"
	}

	// Locate Go compiler
	goPath, err := exec.LookPath("go")
	if err != nil && runtime.GOOS != "windows" {
		if _, statErr := os.Stat("/usr/local/go/bin/go"); statErr == nil {
			goPath = "/usr/local/go/bin/go"
		}
	}

	var compiledBinary string

	// 1. Direct Go install
	if goPath != "" {
		output.Infof("Running: go install %s\n", RepoURL)
		cmd := exec.Command(goPath, "install", "-v", RepoURL)
		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err == nil {
			goBin := filepath.Join(homeDir, "go", "bin", binName)
			if os.Getenv("GOBIN") != "" {
				goBin = filepath.Join(os.Getenv("GOBIN"), binName)
			} else if os.Getenv("GOPATH") != "" {
				goBin = filepath.Join(os.Getenv("GOPATH"), "bin", binName)
			}

			if _, err := os.Stat(goBin); err == nil {
				compiledBinary = goBin
			}
		}
	}

	// 2. Git repo rebuild fallback
	if compiledBinary == "" {
		repoDirs := []string{".", filepath.Join(homeDir, "DorkX")}
		for _, dir := range repoDirs {
			if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
				output.Infof("Pulling from repository at %s...\n", dir)
				pullCmd := exec.Command("git", "-C", dir, "pull")
				pullCmd.Stdout = os.Stderr
				pullCmd.Stderr = os.Stderr
				_ = pullCmd.Run()

				if goPath != "" {
					outBin := filepath.Join(dir, binName)
					buildCmd := exec.Command(goPath, "build", "-o", outBin, ".")
					buildCmd.Dir = dir
					buildCmd.Stdout = os.Stderr
					buildCmd.Stderr = os.Stderr
					if err := buildCmd.Run(); err == nil {
						compiledBinary = outBin
						break
					}
				}
			}
		}
	}

	if compiledBinary == "" {
		output.Errorf("Update failed. Please run manually:\n  go install %s\n", RepoURL)
		return
	}

	// Overwrite current running executable safely
	selfPath, err := os.Executable()
	if err == nil && selfPath != "" {
		if err := replaceBinary(compiledBinary, selfPath); err == nil {
			output.Successf("Updated current binary at %s\n", selfPath)
		}
	}

	output.Successf("DorkX successfully updated!\n")
	fmt.Fprintln(os.Stderr, "Run 'dorkx -version' to verify.")
}

func replaceBinary(src, dst string) error {
	if filepath.Clean(src) == filepath.Clean(dst) {
		return nil
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if runtime.GOOS == "windows" {
		// On Windows, running binaries cannot be removed directly, but can be renamed
		oldPath := dst + ".old"
		_ = os.Remove(oldPath)
		_ = os.Rename(dst, oldPath)
	} else {
		_ = os.Remove(dst)
	}
	return os.WriteFile(dst, data, 0755)
}
