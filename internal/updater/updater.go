package updater

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/frozeniceee/DorkX/internal/output"
)

const RepoURL = "github.com/frozeniceee/DorkX@latest"

// Update pulls the latest version from GitHub and updates system binaries
func Update() {
	output.Infof("Updating DorkX to the latest version from GitHub...\n")

	homeDir, _ := os.UserHomeDir()

	// Locate Go executable
	goPath, err := exec.LookPath("go")
	if err != nil {
		if _, statErr := os.Stat("/usr/local/go/bin/go"); statErr == nil {
			goPath = "/usr/local/go/bin/go"
		}
	}

	// 1. Try Go direct install
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
				copyBinary(goBin, "/usr/local/bin/dorkx")
				copyBinary(goBin, "/usr/local/go/bin/dorkx")
				output.Successf("DorkX successfully updated to the latest version!\n")
				fmt.Fprintln(os.Stderr, "Run 'dorkx -version' to verify.")
				return
			}
		}
	}

	// 2. Fallback: Check local repo in current dir or ~/DorkX
	repoDirs := []string{".", filepath.Join(homeDir, "DorkX")}
	for _, dir := range repoDirs {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			output.Infof("Pulling latest version from %s...\n", dir)
			pullCmd := exec.Command("git", "-C", dir, "pull")
			pullCmd.Stdout = os.Stderr
			pullCmd.Stderr = os.Stderr
			if err := pullCmd.Run(); err == nil {
				if goPath != "" {
					outBin := filepath.Join(dir, "dorkx")
					buildCmd := exec.Command(goPath, "build", "-o", outBin, dir)
					buildCmd.Stdout = os.Stderr
					buildCmd.Stderr = os.Stderr
					if err := buildCmd.Run(); err == nil {
						copyBinary(outBin, "/usr/local/bin/dorkx")
						copyBinary(outBin, "/usr/local/go/bin/dorkx")
						output.Successf("DorkX successfully updated to the latest version!\n")
						fmt.Fprintln(os.Stderr, "Run 'dorkx -version' to verify.")
						return
					}
				}
			}
		}
	}

	output.Errorf("Update failed. Please run manually:\n  go install %s\n", RepoURL)
}

func copyBinary(src, dst string) {
	data, err := os.ReadFile(src)
	if err != nil {
		return
	}
	_ = os.WriteFile(dst, data, 0755)
}
