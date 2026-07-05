package scanner

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"vibeark/internal/apps"
)

// ScanResult represents the result of scanning an app
type ScanResult struct {
	ID       string
	Status   string // "installed" | "not-found"
	ExePath  string
}

// ScanApp checks if an app is installed by looking for its exe
func ScanApp(app apps.AppEntry, installDir string) ScanResult {
	appDir := filepath.Join(installDir, app.ID)
	exeName := app.ExePath

	// If no wildcard, check direct path
	if !strings.Contains(exeName, "*") && !strings.Contains(exeName, "?") {
		direct := filepath.Join(appDir, exeName)
		if _, err := os.Stat(direct); err == nil {
			return ScanResult{
				ID:      app.ID,
				Status:  "installed",
				ExePath: direct,
			}
		}
	}

	// Search with glob pattern
	found := findExe(appDir, exeName)
	if found != "" {
		return ScanResult{
			ID:      app.ID,
			Status:  "installed",
			ExePath: found,
		}
	}

	return ScanResult{
		ID:     app.ID,
		Status: "not-found",
	}
}

// ScanAll scans all apps
func ScanAll(appDefs []apps.AppEntry, installDir string) []ScanResult {
	results := make([]ScanResult, len(appDefs))
	for i, app := range appDefs {
		results[i] = ScanApp(app, installDir)
	}
	return results
}

// findExe searches for an exe file matching the pattern
func findExe(startDir, exeName string) string {
	// Convert glob pattern to regex
	pattern := strings.ReplaceAll(exeName, "*", ".*")
	pattern = strings.ReplaceAll(pattern, "?", ".")
	re, err := regexp.Compile("(?i)^" + pattern + "$")
	if err != nil {
		return ""
	}

	// Check startDir
	entries, err := os.ReadDir(startDir)
	if err != nil {
		return ""
	}

	for _, entry := range entries {
		if !entry.IsDir() && re.MatchString(entry.Name()) {
			return filepath.Join(startDir, entry.Name())
		}
	}

	// Check one level of subdirectories
	for _, entry := range entries {
		if entry.IsDir() {
			subDir := filepath.Join(startDir, entry.Name())
			subEntries, err := os.ReadDir(subDir)
			if err != nil {
				continue
			}
			for _, subEntry := range subEntries {
				if !subEntry.IsDir() && re.MatchString(subEntry.Name()) {
					return filepath.Join(subDir, subEntry.Name())
				}
			}
		}
	}

	return ""
}
