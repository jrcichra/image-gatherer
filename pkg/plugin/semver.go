package plugin

import (
	"context"
	"fmt"

	"github.com/blang/semver"
	"github.com/jrcichra/latest-image-gatherer/pkg/registry"
)

type Semver struct {
}

func (s *Semver) Get(ctx context.Context, container string) (string, string, error) {
	r, err := registry.NewRegistry(container)
	if err != nil {
		return "", "", err
	}
	tags, err := r.GetAllTags(ctx, container)
	if err != nil {
		return "", "", err
	}
	// build list of semver versions
	versions := make([]semver.Version, 0, len(tags))
	for _, tag := range tags {
		v, err := semver.ParseTolerant(tag)
		if err == nil {
			versions = append(versions, v)
		}
	}
	// find the latest
	var latest semver.Version
	for _, v := range versions {
		if v.GT(latest) {
			latest = v
		}
	}
	digest, err := r.GetDigestFromTag(ctx, container, latest.String())
	if err != nil {
		return "", "", err
	}
	// the response should be the full output
	result := fmt.Sprintf("%s@%s", container, digest)
	return result, latest.String(), nil
}
