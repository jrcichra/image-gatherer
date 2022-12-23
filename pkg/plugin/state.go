package plugin

import (
	"sync"

	"gopkg.in/yaml.v3"
)

type State struct {
	mutex      sync.RWMutex
	Containers map[string]string `yaml:"containers"`
}

func (s *State) Add(key, value string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.Containers == nil {
		s.Containers = make(map[string]string)
	}
	s.Containers[key] = value
}

func (s *State) Marshal() ([]byte, error) {
	s.mutex.RLock()
	b, err := yaml.Marshal(s)
	s.mutex.RUnlock()
	return b, err
}
