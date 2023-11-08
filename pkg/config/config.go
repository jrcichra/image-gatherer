package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Container struct {
	Name       string            `yaml:"container"`
	PluginName string            `yaml:"plugin"`
	Options    map[string]string `yaml:"options"`
	Pin        string            `yaml:"pin"`
}

type Output struct {
	PluginName string            `yaml:"plugin"`
	Options    map[string]string `yaml:"options"`
}

type Config struct {
	Containers map[string]Container `yaml:"containers"`
	Output     Output               `yaml:"output"`
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
