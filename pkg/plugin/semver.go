package plugin

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/blang/semver"
	"github.com/jrcichra/image-gatherer/pkg/registry"
)

type Semver struct{}

type Version struct {
	version semver.Version
	tag     string
}

func getIgnoreRegexes(regexStrings string) ([]*regexp.Regexp, error) {
	if regexStrings == "" {
		return nil, nil
	}
	parts := strings.Split(regexStrings, ",")
	regexes := make([]*regexp.Regexp, 0, len(parts))
	for _, regexString := range parts {
		regex, err := regexp.Compile(regexString)
		if err != nil {
			return nil, err
		}
		regexes = append(regexes, regex)
	}
	return regexes, nil
}

func matchesAnIgnoredRegex(version semver.Version, regexes []*regexp.Regexp) bool {
	for _, regex := range regexes {
		if regex.MatchString(version.String()) {
			return true
		}
	}
	return false
}

func findLatestVersion(versions []Version) (Version, error) {
	if len(versions) == 0 {
		return Version{}, fmt.Errorf("no valid semver tags found")
	}
	latest := versions[0]
	for _, v := range versions[1:] {
		if v.version.GT(latest.version) {
			latest = v
		}
	}
	return latest, nil
}

func (s *Semver) GetTag(ctx context.Context, container string, options map[string]string) (string, error) {
	ignoreRegexes, err := getIgnoreRegexes(options["ignore_regexes"])
	if err != nil {
		return "", err
	}
	r, err := registry.NewRegistry(container)
	if err != nil {
		return "", err
	}
	tags, err := r.GetAllTags(ctx, container)
	if err != nil {
		return "", err
	}

	versions := make([]Version, 0, len(tags))
	for _, tag := range tags {
		v, err := semver.ParseTolerant(tag)
		if err == nil && !matchesAnIgnoredRegex(v, ignoreRegexes) {
			versions = append(versions, Version{version: v, tag: tag})
		}
	}

	latest, err := findLatestVersion(versions)
	if err != nil {
		return "", fmt.Errorf("%s: %w", container, err)
	}
	return fmt.Sprintf("%s:%s", container, latest.tag), nil
}
