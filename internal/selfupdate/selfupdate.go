// Package selfupdate replaces the running ageout binary with the latest
// release published on GitHub, via github.com/creativeprojects/go-selfupdate.
package selfupdate

import (
	"context"

	su "github.com/creativeprojects/go-selfupdate"
)

// repoSlug is ageout's GitHub repository, matching the module path in go.mod.
const repoSlug = "sig9org/ageout"

// Update checks GitHub for a release newer than current and, if one exists,
// downloads it and replaces the running executable in place. current must be
// a semver version (as produced by the release build's -ldflags); dev builds
// have no meaningful version to compare against and will return an error.
func Update(ctx context.Context, current string) (updated bool, latest string, err error) {
	repo := su.ParseSlug(repoSlug)
	release, err := su.UpdateSelf(ctx, current, repo)
	if err != nil {
		return false, "", err
	}
	return !release.Equal(current), release.Version(), nil
}
