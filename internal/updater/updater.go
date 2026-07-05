package updater

import (
	"fmt"
	"vibeark/internal/apps"
	"vibeark/internal/httpclient"
)

// CheckResult represents the result of a version check
type CheckResult struct {
	ID            string
	RemoteVersion string
	Error         error
}

// CheckVersion checks for the latest version of an app
func CheckVersion(app apps.AppEntry, proxyPort string) CheckResult {
	switch app.Source.Type {
	case "web-scrape":
		return checkWebScrape(app, proxyPort)
	default:
		return CheckResult{
			ID:    app.ID,
			Error: fmt.Errorf("unsupported source type: %s", app.Source.Type),
		}
	}
}

func checkWebScrape(app apps.AppEntry, proxyPort string) CheckResult {
	if app.Source.URL == "" {
		return CheckResult{
			ID:    app.ID,
			Error: fmt.Errorf("no URL specified for web scrape"),
		}
	}

	if app.Source.VersionPattern == "" {
		return CheckResult{
			ID:    app.ID,
			Error: fmt.Errorf("no version pattern specified"),
		}
	}

	client := httpclient.NewClient(proxyPort)
	body, err := client.Get(app.Source.URL)
	if err != nil {
		return CheckResult{
			ID:    app.ID,
			Error: fmt.Errorf("web scrape error: %v", err),
		}
	}

	version, err := httpclient.ExtractVersion(body, app.Source.VersionPattern)
	if err != nil {
		return CheckResult{
			ID:    app.ID,
			Error: fmt.Errorf("version pattern not found: %v", err),
		}
	}

	return CheckResult{
		ID:            app.ID,
		RemoteVersion: version,
	}
}
