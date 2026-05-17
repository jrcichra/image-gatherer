package plugin

import "fmt"

// NewInputPlugin validates options and returns the named input plugin.
func NewInputPlugin(name string, options map[string]string) (InputPlugin, error) {
	switch name {
	case "git":
		if options["url"] == "" {
			return nil, fmt.Errorf("git input plugin: url option is required")
		}
		if options["branch"] == "" {
			return nil, fmt.Errorf("git input plugin: branch option is required")
		}
		return &GitCommit{}, nil
	case "semver":
		if _, err := getIgnoreRegexes(options["ignore_regexes"]); err != nil {
			return nil, fmt.Errorf("semver plugin: invalid ignore_regexes: %w", err)
		}
		return &Semver{}, nil
	default:
		return nil, fmt.Errorf("unknown input plugin: %s", name)
	}
}

// NewOutputPlugin validates options and returns the named output plugin.
func NewOutputPlugin(name string, options map[string]string) (OutputPlugin, error) {
	switch name {
	case "file":
		if options["name"] == "" {
			return nil, fmt.Errorf("file output plugin: name option is required")
		}
		return &File{}, nil
	case "git":
		if options["url"] == "" {
			return nil, fmt.Errorf("git output plugin: url option is required")
		}
		if options["branch"] == "" {
			return nil, fmt.Errorf("git output plugin: branch option is required")
		}
		if options["filename"] == "" {
			return nil, fmt.Errorf("git output plugin: filename option is required")
		}
		return &GitRepo{}, nil
	default:
		return nil, fmt.Errorf("unknown output plugin: %s", name)
	}
}
