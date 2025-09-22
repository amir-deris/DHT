package api

// Basic request/response types for client API (subject to change).

type PutRequest struct {
	Key   string `json:"key"`
	Value []byte `json:"value"`
}

type PutResponse struct {
	Version map[string]uint64 `json:"version,omitempty"`
}

type GetResponse struct {
	Key      string              `json:"key"`
	Value    []byte              `json:"value,omitempty"`
	Versions []map[string]uint64 `json:"versions,omitempty"`
	Found    bool                `json:"found"`
}

// Internal replication types

type ReplicateRequest struct {
	Key     string            `json:"key"`
	Value   []byte            `json:"value"`
	Version map[string]uint64 `json:"version"`
}

type ReplicateResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

type ReplicateGetRequest struct {
	Key string `json:"key"`
}

type ReplicateGetResponse struct {
	Key     string            `json:"key"`
	Value   []byte            `json:"value,omitempty"`
	Version map[string]uint64 `json:"version,omitempty"`
	Found   bool              `json:"found"`
}
