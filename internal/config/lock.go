package config

import (
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// LockEntry represents a single app's lock state
type LockEntry struct {
	ID               string `yaml:"id"`
	Status           string `yaml:"status"` // "installed" | "not-found" | "error"
	InstalledVersion string `yaml:"installedVersion,omitempty"`
	InstalledAt      string `yaml:"installedAt,omitempty"`
	LastChecked      string `yaml:"lastChecked,omitempty"`
	LastDownloaded   string `yaml:"lastDownloaded,omitempty"`
	ExePath          string `yaml:"exePath,omitempty"`
	Error            string `yaml:"error,omitempty"`
}

// LockFile represents the complete lock file
type LockFile struct {
	Version   int                  `yaml:"version"`
	UpdatedAt string               `yaml:"updatedAt"`
	Entries   map[string]*LockEntry `yaml:"entries"`
}

// LockPath returns the lock file path for a given install directory
func LockPath(installDir string) string {
	return filepath.Join(installDir, "vibeark-lock.yaml")
}

// LoadLock loads the lock file, creates empty if not exists
func LoadLock(installDir string) *LockFile {
	p := LockPath(installDir)
	lock := &LockFile{
		Version: 1,
		Entries: make(map[string]*LockEntry),
	}

	if _, err := os.Stat(p); os.IsNotExist(err) {
		return lock
	}

	data, err := os.ReadFile(p)
	if err != nil {
		return lock
	}

	if err := yaml.Unmarshal(data, lock); err != nil {
		return lock
	}

	if lock.Entries == nil {
		lock.Entries = make(map[string]*LockEntry)
	}

	return lock
}

// SaveLock saves the lock file
func SaveLock(installDir string, lock *LockFile) error {
	lock.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	data, err := yaml.Marshal(lock)
	if err != nil {
		return err
	}

	return os.WriteFile(LockPath(installDir), data, 0644)
}

// UpdateLockEntry updates or creates a lock entry
func UpdateLockEntry(lock *LockFile, id string, update LockEntry) {
	entry, exists := lock.Entries[id]
	if !exists {
		entry = &LockEntry{ID: id}
		lock.Entries[id] = entry
	}

	if update.Status != "" {
		entry.Status = update.Status
	}
	if update.InstalledVersion != "" {
		entry.InstalledVersion = update.InstalledVersion
	}
	if update.InstalledAt != "" {
		entry.InstalledAt = update.InstalledAt
	}
	if update.LastChecked != "" {
		entry.LastChecked = update.LastChecked
	}
	if update.LastDownloaded != "" {
		entry.LastDownloaded = update.LastDownloaded
	}
	if update.ExePath != "" {
		entry.ExePath = update.ExePath
	}
	if update.Error != "" {
		entry.Error = update.Error
	}
}

// NowUTC returns current time in RFC3339 format
func NowUTC() string {
	return time.Now().UTC().Format(time.RFC3339)
}
