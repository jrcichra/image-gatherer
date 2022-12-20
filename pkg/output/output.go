package output

import (
	"sync"

	"gopkg.in/yaml.v3"
)

type Output struct {
	mutex      sync.RWMutex
	Containers map[string]string `yaml:"containers"`
}

func NewOutput() *Output {
	var o Output
	o.Containers = make(map[string]string)
	return &o
}

func (o *Output) Add(key, value string) {
	o.mutex.Lock()
	o.Containers[key] = value
	o.mutex.Unlock()
}

func (o *Output) Marshal() ([]byte, error) {
	return yaml.Marshal(o)
}
