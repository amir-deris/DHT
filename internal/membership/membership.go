package membership

// Placeholder for gossip-based membership and failure detection.
// Phase 4 will implement SWIM-like or memberlist-based gossip.

type Node struct {
	ID   string
	Addr string
}

type Cluster struct{}

func NewCluster() *Cluster { return &Cluster{} }
