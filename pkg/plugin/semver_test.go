package plugin

import (
	"context"
	"os"
	"runtime"
	"testing"

	"github.com/blang/semver"
	"github.com/google/go-containerregistry/pkg/v1"
	"github.com/jrcichra/image-gatherer/pkg/registry"
)

func TestFindLatestVersion(t *testing.T) {
	tests := []struct {
		name     string
		versions []Version
		wantTag  string
		wantErr  bool
	}{
		{
			name:    "empty returns error",
			wantErr: true,
		},
		{
			name:     "single version",
			versions: []Version{{version: semver.MustParse("1.0.0"), tag: "1.0.0"}},
			wantTag:  "1.0.0",
		},
		{
			name: "picks latest",
			versions: []Version{
				{version: semver.MustParse("1.0.0"), tag: "1.0.0"},
				{version: semver.MustParse("2.0.0"), tag: "2.0.0"},
				{version: semver.MustParse("1.5.0"), tag: "1.5.0"},
			},
			wantTag: "2.0.0",
		},
		{
			name: "stable beats same pre-release",
			versions: []Version{
				{version: semver.MustParse("1.0.0-rc1"), tag: "1.0.0-rc1"},
				{version: semver.MustParse("1.0.0"), tag: "1.0.0"},
			},
			wantTag: "1.0.0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := findLatestVersion(tt.versions)
			if (err != nil) != tt.wantErr {
				t.Fatalf("findLatestVersion() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got.tag != tt.wantTag {
				t.Errorf("findLatestVersion() tag = %q, want %q", got.tag, tt.wantTag)
			}
		})
	}
}

func TestMatchesAnIgnoredRegex(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		regexCSV string
		want     bool
	}{
		{"no regexes", "1.0.0", "", false},
		{"matches rc pattern", "1.0.0-rc1", `-rc\d+`, true},
		{"no match", "1.0.0", `-rc\d+`, false},
		{"matches one of multiple", "1.0.0-alpha", `-rc\d+,-alpha`, true},
		{"matches none of multiple", "1.0.0", `-rc\d+,-alpha`, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			regexes, err := getIgnoreRegexes(tt.regexCSV)
			if err != nil {
				t.Fatal(err)
			}
			v := semver.MustParse(tt.version)
			if got := matchesAnIgnoredRegex(v, regexes); got != tt.want {
				t.Errorf("matchesAnIgnoredRegex() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetIgnoreRegexes(t *testing.T) {
	t.Run("empty string returns nil", func(t *testing.T) {
		got, err := getIgnoreRegexes("")
		if err != nil {
			t.Fatal(err)
		}
		if got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})
	t.Run("invalid regex returns error", func(t *testing.T) {
		_, err := getIgnoreRegexes(`[invalid`)
		if err == nil {
			t.Error("expected error for invalid regex")
		}
	})
	t.Run("multiple regexes parsed", func(t *testing.T) {
		got, err := getIgnoreRegexes(`-rc\d+,-alpha`)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 2 {
			t.Errorf("expected 2 regexes, got %d", len(got))
		}
	})
}

func TestTargetPlatform(t *testing.T) {
	t.Run("defaults to linux/current arch", func(t *testing.T) {
		got := targetPlatform(map[string]string{})
		if got.OS != "linux" || got.Architecture != runtime.GOARCH {
			t.Errorf("targetPlatform() = %+v, want linux/%s", got, runtime.GOARCH)
		}
	})
	t.Run("overrides via platform option", func(t *testing.T) {
		got := targetPlatform(map[string]string{"platform": "linux/arm64"})
		if got.OS != "linux" || got.Architecture != "arm64" {
			t.Errorf("targetPlatform() = %+v, want linux/arm64", got)
		}
	})
	t.Run("arch-only platform falls back to linux", func(t *testing.T) {
		got := targetPlatform(map[string]string{"platform": "arm64"})
		if got.OS != "linux" || got.Architecture != runtime.GOARCH {
			t.Errorf("targetPlatform() = %+v, want linux/%s", got, runtime.GOARCH)
		}
	})
}

// requireLiveTests skips network-dependent tests unless explicitly enabled, so
// the default unit test suite stays offline and deterministic.
func requireLiveTests(t *testing.T) {
	t.Helper()
	if os.Getenv("IMAGE_GATHERER_LIVE_TESTS") != "1" {
		t.Skip("set IMAGE_GATHERER_LIVE_TESTS=1 to run live registry tests")
	}
}

func TestSemverLiveResolve(t *testing.T) {
	requireLiveTests(t)
	s := &Semver{}
	got, err := s.GetTag(context.Background(), "docker.io/library/busybox", map[string]string{})
	if err != nil {
		t.Fatalf("GetTag() error = %v", err)
	}
	if got == "" || got == "docker.io/library/busybox" {
		t.Fatalf("GetTag() = %q, want a resolvable busybox:<version>", got)
	}
	t.Logf("resolved %s", got)
}

func TestVerifyPullDiscriminates(t *testing.T) {
	requireLiveTests(t)
	r, err := registry.NewRegistry("docker.io/library/busybox")
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}
	ctx := context.Background()
	plat := v1.Platform{OS: "linux", Architecture: "amd64"}

	// A real, published tag must verify.
	if err := r.VerifyPull(ctx, "1.36", plat); err != nil {
		t.Errorf("VerifyPull(busybox:1.36) unexpectedly failed: %v", err)
	}
	// A tag that is not (and has not been) published must fail — this is
	// exactly the class of "junk" tag the guard is meant to reject.
	if err := r.VerifyPull(ctx, "999.999.999", plat); err == nil {
		t.Error("VerifyPull(busybox:999.999.999) unexpectedly succeeded; guard must reject unresolvable tags")
	}
}
