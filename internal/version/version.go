// Package version provides version information for the tk binary.
// Version is determined in order of precedence:
//  1. ldflags injection (-X github.com/iheanyi/tasuku/internal/version.version=v0.6.0)
//  2. debug.ReadBuildInfo() for go install builds
//  3. Hardcoded fallback
package version

import (
	"runtime/debug"
	"sync"
)

// These variables are set via ldflags at build time.
// Example: go build -ldflags "-X github.com/iheanyi/tasuku/internal/version.version=v0.6.0"
var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

// Fallback version when not set by ldflags (must match git tag format)
const fallbackVersion = "v0.6.0"

var versionOnce sync.Once
var resolvedVersion string

// Version returns the current version string.
// It uses ldflags if set, otherwise falls back to debug.BuildInfo,
// and finally to the hardcoded fallback.
func Version() string {
	versionOnce.Do(func() {
		resolvedVersion = resolveVersion()
	})
	return resolvedVersion
}

func resolveVersion() string {
	// 1. Check ldflags-injected version
	if version != "dev" && version != "" {
		return version
	}

	// 2. Try debug.ReadBuildInfo() for go install builds
	if info, ok := debug.ReadBuildInfo(); ok {
		// Check for module version (set when installed via go install)
		if info.Main.Version != "" && info.Main.Version != "(devel)" {
			return info.Main.Version
		}

		// Check for vcs.revision in settings (git commit)
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" && setting.Value != "" {
				// Return short commit hash as version indicator
				if len(setting.Value) > 7 {
					return "dev-" + setting.Value[:7]
				}
				return "dev-" + setting.Value
			}
		}
	}

	// 3. Fallback to hardcoded version
	return fallbackVersion
}

// Commit returns the git commit hash if available.
func Commit() string {
	if commit != "unknown" && commit != "" {
		return commit
	}

	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				return setting.Value
			}
		}
	}

	return "unknown"
}

// Date returns the build date if available.
func Date() string {
	if date != "unknown" && date != "" {
		return date
	}

	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.time" {
				return setting.Value
			}
		}
	}

	return "unknown"
}
