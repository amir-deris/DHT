package storage

import (
	"fmt"
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

var _ VersionedEngine = (*VersionedInMemoryChannel)(nil)

type VersionedInMemoryChannel struct {
	data map[string]*VersionedValue
	cw   chan dataCommand    //for writing
	cr   chan VersionedValue //for reading
}

func NewVersionedInMemoryChannel() *VersionedInMemoryChannel {
	versionedMemory := &VersionedInMemoryChannel{
		data: make(map[string]*VersionedValue),
		cw:   make(chan dataCommand),
		cr:   make(chan VersionedValue),
	}
	go readMessage(versionedMemory)
	return versionedMemory
}

func readMessage(v *VersionedInMemoryChannel) {
	for {
		dataCommand := <-v.cw
		key := dataCommand.key
		switch dataCommand.command {
		case Get:
			if value, ok := v.data[key]; ok {
				v.cr <- *value.Copy()
			} else {
				v.cr <- *NewVersionedValue(nil, nil)
			}
		case Put:
			v.data[key] = dataCommand.value
		case Delete:
			if value, ok := v.data[key]; ok {
				value.Tombstone = true
			}
		default:
			panic("Unknown command")
		}
	}
}

func (v *VersionedInMemoryChannel) GetVersioned(key string) (*VersionedValue, bool) {
	d := dataCommand{
		command: Get,
		key:     key,
	}
	v.cw <- d
	val := <-v.cr
	return &val, true
}

func (v *VersionedInMemoryChannel) PutVersioned(key string, value *VersionedValue) error {
	if value == nil {
		return fmt.Errorf("cannot store nil versioned value")
	}
	d := dataCommand{
		command: Put,
		key:     key,
		value:   value.Copy(),
	}
	v.cw <- d
	fmt.Println("PUT VALUE FOR KEY ", key)
	return nil
}

func (v *VersionedInMemoryChannel) DeleteVersioned(key string) error {
	if value, ok := v.data[key]; ok {
		d := dataCommand{
			command: Delete,
			key:     key,
			value:   value,
		}
		v.cw <- d
	} else {
		return fmt.Errorf("key %s not found", key)
	}
	return nil
}

type dataCommand struct {
	command
	key   string
	value *VersionedValue
}

type command int

const (
	Get command = iota
	Put
	Delete
)
