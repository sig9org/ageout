// Package selfupdate adapts github.com/sig9org/selfupdate-go to ageout's
// self-update result format.
package selfupdate

import (
	"context"

	su "github.com/sig9org/selfupdate-go"
)

const repoSlug = "sig9org/ageout"

// Update checks GitHub for a release newer than current and, if one exists,
// downloads it and replaces the running executable in place.
func Update(ctx context.Context, current string) (updated bool, latest string, err error) {
	updater, err := su.New(su.Config{
		Repository: repoSlug,
		Validator:  su.SHA256Validator{},
	})
	if err != nil {
		return false, "", err
	}
	result, err := updater.Update(ctx, current)
	if err != nil {
		return false, result.LatestVersion, err
	}
	return result.Updated, result.LatestVersion, nil
}
