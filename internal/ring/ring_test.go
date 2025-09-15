package ring

import (
	"testing"
)

func TestRingBasicOperations(t *testing.T) {
	ring := New(10) // 10 virtual nodes per physical node

	// Test adding nodes
	err := ring.AddNode("node1", "127.0.0.1:8080")
	if err != nil {
		t.Fatalf("Failed to add node1: %v", err)
	}

	err = ring.AddNode("node2", "127.0.0.1:8081")
	if err != nil {
		t.Fatalf("Failed to add node2: %v", err)
	}

	err = ring.AddNode("node3", "127.0.0.1:8082")
	if err != nil {
		t.Fatalf("Failed to add node3: %v", err)
	}

	// Test ring size
	if ring.Size() != 3 {
		t.Errorf("Expected ring size 3, got %d", ring.Size())
	}

	// Test preference list
	prefList, err := ring.GetPreferenceList("test-key", 2)
	if err != nil {
		t.Fatalf("Failed to get preference list: %v", err)
	}

	if len(prefList) != 2 {
		t.Errorf("Expected preference list length 2, got %d", len(prefList))
	}

	// Test that preference list contains valid nodes
	for _, nodeID := range prefList {
		address, exists := ring.GetNodeAddress(nodeID)
		if !exists {
			t.Errorf("Node %s not found in ring", nodeID)
		}
		if address == "" {
			t.Errorf("Empty address for node %s", nodeID)
		}
	}

	// Test removing a node
	err = ring.RemoveNode("node2")
	if err != nil {
		t.Fatalf("Failed to remove node2: %v", err)
	}

	if ring.Size() != 2 {
		t.Errorf("Expected ring size 2 after removal, got %d", ring.Size())
	}

	// Test that removed node is not in preference list
	prefList2, err := ring.GetPreferenceList("test-key", 3)
	if err != nil {
		t.Fatalf("Failed to get preference list after removal: %v", err)
	}

	for _, nodeID := range prefList2 {
		if nodeID == "node2" {
			t.Errorf("Removed node2 still appears in preference list")
		}
	}
}

func TestRingConsistency(t *testing.T) {
	ring := New(5)

	// Add nodes in different order
	ring.AddNode("node1", "127.0.0.1:8080")
	ring.AddNode("node2", "127.0.0.1:8081")
	ring.AddNode("node3", "127.0.0.1:8082")

	// Test that same key always maps to same preference list
	key := "consistent-test-key"
	prefList1, _ := ring.GetPreferenceList(key, 2)
	prefList2, _ := ring.GetPreferenceList(key, 2)

	if len(prefList1) != len(prefList2) {
		t.Errorf("Preference list lengths differ: %d vs %d", len(prefList1), len(prefList2))
	}

	for i := range prefList1 {
		if prefList1[i] != prefList2[i] {
			t.Errorf("Preference list order differs at index %d: %s vs %s", i, prefList1[i], prefList2[i])
		}
	}
}

func TestRingEmpty(t *testing.T) {
	ring := New(10)

	// Test operations on empty ring
	_, err := ring.GetPreferenceList("test-key", 1)
	if err == nil || err.Error() != "no nodes in ring" {
		t.Error("Expected error when getting preference list from empty ring")
	}

	// Test removing non-existent node
	err = ring.RemoveNode("nonexistent")
	if err == nil || err.Error() != "node nonexistent does not exist" {
		t.Error("Expected error when removing non-existent node")
	}
}

func TestRingDuplicateNode(t *testing.T) {
	ring := New(10)

	err := ring.AddNode("node1", "127.0.0.1:8080")
	if err != nil {
		t.Fatalf("Failed to add node1: %v", err)
	}

	// Try to add same node again
	err = ring.AddNode("node1", "127.0.0.1:8081")
	if err == nil || err.Error() != "node node1 already exists" {
		t.Error("Expected error when adding duplicate node")
	}
}
