package storage

import (
	"testing"
	"time"

	"github.com/amirderis/DHT/internal/clock"
)

func TestNewVersionedValue(t *testing.T) {
	v := NewVersionedValue([]byte("hello"), clock.VectorClock{"node1": 1})
	if string(v.Value) != "hello" {
		t.Errorf("Expected hello, got %s", v.Value)
	}
	if v.Version["node1"] != 1 {
		t.Errorf("Expected version clock for node1 to be 1, got %d", v.Version["node1"])
	}
	if v.Tombstone != false {
		t.Error("Expected tombstone to be false")
	}
	now := time.Now()
	if v.Timestamp.Before(now.Add(-2 * time.Second)) || v.Timestamp.After(now.Add(2 * time.Second)) {
		t.Errorf("Expected timestamp to be close to current time, got difference %f", v.Timestamp.Sub(now).Seconds())
	}
}

func TestCopy(t *testing.T) {
	v1 := NewVersionedValue([]byte("value 1"), clock.VectorClock{"node 1": 1})
	v1.Tombstone = true
	v2 := v1.Copy()
	if string(v2.Value) != "value 1" {
		t.Errorf("Expected %s, got %s", "value 1", string(v2.Value))
	}
	v2.Value = append(v2.Value, byte(123))
	if len(v2.Value) == len(v1.Value) {
		t.Errorf("Expected the copy method to create a copy of value")
	}
	if v2.Version["node 1"] != 1 {
		t.Errorf("Expected 1, got %d", v2.Version["node 1"])
	}
	if v2.Timestamp != v1.Timestamp {
		t.Errorf("Expected same timestamp, got %s, %s", v2.Timestamp, v1.Timestamp)
	}
	if v2.Tombstone != true {
		t.Error("Expected true tombstone")
	}
}

func TestEmpty(t *testing.T) {
	var v *VersionedValue
	if !v.IsEmpty() {
		t.Error("Expected nil VersionedValue to be empty")
	}
	v = NewVersionedValue([]byte{}, clock.VectorClock{})
	if !v.IsEmpty() {
		t.Error("Expected a VersionedValue with no value to be empty")
	}
}

func TestVersionedEngine(t *testing.T) {
	t.Error("to be completed")
}