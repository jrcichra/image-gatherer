package plugin

import (
	"context"
	"fmt"
	"strings"

	"github.com/blang/semver"
	"github.com/jrcichra/image-gatherer/pkg/registry"
)

type Semver struct {
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
	versions := make([]semver.Version, 0, len(tags))
	prefix := ""
	for _, tag := range tags {
		if strings.HasPrefix(tag, "v") {
			prefix = "v"
		}
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
	result := fmt.Sprintf("%s:%s%s", container, prefix, latest.String())
	return result, nil
}
