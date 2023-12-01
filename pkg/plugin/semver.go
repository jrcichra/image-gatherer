package plugin

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/blang/semver"
	"github.com/jrcichra/image-gatherer/pkg/registry"
)

type Semver struct {
}

type Version struct {
	version semver.Version
	tag     string
}

func getIgnoreRegexes(regexStrings string) ([]*regexp.Regexp, error) {
	ignoreRegexes := make([]*regexp.Regexp, 0)
	if regexStrings == "" {
		return ignoreRegexes, nil
	}
	for _, regexString := range strings.Split(regexStrings, ",") {
		regex, err := regexp.Compile(regexString)
		if err != nil {
			return nil, err
		}
		ignoreRegexes = append(ignoreRegexes, regex)
	}
	return ignoreRegexes, nil
}

func matchesAnIgnoredRegex(version semver.Version, regexes []*regexp.Regexp) bool {
	for _, regex := range regexes {
		if regex.Match([]byte(version.String())) {
			return true
		}
	}
	return false
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
	// build list of semver versions

	versions := make([]Version, 0, len(tags))
	for _, tag := range tags {
		v, err := semver.ParseTolerant(tag)
		if err == nil && !matchesAnIgnoredRegex(v, ignoreRegexes) {
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
