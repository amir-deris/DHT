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
}

// NewVersionedValue creates a new versioned value with the given data and vector clock.
func NewVersionedValue(value []byte, version clock.VectorClock) *VersionedValue {
	return &VersionedValue{
		Value:     value,
		Version:   version,
		Timestamp: time.Now(),
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
	}
}

// IsEmpty returns true if the versioned value has no data.
func (vv *VersionedValue) IsEmpty() bool {
	return vv == nil || len(vv.Value) == 0
}

// IsTombstone returns true if this represents a deleted key.
func (vv *VersionedValue) IsTombstone() bool {
	return vv != nil && len(vv.Value) == 0 && !vv.Version.IsEmpty()
}

// CreateTombstone creates a tombstone (deleted key marker) with the given vector clock.
func CreateTombstone(version clock.VectorClock) *VersionedValue {
	return &VersionedValue{
		Value:     []byte{}, // Empty value indicates tombstone
		Version:   version,
		Timestamp: time.Now(),
	}
}

// VersionedEngine extends the basic Engine interface to handle versioned data.
type VersionedEngine interface {
	// Basic operations with versioned data
	GetVersioned(key string) (*VersionedValue, bool)
	PutVersioned(key string, value *VersionedValue) error
	DeleteVersioned(key string, version clock.VectorClock) error

	// Legacy interface for backward compatibility
	Engine
}

// VersionedInMemory is an in-memory implementation of VersionedEngine.
type VersionedInMemory struct {
	mu       sync.RWMutex
	data     map[string]*VersionedValue
}

// NewVersionedInMemory creates a new in-memory versioned storage engine.
func NewVersionedInMemory() *VersionedInMemory {
	return &VersionedInMemory{
		data:     make(map[string]*VersionedValue),
	}
}

// GetVersioned retrieves the versioned value for a key.
func (s *VersionedInMemory) GetVersioned(key string) (*VersionedValue, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	value, exists := s.data[key]
	if !exists {
		return nil, false
	}

	return value.Copy(), true
}

// PutVersioned stores a versioned value for a key.
func (s *VersionedInMemory) PutVersioned(key string, value *VersionedValue) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if value == nil {
		return fmt.Errorf("cannot store nil versioned value")
	}

	s.data[key] = value.Copy()
	return nil
}

// DeleteVersioned marks a key as deleted with a tombstone.
func (s *VersionedInMemory) DeleteVersioned(key string, version clock.VectorClock) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tombstone := CreateTombstone(version)
	s.data[key] = tombstone
	return nil
}

// Legacy Engine interface implementation for backward compatibility

func (s *VersionedInMemory) Get(key string) ([]byte, bool) {
	value, found := s.GetVersioned(key)
	if !found || value.IsTombstone() {
		return nil, false
	}
	return value.Value, true
}

func (s *VersionedInMemory) Put(key string, value []byte) error {
	// Create a simple version for backward compatibility
	version := clock.NewWithNode("legacy")
	versionedValue := NewVersionedValue(value, version)
	return s.PutVersioned(key, versionedValue)
}

func (s *VersionedInMemory) Delete(key string) error {
	// Create a simple version for backward compatibility
	version := clock.NewWithNode("legacy")
	return s.DeleteVersioned(key, version)
}
