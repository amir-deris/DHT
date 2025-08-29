package clock

// VectorClock is a simple version vector: node-id -> counter.
// Conflict resolution will be implemented in Phase 3.

type VectorClock map[string]uint64

// Compare returns -1 if a < b, 1 if a > b, 0 if concurrent or equal.
func Compare(a, b VectorClock) int {
	aDom, bDom := true, true
	for k, av := range a {
		if bv, ok := b[k]; !ok || av > bv {
			bDom = false
		}
		if bv, ok := b[k]; ok && av < bv {
			aDom = false
		}
	}
	for k, bv := range b {
		if av, ok := a[k]; !ok || bv > av {
			aDom = false
		}
		if av, ok := a[k]; ok && bv < av {
			bDom = false
		}
	}
	if aDom && !bDom {
		return 1
	}
	if bDom && !aDom {
		return -1
	}
	return 0
}

// Bump increments the counter for nodeID in the clock.
func (vc VectorClock) Bump(nodeID string) {
	vc[nodeID] = vc[nodeID] + 1
}
