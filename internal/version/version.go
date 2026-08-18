// Package version holds ageout's build-time version string.
package version

// Version is ageout's released version. It is overridden at build time via
// -ldflags "-X github.com/sig9org/ageout/internal/version.Version=...".
var Version = "dev"
