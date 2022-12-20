package output

import (
	"sync"

	"gopkg.in/yaml.v3"
)

type Output struct {
	sync.RWMutex
	Containers map[string]string `yaml:"containers"`
}

func NewOutput() *Output {
	var o Output
	o.Containers = make(map[string]string)
	return &o
}

func (o *Output) Add(key, value string) {
	o.Lock()
	o.Containers[key] = value
	o.Unlock()
}

func (o *Output) Marshal() ([]byte, error) {
	return yaml.Marshal(o)
}
