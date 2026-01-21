// Package version provides build version information.
package version

import (
	"fmt"
	"runtime"
)

// These variables are set at build time using ldflags.
var (
	// Version is the semantic version (e.g., "1.0.0")
	Version = "dev"

	// Commit is the git commit hash
	Commit = "unknown"

	// BuildDate is the build timestamp
	BuildDate = "unknown"
)

// Info returns formatted version information.
func Info() string {
	return fmt.Sprintf("v%s", Version)
}

// Full returns full version information including commit and build date.
func Full() string {
	return fmt.Sprintf(
		"hytale-downloader %s (%s) built %s\n%s/%s",
		Version,
		Commit,
		BuildDate,
		runtime.GOOS,
		runtime.GOARCH,
	)
}

// Short returns a short version string.
func Short() string {
	if Version == "dev" {
		return "dev"
	}
	return fmt.Sprintf("v%s", Version)
}
