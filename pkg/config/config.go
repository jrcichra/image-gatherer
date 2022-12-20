package config

import (
	"os"

	"github.com/jrcichra/latest-image-gatherer/pkg/plugin"
	"gopkg.in/yaml.v3"
)

type Entry struct {
	Container  string `yaml:"container"`
	UpdateType string `yaml:"update_type"`
	// plugin types
	Git plugin.Git `yaml:"git"`
}

type Config struct {
	Entries map[string]Entry `yaml:"containers"`
}

func LoadConfig(path string) (Config, error) {
	var c Config
	buf, err := os.ReadFile(path)
	if err != nil {
		return c, err
	}
	if err := yaml.Unmarshal(buf, &c); err != nil {
		return c, err
	}
	return c, nil
}

func LoadConfigOrDie(path string) Config {
	c, err := LoadConfig(path)
	if err != nil {
		panic(err)
	}
	return c
}
