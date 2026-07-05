package downloader

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"vibeark/internal/apps"
	"vibeark/internal/config"
	"vibeark/internal/httpclient"
	"vibeark/internal/i18n"
	"vibeark/internal/updater"
)

// ProgressCallback is called during download with progress info
type ProgressCallback func(transferred, total int64, percent float64)

// StatusCallback is called with status updates
type StatusCallback func(status string, url string)

// Result represents the result of a download operation
type Result struct {
	ID          string
	Success     bool
	ExePath     string
	Version     string
	DownloadURL string
	Error       string
	Cancelled   bool
}

// DownloadAndInstall downloads and installs an app
func DownloadAndInstall(app apps.AppEntry, installDir string, version string, settings *config.Settings, onProgress ProgressCallback, onStatus StatusCallback) Result {
	tempDir := filepath.Join(installDir, ".vibeark-temp")
	appDir := filepath.Join(installDir, app.ID)
	var finalDownloadURL string

	// Ensure directories
	os.MkdirAll(appDir, 0755)
	os.MkdirAll(tempDir, 0755)

	// Fetch version if not provided
	if version == "" {
		// First check lock file
		lock := config.LoadLock(installDir)
		if entry, ok := lock.Entries[app.ID]; ok && entry.InstalledVersion != "" {
			version = entry.InstalledVersion
		}

		// If still empty, try remote check (skip for direct-download)
		if version == "" && app.Source.Type != "direct-download" {
			result := updater.CheckVersion(app, settings.Proxy)
			if result.RemoteVersion != "" {
				version = result.RemoteVersion
			}
		}
	}

	// Get download URL
	downloadURL := app.Download.URL
	if downloadURL != "" && version != "" {
		downloadURL = httpclient.ReplaceVersion(downloadURL, version)
	}

	// If no download URL template, try extracting from source page
	if downloadURL == "" && app.Source.DownloadURLPattern != "" {
		client := httpclient.NewClient(settings.Proxy)
		body, err := client.Get(app.Source.URL)
		if err != nil {
			return Result{ID: app.ID, Success: false, Error: i18n.Tf("dl_fetch_page_fail", err), Version: version}
		}
		extracted, err := httpclient.ExtractVersion(body, app.Source.DownloadURLPattern)
		if err != nil {
			return Result{ID: app.ID, Success: false, Error: i18n.Tf("dl_no_link_found", err), Version: version}
		}
		if !strings.HasPrefix(extracted, "http") {
			extracted = "https://" + extracted
		}
		downloadURL = extracted
	}

	// Handle redirect-page type: fetch redirect page, extract real download URL
	if app.Download.Type == "redirect-page" && app.Download.URL != "" {
		client := httpclient.NewClient(settings.Proxy)
		body, err := client.Get(app.Download.URL)
		if err != nil {
			return Result{ID: app.ID, Success: false, Error: i18n.Tf("dl_redirect_fail", err), Version: version}
		}
		extracted, err := httpclient.ExtractVersion(body, app.Download.RedirectURLPattern)
		if err != nil {
			return Result{ID: app.ID, Success: false, Error: i18n.Tf("dl_no_link_found", err), Version: version}
		}
		downloadURL = extracted
	}

	// Handle GitHub releases
	if app.Download.Type == "github-release" {
		ghURL := getGitHubDownloadURL(app, settings.Proxy)
		if ghURL != "" {
			downloadURL = ghURL
		}
	}

	if downloadURL == "" {
		return Result{ID: app.ID, Success: false, Error: i18n.T("dl_no_download_url"), Version: version}
	}

	// Resolve Lanzou cloud links
	if httpclient.IsLanzouURL(downloadURL) {
		client := httpclient.NewClientNoTimeout(settings.Proxy)
		resolved, err := client.ResolveLanzou(downloadURL)
		if err != nil {
			return Result{ID: app.ID, Success: false, Error: i18n.Tf("dl_lanzou_fail", err), Version: version}
		}
		downloadURL = resolved
	}

	// Apply GitHub proxy prefix if configured and URL is from GitHub
	if settings.GhProxy != "" && isGitHubURL(downloadURL) {
		downloadURL = settings.GhProxy + downloadURL
	}

	// Store final download URL
	finalDownloadURL = downloadURL

	// Determine filename
	ext := getExtension(downloadURL)
	fileName := getFileName(downloadURL, app.ID)
	tempFile := filepath.Join(tempDir, fileName)

	// Store final download URL for result
	finalDownloadURL = downloadURL

	// Download with proxy fallback
	if onStatus != nil {
		onStatus("downloading", finalDownloadURL)
	}
	var downloadResult downloadResult
	ua := app.Download.UA
	if settings.Proxy != "" {
		downloadResult = downloadFile(downloadURL, tempFile, settings.Proxy, ua, onProgress)
		if !downloadResult.success && !downloadResult.cancelled {
			downloadResult = downloadFile(downloadURL, tempFile, "", ua, onProgress)
		}
	} else {
		downloadResult = downloadFile(downloadURL, tempFile, "", ua, onProgress)
	}

	if !downloadResult.success {
		return Result{ID: app.ID, Success: false, Error: downloadResult.error, Cancelled: downloadResult.cancelled, Version: version}
	}

	// Extract
	if onStatus != nil {
		onStatus("extracting", "")
	}
	if err := extractFile(tempFile, appDir, ext); err != nil {
		return Result{ID: app.ID, Success: false, Error: i18n.Tf("dl_extract_fail", err), Version: version}
	}

	// Flatten directory
	flattenDir(appDir)

	// Find exe
	exePath := findExeInDir(appDir, app.ExePath)

	// Cleanup
	os.RemoveAll(tempDir)

	if exePath != "" {
		return Result{ID: app.ID, Success: true, ExePath: exePath, Version: version, DownloadURL: finalDownloadURL}
	}
	return Result{ID: app.ID, Success: true, Version: version, DownloadURL: finalDownloadURL}
}

type downloadResult struct {
	success   bool
	error     string
	cancelled bool
}

func downloadFile(url, destPath, proxyPort, ua string, onProgress ProgressCallback) downloadResult {
	client := httpclient.NewClientNoTimeout(proxyPort)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return downloadResult{error: err.Error()}
	}
	if ua != "" {
		req.Header.Set("User-Agent", ua)
	}

	resp, err := client.HTTPClient.Do(req)
	if err != nil {
		return downloadResult{error: err.Error()}
	}
	defer resp.Body.Close()

	// Accept 200 OK and 206 Partial Content
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return downloadResult{error: fmt.Sprintf("HTTP %d %s", resp.StatusCode, resp.Status)}
	}

	file, err := os.Create(destPath)
	if err != nil {
		return downloadResult{error: err.Error()}
	}
	defer file.Close()

	total := resp.ContentLength
	var transferred int64
	buf := make([]byte, 32*1024)

	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			_, writeErr := file.Write(buf[:n])
			if writeErr != nil {
				return downloadResult{error: writeErr.Error()}
			}
			transferred += int64(n)
			if onProgress != nil {
				percent := float64(0)
				if total > 0 {
					percent = float64(transferred) / float64(total) * 100
				}
				onProgress(transferred, total, percent)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return downloadResult{error: readErr.Error()}
		}
	}

	// Validate download
	stat, err := os.Stat(destPath)
	if err != nil {
		return downloadResult{error: err.Error()}
	}

	if stat.Size() < 1000 {
		os.Remove(destPath)
		return downloadResult{error: i18n.Tf("dl_invalid_bytes", stat.Size())}
	}

	// Check if HTML
	header := make([]byte, 4)
	f, _ := os.Open(destPath)
	f.Read(header)
	f.Close()

	if string(header[:4]) == "<!DO" || string(header[:4]) == "<htm" {
		os.Remove(destPath)
		return downloadResult{error: i18n.T("dl_html_not_file")}
	}

	return downloadResult{success: true}
}

func extractFile(tempFile, appDir, ext string) error {
	// Detect by magic bytes
	header := make([]byte, 8)
	f, err := os.Open(tempFile)
	if err != nil {
		return err
	}
	f.Read(header)
	f.Close()

	isZip := header[0] == 0x50 && header[1] == 0x4B
	is7z := header[0] == 0x37 && header[1] == 0x7A && header[2] == 0xBC
	isExe := header[0] == 0x4D && header[1] == 0x5A

	if isZip || ext == ".zip" {
		return extractZip(tempFile, appDir)
	} else if is7z || ext == ".7z" {
		return extract7z(tempFile, appDir)
	} else if isExe || ext == ".exe" {
		fileName := filepath.Base(tempFile)
		return copyFile(tempFile, filepath.Join(appDir, fileName))
	}

	// Try as zip
	return extractZip(tempFile, appDir)
}

func extractZip(zipPath, destDir string) error {
	// Use PowerShell Expand-Archive with hidden window
	psCmd := fmt.Sprintf("Expand-Archive -Path '%s' -DestinationPath '%s' -Force", zipPath, destDir)
	cmd := exec.Command("powershell", "-NoProfile", "-WindowStyle", "Hidden", "-Command", psCmd)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("extraction failed: %s", string(output))
	}

	return nil
}

func extract7z(archivePath, destDir string) error {
	// Try 7z first
	cmd := exec.Command("7z", "x", archivePath, "-o"+destDir, "-y")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Run(); err != nil {
		// Fallback to PowerShell
		return extractZip(archivePath, destDir)
	}
	return nil
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0755)
}

func flattenDir(dir string) {
	for {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return
		}

		// Remove macOS metadata folder
		for _, e := range entries {
			if e.Name() == "__MACOSX" {
				os.RemoveAll(filepath.Join(dir, e.Name()))
			}
		}

		// Reload after cleanup
		entries, err = os.ReadDir(dir)
		if err != nil || len(entries) != 1 {
			return
		}

		if !entries[0].IsDir() {
			return
		}

		innerDir := filepath.Join(dir, entries[0].Name())
		innerEntries, err := os.ReadDir(innerDir)
		if err != nil {
			return
		}

		for _, entry := range innerEntries {
			oldPath := filepath.Join(innerDir, entry.Name())
			newPath := filepath.Join(dir, entry.Name())
			os.Rename(oldPath, newPath)
		}
		os.Remove(innerDir)
	}
}

func findExeInDir(dir, exePattern string) string {
	if exePattern == "" {
		return ""
	}

	// Convert glob to regex
	pattern := strings.ReplaceAll(exePattern, "*", ".*")
	pattern = strings.ReplaceAll(pattern, "?", ".")
	re, err := regexp.Compile("(?i)^" + pattern + "$")
	if err != nil {
		return ""
	}

	// Check direct
	direct := filepath.Join(dir, exePattern)
	if _, err := os.Stat(direct); err == nil {
		return direct
	}

	// Search in dir
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}

	for _, entry := range entries {
		if !entry.IsDir() && re.MatchString(entry.Name()) {
			return filepath.Join(dir, entry.Name())
		}
	}

	// Search one level deep
	for _, entry := range entries {
		if entry.IsDir() {
			subDir := filepath.Join(dir, entry.Name())
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

func getExtension(url string) string {
	// Remove query parameters
	if idx := strings.Index(url, "?"); idx != -1 {
		url = url[:idx]
	}
	return filepath.Ext(url)
}

func getFileName(url, appID string) string {
	// Remove query parameters
	if idx := strings.Index(url, "?"); idx != -1 {
		url = url[:idx]
	}

	name := filepath.Base(url)
	ext := filepath.Ext(name)

	if !strings.Contains(name, ".") {
		if ext == "" {
			return appID + ".zip"
		}
		return appID + ext
	}

	return name
}

func getGitHubDownloadURL(app apps.AppEntry, proxyPort string) string {
	if app.Source.Repo == "" {
		return ""
	}

	releaseURL := fmt.Sprintf("https://github.com/%s/releases/latest", app.Source.Repo)
	client := httpclient.NewClient(proxyPort)

	body, err := client.Get(releaseURL)
	if err != nil {
		return ""
	}

	// Find download links
	linkRe := regexp.MustCompile(`href="([^"]*\/releases\/download\/[^"]*)"`)
	matches := linkRe.FindAllStringSubmatch(body, -1)

	if len(matches) == 0 {
		return ""
	}

	// Find matching asset
	assetPattern := app.Source.AssetPattern
	if assetPattern == "" {
		assetPattern = "*"
	}

	globPattern := strings.ReplaceAll(assetPattern, "*", ".*")
	globPattern = strings.ReplaceAll(globPattern, "?", ".")
	re, err := regexp.Compile("(?i)" + globPattern)
	if err != nil {
		// Return first zip/7z
		for _, match := range matches {
			href := match[1]
			if strings.HasSuffix(href, ".zip") || strings.HasSuffix(href, ".7z") {
				if strings.HasPrefix(href, "http") {
					return href
				}
				return "https://github.com" + href
			}
		}
		return ""
	}

	// Find matching asset
	for _, match := range matches {
		href := match[1]
		fileName := filepath.Base(href)
		if re.MatchString(fileName) {
			if strings.HasPrefix(href, "http") {
				return href
			}
			return "https://github.com" + href
		}
	}

	// Fallback: first zip/7z
	for _, match := range matches {
		href := match[1]
		if strings.HasSuffix(href, ".zip") || strings.HasSuffix(href, ".7z") {
			if strings.HasPrefix(href, "http") {
				return href
			}
			return "https://github.com" + href
		}
	}

	return ""
}

func isGitHubURL(url string) bool {
	return strings.Contains(url, "github.com") || strings.Contains(url, "githubusercontent.com")
}
