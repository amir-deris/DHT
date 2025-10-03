package clock

import (
	"fmt"
	"sort"
	"strings"
)

// VectorClock is a version vector: node-id -> counter.
// Used for conflict detection and causality tracking in distributed systems.
type VectorClock map[string]uint64

// New creates a new empty vector clock.
func New() VectorClock {
	return make(VectorClock)
}

// NewWithNode creates a new vector clock with a single node initialized to 1.
func NewWithNode(nodeID string) VectorClock {
	vc := New()
	vc[nodeID] = 1
	return vc
}

// Compare returns the relationship between two vector clocks:
// -1 if a happens before b (a < b)
//
//	1 if a happens after b (a > b)
//	0 if a and b are concurrent or equal
func Compare(a, b VectorClock) int {
	if a == nil && b == nil {
		return 0
	}
	if a == nil {
		return -1
	}
	if b == nil {
		return 1
	}

	aDom, bDom := true, true

	// Check if a dominates b
	for k, av := range a {
		if bv, ok := b[k]; !ok || av > bv {
			bDom = false
		}
		if bv, ok := b[k]; ok && av < bv {
			aDom = false
		}
	}

	// Check if b dominates a
	for k, bv := range b {
		if av, ok := a[k]; !ok || bv > av {
			aDom = false
		}
		if av, ok := a[k]; ok && bv < av {
			bDom = false
		}
	}

	if aDom && !bDom {
		return 1 // a > b
	}
	if bDom && !aDom {
		return -1 // a < b
	}
	return 0 // concurrent or equal
}

// Increment increments the counter for nodeID in the clock.
func (vc VectorClock) Increment(nodeID string) {
	if vc == nil {
		vc = New()
	}
	vc[nodeID] = vc[nodeID] + 1
}

// Merge creates a new vector clock that contains the maximum value for each node.
// This is used to create a clock that happens after both input clocks.
func (a VectorClock) Merge(b VectorClock) VectorClock {
	if a == nil && b == nil {
		return New()
	}
	if a == nil {
		return b.Copy()
	}
	if b == nil {
		return a.Copy()
	}

	merged := New()

	// Add all nodes from a
	for nodeID, value := range a {
		merged[nodeID] = value
	}

	// Add nodes from b, taking the maximum
	for nodeID, value := range b {
		if existing, ok := merged[nodeID]; !ok || value > existing {
			merged[nodeID] = value
		}
	}

	return merged
}

// Copy creates a deep copy of the vector clock.
func (vc VectorClock) Copy() VectorClock {
	if vc == nil {
		return New()
	}

	copy := New()
	for nodeID, value := range vc {
		copy[nodeID] = value
	}
	return copy
}

// IsEmpty returns true if the vector clock has no entries.
func (vc VectorClock) IsEmpty() bool {
	return len(vc) == 0
}

// String returns a human-readable string representation of the vector clock.
func (vc VectorClock) String() string {
	if vc.IsEmpty() {
		return "{}"
	}

	// Sort nodes for consistent output
	nodes := make([]string, 0, len(vc))
	for nodeID := range vc {
		nodes = append(nodes, nodeID)
	}
	sort.Strings(nodes)

	parts := make([]string, 0, len(nodes))
	for _, nodeID := range nodes {
		parts = append(parts, fmt.Sprintf("%s:%d", nodeID, vc[nodeID]))
	}

	return "{" + strings.Join(parts, ", ") + "}"
}
