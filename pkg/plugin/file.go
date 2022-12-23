package plugin

import (
	"context"
	"fmt"
	"os"
)

type File struct {
	State
}

var _ OutputPlugin = &File{}

func (f *File) Synth(ctx context.Context, options map[string]string) error {
	filename := options["name"]
	b, err := f.Marshal()
	if err != nil {
		return err
	}
	if filename == "" {
		return fmt.Errorf("name: must be provided for output plugin: file")
	}
	return os.WriteFile(filename, b, 0644)
}
