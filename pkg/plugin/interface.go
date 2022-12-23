package plugin

import "context"

type InputPlugin interface {
	GetTag(ctx context.Context, name string, options map[string]string) (string, error)
}

type OutputPlugin interface {
	Add(key, value string)
	Synth(ctx context.Context, options map[string]string) error
}
