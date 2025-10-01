package storage

import (
	"fmt"
	"sync"
	"time"

	"github.com/amirderis/DHT/internal/clock"
)

// VersionedValue represents a key-value pair with vector clock metadata.
type VersionedValue struct {
	Value     []byte            `json:"value"`
	Version   clock.VectorClock `json:"version"`
	Timestamp time.Time         `json:"timestamp"`
	Tombstone bool
}

// NewVersionedValue creates a new versioned value with the given data and vector clock.
func NewVersionedValue(value []byte, version clock.VectorClock) *VersionedValue {
	return &VersionedValue{
		Value:     value,
		Version:   version,
		Timestamp: time.Now(),
		Tombstone: false,
	}
}

// Copy creates a deep copy of the versioned value.
func (vv *VersionedValue) Copy() *VersionedValue {
	if vv == nil {
		return nil
	}

	// Copy the value bytes
	valueCopy := make([]byte, len(vv.Value))
	copy(valueCopy, vv.Value)

	return &VersionedValue{
		Value:     valueCopy,
		Version:   vv.Version.Copy(),
		Timestamp: vv.Timestamp,
		Tombstone: vv.Tombstone,
	}
}

// IsEmpty returns true if the versioned value has no data.
func (vv *VersionedValue) IsEmpty() bool {
	return vv == nil || len(vv.Value) == 0
}

// VersionedEngine extends the basic Engine interface to handle versioned data.
type VersionedEngine interface {
	// Basic operations with versioned data
	GetVersioned(key string) (*VersionedValue, bool)
	PutVersioned(key string, value *VersionedValue) error
	DeleteVersioned(key string) error
}

var _ VersionedEngine = (*VersionedInMemory)(nil)

// VersionedInMemory is an in-memory implementation of VersionedEngine.
type VersionedInMemory struct {
	mu   sync.RWMutex
	data map[string]*VersionedValue
}

// NewVersionedInMemory creates a new in-memory versioned storage engine.
func NewVersionedInMemory() *VersionedInMemory {
	return &VersionedInMemory{
		data: make(map[string]*VersionedValue),
	}
}

func (s *VersionedInMemory) LockForOperation(f func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f()
}

// GetVersioned retrieves the versioned value for a key.
func (s *VersionedInMemory) GetVersioned(key string) (*VersionedValue, bool) {
	value, exists := s.data[key]
	if !exists {
		return nil, false
	}

	return value.Copy(), true
}

// PutVersioned stores a versioned value for a key.
func (s *VersionedInMemory) PutVersioned(key string, value *VersionedValue) error {
	if value == nil {
		return fmt.Errorf("cannot store nil versioned value")
	}

	s.data[key] = value.Copy()
	return nil
}

// DeleteVersioned marks a key as deleted with a tombstone.
func (s *VersionedInMemory) DeleteVersioned(key string) error {
	if value, ok := s.data[key]; ok {
		value.Tombstone = true
	}
	return nil
}
