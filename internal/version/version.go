// Package version holds build metadata injected at release time via -ldflags.
package version

import "fmt"

var (
	// Version is the semantic release version.
	Version = "1.0.0"
	// Commit is the git commit hash at build time.
	Commit = "none"
	// Date is the build timestamp (UTC).
	Date = "unknown"
)

// String returns the semantic version.
func String() string {
	return Version
}

// Full returns version plus optional build metadata.
func Full() string {
	if Commit == "none" {
		return Version
	}
	return fmt.Sprintf("%s (commit %s, built %s)", Version, Commit, Date)
}
