package plugin

import (
	"context"
	"fmt"

	"github.com/blang/semver"
	"github.com/jrcichra/image-gatherer/pkg/registry"
)

type Semver struct {
}

type Version struct {
	version semver.Version
	tag     string
}

func (s *Semver) GetTag(ctx context.Context, container string, options map[string]string) (string, error) {
	r, err := registry.NewRegistry(container)
	if err != nil {
		return "", err
	}
	tags, err := r.GetAllTags(ctx, container)
	if err != nil {
		return "", err
	}
	// build list of semver versions

	versions := make([]Version, 0, len(tags))
	for _, tag := range tags {
		v, err := semver.ParseTolerant(tag)
		if err == nil {
			versions = append(versions, Version{
				version: v,
				tag:     tag,
			})
		}
	}
	// find the latest
	var latest Version
	for _, v := range versions {
		if v.version.GT(latest.version) {
			latest = v
		}
	}
	result := fmt.Sprintf("%s:%s", container, latest.tag)
	return result, nil
}
