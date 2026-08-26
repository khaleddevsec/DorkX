package updater

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/frozeniceee/DorkX/internal/output"
)

const RepoURL = "github.com/frozeniceee/DorkX@latest"

// GetLatestVersion checks GitHub for the newest AppVersion
func GetLatestVersion() string {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("https://raw.githubusercontent.com/frozeniceee/DorkX/main/main.go")
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return ""
	}

	re := regexp.MustCompile(`AppVersion\s*=\s*"([^"]+)"`)
	matches := re.FindStringSubmatch(string(body))
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// VersionInfo returns formatted version string with (latest) or (outdated)
func VersionInfo(currentVersion string) string {
	latest := GetLatestVersion()
	if latest == "" {
		return currentVersion
	}

	if strings.TrimPrefix(currentVersion, "v") == strings.TrimPrefix(latest, "v") {
		return fmt.Sprintf("%s (%s)", currentVersion, "\033[32mlatest\033[0m")
	}

	return fmt.Sprintf("%s (%s -> %s available, run: dorkx -update)", currentVersion, "\033[33moutdated\033[0m", latest)
}

// Update pulls the latest version from GitHub, builds, and replaces active binaries
func Update(currentVersion string) {
	output.Infof("Checking for DorkX updates from GitHub...\n")

	latest := GetLatestVersion()
	if latest != "" && strings.TrimPrefix(currentVersion, "v") == strings.TrimPrefix(latest, "v") {
		output.Successf("DorkX is already at the latest version (%s)!\n", currentVersion)
		return
	}

	output.Infof("Updating DorkX to %s...\n", latest)

	homeDir, _ := os.UserHomeDir()

	// Locate Go compiler
	goPath, err := exec.LookPath("go")
	if err != nil {
		if _, statErr := os.Stat("/usr/local/go/bin/go"); statErr == nil {
			goPath = "/usr/local/go/bin/go"
		}
	}

	var compiledBinary string

	// 1. Direct Go install
	if goPath != "" {
		cmd := exec.Command(goPath, "install", "-v", RepoURL)
		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err == nil {
			goBin := filepath.Join(homeDir, "go", "bin", "dorkx")
			if os.Getenv("GOBIN") != "" {
				goBin = filepath.Join(os.Getenv("GOBIN"), "dorkx")
			} else if os.Getenv("GOPATH") != "" {
				goBin = filepath.Join(os.Getenv("GOPATH"), "bin", "dorkx")
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
					outBin := filepath.Join(dir, "dorkx")
					buildCmd := exec.Command(goPath, "build", "-o", outBin, dir)
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

	// Overwrite all possible system paths safely
	selfPath, _ := os.Executable()
	destinations := []string{
		selfPath,
		"/usr/local/bin/dorkx",
		"/usr/local/go/bin/dorkx",
		filepath.Join(homeDir, "go", "bin", "dorkx"),
		filepath.Join(homeDir, "DorkX", "dorkx"),
	}

	for _, dst := range destinations {
		if dst != "" {
			replaceBinary(compiledBinary, dst)
		}
	}

	output.Successf("DorkX successfully updated to %s!\n", latest)
	fmt.Fprintln(os.Stderr, "Run 'dorkx -version' to verify.")
}

func replaceBinary(src, dst string) {
	data, err := os.ReadFile(src)
	if err != nil {
		return
	}
	// In Linux, unlinking the destination first avoids "text file busy" (ETXTBSY)
	_ = os.Remove(dst)
	_ = os.WriteFile(dst, data, 0755)
}
