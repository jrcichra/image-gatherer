package plugin

import "testing"

func TestFindMatchingTag(t *testing.T) {
	const fullHash = "abc1234567890abcdef1234567890abcdef123456"

	tests := []struct {
		name string
		tags []string
		want string
	}{
		{
			name: "matches sha- prefixed tag",
			tags: []string{"sha-abc123456"},
			want: "sha-abc123456",
		},
		{
			name: "matches tag without sha- prefix",
			tags: []string{"abc123456"},
			want: "abc123456",
		},
		{
			name: "no match",
			tags: []string{"sha-deadbeef1"},
			want: "",
		},
		{
			name: "rejects non-hex after sha-",
			tags: []string{"sha-zzzzzzzz"},
			want: "",
		},
		{
			name: "rejects fewer than 7 chars",
			tags: []string{"sha-abc12"},
			want: "",
		},
		{
			name: "suffix does not match",
			// "def123456" is not a prefix of fullHash
			tags: []string{"sha-def123456"},
			want: "",
		},
		{
			name: "picks first match among multiple",
			tags: []string{"sha-deadbeef1", "sha-abc123456", "sha-abc1234567"},
			want: "sha-abc123456",
		},
		{
			name: "empty tag list",
			tags: []string{},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := findMatchingTag(fullHash, tt.tags); got != tt.want {
				t.Errorf("findMatchingTag() = %q, want %q", got, tt.want)
			}
		})
	}
}
