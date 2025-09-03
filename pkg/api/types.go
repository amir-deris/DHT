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
