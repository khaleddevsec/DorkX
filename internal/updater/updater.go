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

	// 1. Try updating via "go install github.com/frozeniceee/DorkX@latest"
	goPath, err := exec.LookPath("go")
	if err != nil {
		// Fallback to common Linux path
		if _, statErr := os.Stat("/usr/local/go/bin/go"); statErr == nil {
			goPath = "/usr/local/go/bin/go"
		}
	}

	if goPath != "" {
		cmd := exec.Command(goPath, "install", "-v", RepoURL)
		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err == nil {
			// Find compiled binary
			homeDir, _ := os.UserHomeDir()
			goBin := filepath.Join(homeDir, "go", "bin", "dorkx")
			if os.Getenv("GOBIN") != "" {
				goBin = filepath.Join(os.Getenv("GOBIN"), "dorkx")
			} else if os.Getenv("GOPATH") != "" {
				goBin = filepath.Join(os.Getenv("GOPATH"), "bin", "dorkx")
			}

			if _, err := os.Stat(goBin); err == nil {
				copyBinary(goBin, "/usr/local/bin/dorkx")
				copyBinary(goBin, "/usr/local/go/bin/dorkx")
			}

			output.Successf("DorkX successfully updated to the latest version!\n")
			fmt.Fprintln(os.Stderr, "Run 'dorkx -version' to verify.")
			return
		}
	}

	// 2. Fallback: If inside a git repository, pull and build locally
	if _, err := os.Stat(".git"); err == nil {
		output.Infof("Pulling latest changes from git repository...\n")
		pullCmd := exec.Command("git", "pull")
		pullCmd.Stdout = os.Stderr
		pullCmd.Stderr = os.Stderr
		if err := pullCmd.Run(); err == nil {
			if goPath != "" {
				buildCmd := exec.Command(goPath, "build", "-o", "dorkx", ".")
				buildCmd.Stdout = os.Stderr
				buildCmd.Stderr = os.Stderr
				if err := buildCmd.Run(); err == nil {
					copyBinary("dorkx", "/usr/local/bin/dorkx")
					copyBinary("dorkx", "/usr/local/go/bin/dorkx")
					output.Successf("DorkX successfully updated and recompiled!\n")
					return
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
