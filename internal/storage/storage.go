package storage

import "sync"

type Storer interface {
	Get(key string) (value []byte, ok bool)
	Put(key string, value []byte) error
	Delete(key string) error
}

// InMemory is a simple in-memory map-backed store for development/testing.
type InMemory struct {
	mu   sync.RWMutex
	data map[string][]byte
}

func NewInMemory() *InMemory {
	return &InMemory{data: make(map[string][]byte)}
}

func (s *InMemory) Get(key string) ([]byte, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.data[key]
	if !ok {
		return nil, false
	}
	// copy to avoid external mutation
	out := make([]byte, len(v))
	copy(out, v)
	return out, true
}

func (s *InMemory) Put(key string, value []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	v := make([]byte, len(value))
	copy(v, value)
	s.data[key] = v
	return nil
}

func (s *InMemory) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
	return nil
}
