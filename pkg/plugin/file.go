package plugin

import (
	"context"
	"os"

	"gopkg.in/yaml.v3"
)

type File struct {
	State
}

var _ OutputPlugin = &File{}

func (f *File) Open(_ context.Context, options map[string]string) error {
	filename := options["name"]
	existing, err := os.ReadFile(filename)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	return yaml.Unmarshal(existing, &f.State)
}

func (f *File) Close(_ context.Context, options map[string]string) error {
	filename := options["name"]
	b, err := f.Marshal()
	if err != nil {
		return err
	}
	return os.WriteFile(filename, b, 0644)
}
