package version

import (
	"os/exec"
	"strings"
	"sync"
)

var (
	version     string
	gitCommit   string
	versionOnce sync.Once
)

// getVersionInfo gets version from git tag and commit hash
func getVersionInfo() (string, string) {
	versionOnce.Do(func() {
		// Try to get the latest git tag
		cmd := exec.Command("git", "describe", "--tags", "--abbrev=0")
		output, err := cmd.Output()
		if err == nil && len(output) > 0 {
			version = strings.TrimSpace(string(output))
		} else {
			version = "dev"
		}

		// Get the short commit hash
		cmd = exec.Command("git", "rev-parse", "--short=7", "HEAD")
		output, err = cmd.Output()
		if err == nil {
			gitCommit = strings.TrimSpace(string(output))
		} else {
			gitCommit = ""
		}
	})
	return version, gitCommit
}

// GetVersion returns the version string with git commit if available
func GetVersion() string {
	ver, commit := getVersionInfo()
	if commit != "" && ver != "dev" {
		return ver + "-" + commit
	}
	return ver
}

// GetFullVersion returns version with commit info
func GetFullVersion() string {
	ver, commit := getVersionInfo()
	if commit != "" && ver != "dev" {
		return ver + " (commit: " + commit + ")"
	}
	return ver
}
