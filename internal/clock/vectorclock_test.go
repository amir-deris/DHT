package clock

import (
	"testing"
)

func TestVectorClockBasicOperations(t *testing.T) {
	vc := New()
	if vc == nil {
		t.Fatal("New() returned nil")
	}
	if !vc.IsEmpty() {
		t.Error("New vector clock should be empty")
	}

	vc = NewWithNode("node1")
	if vc.IsEmpty() {
		t.Error("NewWithNode should not be empty")
	}
	if vc["node1"] != 1 {
		t.Errorf("Expected node1 to have value 1, got %d", vc["node1"])
	}

	vc.Increment("node1")
	if vc["node1"] != 2 {
		t.Errorf("Expected node1 to have value 2 after increment, got %d", vc["node1"])
	}

	vc.Increment("node2")
	if vc["node2"] != 1 {
		t.Errorf("Expected node2 to have value 1 after first increment, got %d", vc["node2"])
	}
}

func TestVectorClockCompare(t *testing.T) {
	vc1 := NewWithNode("node1")
	vc2 := NewWithNode("node1")
	if Compare(vc1, vc2) != 0 {
		t.Error("Equal clocks should compare as 0")
	}

	// Test a < b (a happens before b)
	vc1 = NewWithNode("node1")
	vc2 = NewWithNode("node1")
	vc2.Increment("node1")
	if Compare(vc1, vc2) != -1 {
		t.Error("vc1 should be less than vc2")
	}

	// Test a > b (a happens after b)
	vc1 = NewWithNode("node1")
	vc1.Increment("node1")
	vc2 = NewWithNode("node1")
	if Compare(vc1, vc2) != 1 {
		t.Error("vc1 should be greater than vc2")
	}

	// Test concurrent clocks
	vc1 = NewWithNode("node1")
	vc2 = NewWithNode("node2")
	if Compare(vc1, vc2) != 0 {
		t.Error("Concurrent clocks should compare as 0")
	}

	// Test nil clocks
	if Compare(nil, nil) != 0 {
		t.Error("Two nil clocks should compare as 0")
	}
	if Compare(nil, New()) != -1 {
		t.Error("nil should be less than empty clock")
	}
	if Compare(New(), nil) != 1 {
		t.Error("empty clock should be greater than nil")
	}
}

func TestVectorClockMerge(t *testing.T) {
	// Test merging two clocks
	vc1 := NewWithNode("node1")
	vc1.Increment("node1")
	vc1.Increment("node2")

	vc2 := NewWithNode("node2")
	vc2.Increment("node2")
	vc2.Increment("node3")

	merged := vc1.Merge(vc2)
	if merged["node1"] != 2 || merged["node2"] != 2 || merged["node3"] != 1 {
		t.Errorf("Expected node1=2, node2=2, node3=1, got %v", merged.String())
	}

	// Test merging with nil
	merged = vc1.Merge(nil)
	if Compare(merged, vc1) != 0 {
		t.Error("Merging with nil should return the other clock")
	}
}

func TestVectorClockCopy(t *testing.T) {
	vc1 := NewWithNode("node1")
	vc1.Increment("node2")

	vc2 := vc1.Copy()

	// Modify original
	vc1.Increment("node1")

	// Copy should be unchanged
	if vc2["node1"] != 1 || vc2["node2"] != 1 {
		t.Error("Copy should not be affected by changes to original")
	}
}
