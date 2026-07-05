package updater

import (
	"strconv"
	"strings"

	"github.com/blang/semver/v4"
)

// CompareVersions compares two version strings
// Returns: -1 if a < b, 0 if a == b, 1 if a > b
func CompareVersions(a, b string) int {
	// Try semver first
	sa, errA := semver.ParseTolerant(a)
	sb, errB := semver.ParseTolerant(b)
	if errA == nil && errB == nil {
		return sa.Compare(sb)
	}

	// Fallback: numeric comparison (for build revisions like Chromium)
	na, errA := strconv.Atoi(a)
	nb, errB := strconv.Atoi(b)
	if errA == nil && errB == nil {
		if na < nb {
			return -1
		}
		if na > nb {
			return 1
		}
		return 0
	}

	// String comparison as last resort
	return strings.Compare(a, b)
}
