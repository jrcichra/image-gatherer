package plugin

import (
	"context"
	"fmt"
	"regexp"
	"runtime"
	"sort"
	"strings"

	"github.com/blang/semver"
	"github.com/google/go-containerregistry/pkg/v1"
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

// targetPlatform returns the platform an image should be verified against.
// It defaults to a linux image matching this process's architecture (matching
// the arch the cluster pods we schedule run on) and can be overridden with the
// plugin option `platform` in the "os/arch" form.
func targetPlatform(options map[string]string) v1.Platform {
	arch := runtime.GOARCH
	if p, ok := options["platform"]; ok && p != "" {
		if os, arch, found := strings.Cut(p, "/"); found && os != "" {
			return v1.Platform{OS: os, Architecture: arch}
		}
	}
	return v1.Platform{OS: "linux", Architecture: arch}
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
	if len(versions) == 0 {
		return "", fmt.Errorf("%s: no valid semver tags found", container)
	}

	// Highest version first, so we return the newest tag that actually
	// resolves to a pullable image for our platform. This guards against a
	// partial/in-flight push where the newest tag is listed in the registry
	// but its manifest is not (yet) valid.
	sort.Slice(versions, func(i, j int) bool { return versions[i].version.GT(versions[j].version) })

	platform := targetPlatform(options)
	var lastErr error
	for _, v := range versions {
		err := r.VerifyPull(ctx, v.tag, platform)
		if err == nil {
			return fmt.Sprintf("%s:%s", container, v.tag), nil
		}
		lastErr = fmt.Errorf("%s:%s is not pullable for %s/%s: %w",
			container, v.tag, platform.OS, platform.Architecture, err)
	}
	return "", fmt.Errorf("%s: no pullable version found (last error: %v)", container, lastErr)
}
