package ring

import (
	"crypto/md5"
	"fmt"
	"math"
	"sort"
	"sync"
)

// NodeID represents a unique node identifier
type NodeID string

// VNode represents a virtual node on the ring
type VNode struct {
	ID     string // Virtual node ID (e.g., "node1-vnode-0")
	NodeID NodeID // Physical node ID
	Hash   uint64 // Position on the ring
}

// Ring implements consistent hashing with virtual nodes
type Ring struct {
	mu         sync.RWMutex
	vnodes     []VNode
	nodes      map[NodeID]string // nodeID -> address
	vnodeCount int               // Number of virtual nodes per physical node
	ringSize   uint64            // Size of the hash ring (2^64)
}

// New creates a new consistent hashing ring
func New(vnodeCount int) *Ring {
	if vnodeCount <= 0 {
		vnodeCount = 100 // Default virtual nodes per physical node
	}
	return &Ring{
		vnodes:     make([]VNode, 0),
		nodes:      make(map[NodeID]string),
		vnodeCount: vnodeCount,
		ringSize:   math.MaxUint64, //2 ^ 64 - 1
	}
}

// AddNode adds a physical node to the ring with virtual nodes
func (r *Ring) AddNode(nodeID NodeID, address string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.nodes[nodeID]; exists {
		return fmt.Errorf("node %s already exists", nodeID)
	}

	r.nodes[nodeID] = address

	// Create virtual nodes for this physical node
	for i := 0; i < r.vnodeCount; i++ {
		vnodeID := fmt.Sprintf("%s-vnode-%d", nodeID, i)
		hash := r.hash(vnodeID)

		vnode := VNode{
			ID:     vnodeID,
			NodeID: nodeID,
			Hash:   hash,
		}

		r.vnodes = append(r.vnodes, vnode)
	}

	// Sort vnodes by hash position
	sort.Slice(r.vnodes, func(i, j int) bool {
		return r.vnodes[i].Hash < r.vnodes[j].Hash
	})

	return nil
}

// RemoveNode removes a physical node and all its virtual nodes
func (r *Ring) RemoveNode(nodeID NodeID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.nodes[nodeID]; !exists {
		return fmt.Errorf("node %s does not exist", nodeID)
	}

	// Remove all virtual nodes for this physical node
	newVnodes := make([]VNode, 0, len(r.vnodes))
	for _, vnode := range r.vnodes {
		if vnode.NodeID != nodeID {
			newVnodes = append(newVnodes, vnode)
		}
	}
	r.vnodes = newVnodes

	// Remove the physical node
	delete(r.nodes, nodeID)

	return nil
}

// GetPreferenceList returns the N nodes responsible for a key, ordered by proximity
func (r *Ring) GetPreferenceList(key string, N int) ([]NodeID, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.vnodes) == 0 {
		return nil, fmt.Errorf("no nodes in ring")
	}

	if N <= 0 || N > len(r.nodes) {
		N = len(r.nodes)
	}

	keyHash := r.hash(key)

	// Find the first vnode clockwise from the key's position
	startIdx := r.findSuccessorIndex(keyHash)

	// Collect unique nodes in order of proximity
	seen := make(map[NodeID]bool)
	preferenceList := make([]NodeID, 0, N)

	// Search clockwise from the starting position
	for i := 0; i < len(r.vnodes) && len(preferenceList) < N; i++ {
		idx := (startIdx + i) % len(r.vnodes)
		vnode := r.vnodes[idx]

		if !seen[vnode.NodeID] {
			preferenceList = append(preferenceList, vnode.NodeID)
			seen[vnode.NodeID] = true
		}
	}

	return preferenceList, nil
}

// GetNodeAddress returns the address for a given node ID
func (r *Ring) GetNodeAddress(nodeID NodeID) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	address, exists := r.nodes[nodeID]
	return address, exists
}

// GetNodes returns all physical nodes in the ring
func (r *Ring) GetNodes() map[NodeID]string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	nodes := make(map[NodeID]string)
	for nodeID, address := range r.nodes {
		nodes[nodeID] = address
	}
	return nodes
}

// Size returns the number of physical nodes in the ring
func (r *Ring) Size() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.nodes)
}

// findSuccessorIndex finds the index of the first vnode clockwise from the given hash
func (r *Ring) findSuccessorIndex(hash uint64) int {
	// Binary search for the first vnode with hash >= keyHash
	idx := sort.Search(len(r.vnodes), func(i int) bool {
		return r.vnodes[i].Hash >= hash
	})

	// If no vnode found with hash >= keyHash, wrap around to the first vnode
	if idx == len(r.vnodes) {
		idx = 0
	}

	return idx
}

// hash computes a 64-bit hash of the input string
func (r *Ring) hash(input string) uint64 {
	h := md5.Sum([]byte(input))
	// Take first 8 bytes to convert the 16 bytes md5 hash into uint64
	return uint64(h[0])<<56 | uint64(h[1])<<48 | uint64(h[2])<<40 | uint64(h[3])<<32 |
		uint64(h[4])<<24 | uint64(h[5])<<16 | uint64(h[6])<<8 | uint64(h[7])
}
