package plugin

import (
	"testing"

	"github.com/blang/semver"
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
			name:    "single version",
			versions: []Version{{version: semver.MustParse("1.0.0"), tag: "1.0.0"}},
			wantTag: "1.0.0",
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
