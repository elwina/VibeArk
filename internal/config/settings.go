package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"

	"gopkg.in/yaml.v3"
)

// Settings represents the application settings
type Settings struct {
	InstallDir string `yaml:"installDir"`
	Proxy      string `yaml:"proxy"`     // Port number like "10808" or empty
	GhProxy    string `yaml:"ghProxy"`   // GitHub release proxy prefix, e.g. "https://gh-proxy.org/"
	Language   string `yaml:"language"`  // "zh" or "en", empty means auto-detect
}

// ConfigDir returns the config directory path (~/.vibeark)
func ConfigDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".vibeark")
}

// SettingsPath returns the settings file path
func SettingsPath() string {
	return filepath.Join(ConfigDir(), "settings.yaml")
}

// GetProxyURL converts port to full proxy URL
func GetProxyURL(port string) string {
	if port == "" {
		return ""
	}
	if matched, _ := regexp.MatchString(`^\d+$`, port); matched {
		return fmt.Sprintf("http://127.0.0.1:%s", port)
	}
	return port
}

// NormalizeProxy extracts port from full URL if needed
func NormalizeProxy(proxy string) string {
	if proxy == "" || proxy == "0" || proxy == "none" {
		return ""
	}
	// Try to extract port from URL like http://127.0.0.1:10808
	re := regexp.MustCompile(`:(\d+)$`)
	if matches := re.FindStringSubmatch(proxy); len(matches) > 1 {
		return matches[1]
	}
	return proxy
}

// LoadSettings loads settings from YAML file, creates default if not exists
func LoadSettings() Settings {
	p := SettingsPath()
	if _, err := os.Stat(p); os.IsNotExist(err) {
		defaults := Settings{
			InstallDir: "D:/ArkApps",
			Proxy:      "10808",
			GhProxy:    "https://gh-proxy.org/",
		}
		SaveSettings(defaults)
		return defaults
	}

	data, err := os.ReadFile(p)
	if err != nil {
		return Settings{InstallDir: "D:/ArkApps", Proxy: "10808", GhProxy: "https://gh-proxy.org/"}
	}

	var s Settings
	if err := yaml.Unmarshal(data, &s); err != nil {
		return Settings{InstallDir: "D:/ArkApps", Proxy: "10808"}
	}

	if s.InstallDir == "" {
		s.InstallDir = "D:/ArkApps"
	}
	s.Proxy = NormalizeProxy(s.Proxy)

	return s
}

// SaveSettings saves settings to YAML file
func SaveSettings(s Settings) error {
	dir := ConfigDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(s)
	if err != nil {
		return err
	}

	return os.WriteFile(SettingsPath(), data, 0644)
}

// ParsePort parses a port string to int, returns 0 if invalid
func ParsePort(port string) int {
	if port == "" {
		return 0
	}
	p, err := strconv.Atoi(port)
	if err != nil {
		return 0
	}
	return p
}
