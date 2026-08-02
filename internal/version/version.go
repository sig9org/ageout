// Package version holds ageout's build-time version string and exposes the
// VCS commit it was built from.
package version

import "runtime/debug"

// Version is ageout's released version. It is overridden at build time via
// -ldflags "-X github.com/sig9org/ageout/internal/version.Version=...".
var Version = "dev"

// Commit returns the short commit hash the running binary was built from.
// It reads Go's automatic VCS build-info stamping (populated by the
// toolchain whenever the build happens inside a git checkout), so it needs
// no ldflags of its own. It returns "unknown" when that information isn't
// available, and appends "-dirty" if the working tree had uncommitted
// changes at build time.
func Commit() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}

	var revision string
	var dirty bool
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			revision = s.Value
		case "vcs.modified":
			dirty = s.Value == "true"
		}
	}
	if revision == "" {
		return "unknown"
	}

	const shortLen = 12
	if len(revision) > shortLen {
		revision = revision[:shortLen]
	}
	if dirty {
		revision += "-dirty"
	}
	return revision
}
