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
	defer o.mutex.Unlock()
	o.Containers[key] = value
}

func (o *Output) Marshal() ([]byte, error) {
	o.mutex.RLock()
	defer o.mutex.RUnlock()
	return yaml.Marshal(o)
}
