# Dynamo-like DHT (Go + Kubernetes)

A production-leaning, Dynamo-inspired Distributed Hash Table implemented in Go and designed to run on Kubernetes. It focuses on availability and horizontal scalability using consistent hashing, sloppy quorum, hinted handoff, read repair, vector clocks, gossip-based membership, and Merkle trees for anti-entropy.

## Key Properties

- **Language**: Go (Golang)
- **Orchestration**: Kubernetes
- **Partitioning**: Consistent Hashing with virtual nodes
- **Replication**: Configurable replication factor N
- **Consistency**: Tunable per-request R/W quorums (sloppy quorum under failure)
- **Conflict Resolution**: Vector Clocks + app-level reconciliation hook
- **Failure Handling**: Hinted Handoff, Read Repair, Anti-Entropy via Merkle trees
- **Membership**: Gossip Protocol for node discovery and liveness
- **Observability**: Structured logging, metrics hooks, health/readiness probes

## Architecture Overview

At a high level, the system is a ring of nodes (pods) managed by Kubernetes. Keys are hashed into a 128-bit space and mapped onto the ring via consistent hashing with virtual nodes (vnodes). Each key is replicated to N distinct nodes along the ring. Clients (or a coordinating node) perform reads/writes with tunable R/W quorum semantics. During failures, writes are accepted to the next healthy nodes (sloppy quorum) and kept as hinted handoff until the original replicas recover.

### Components

- Node Server (Go)
  - HTTP and/or gRPC API for client I/O and internal replication
  - Storage engine abstraction (pluggable: in-memory, Badger, Pebble; default: in-memory for dev)
  - Vector Clock metadata stored per object version
  - Merkle tree per vnode for anti-entropy
  - Gossip-based membership 
  - Background workers: hinted handoff delivery, read-repair triggers, anti-entropy sync
- Kubernetes Manifests/Helm Chart
  - StatefulSet/Deployment, Headless Service, PodDisruptionBudget
  - ConfigMap/Secret for config, RBAC, ServiceAccount
  - Liveness/readiness probes

### Data Flow (Write)

1. Client sends PUT(key, value) with optional consistency override.
2. Coordinator (any node) hashes key → determines the preference list (N replicas).
3. Coordinator issues writes to N replicas; waits for W acks.
4. If some replicas are down, uses sloppy quorum to write to next healthy nodes and stores hints.
5. On recovery, hints are delivered back to the original replicas.

### Data Flow (Read)

1. Client sends GET(key) with optional consistency override.
2. Coordinator reads from R replicas from the preference list.
3. If divergent versions exist, uses vector clocks to reconcile; returns resolved value.
4. If non-coordinator replicas were stale, coordinator performs read repair in the background.

### Anti-Entropy

- Each vnode maintains a Merkle tree snapshot of its key ranges.
- Periodically, replica pairs compare tree roots and only sync differing subtrees to minimize data transfer.

## Status

This repository currently contains the initial project scaffolding. Implementation will be built incrementally with a focus on correctness, clarity, and test coverage.


## License

This project is licensed under the terms of the MIT License. See `LICENSE`.

