package codex

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Commit represents a repository commit
type Commit struct {
	Hash      string    `json:"hash"`
	Parents   []string  `json:"parents,omitempty"`
	Author    string    `json:"author,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message,omitempty"`
	Objects   []string  `json:"objects,omitempty"`
}

// Storage is the interface for pluggable storage backends
type Storage interface {
	PutObject(hash string, payload []byte) error
	GetObject(hash string) ([]byte, error)
	ListObjects(prefix string) ([]string, error)
	PutCommit(c *Commit) error
	GetCommit(hash string) (*Commit, error)
}

// Repository is a lightweight wrapper around a storage backend
type Repository struct {
	storage Storage
	path    string // optional filesystem path for repo (for status/debug)
}

// NewRepository creates a Repository over the provided storage backend
func NewRepository(s Storage, path string) *Repository {
	return &Repository{storage: s, path: path}
}

// Status returns a simple status map for quick inspection
func (r *Repository) Status() (map[string]interface{}, error) {
	m := map[string]interface{}{}
	if r.path != "" {
		// try to read .codex/objects count if path provided
		objsDir := filepath.Join(r.path, ".codex", "objects")
		f, err := os.Open(objsDir)
		if err != nil {
			m["objects_dir"] = objsDir
			m["objects_count"] = 0
			m["objects_error"] = err.Error()
			return m, nil
		}
		defer f.Close()
		list, _ := f.Readdirnames(-1)
		m["objects_dir"] = objsDir
		m["objects_count"] = len(list)
	}
	return m, nil
}

// Helper: serialize commit to JSON
func MarshalCommit(c *Commit) ([]byte, error) {
	return json.Marshal(c)
}

// Helper: deserialize commit from JSON
func UnmarshalCommit(b []byte) (*Commit, error) {
	var c Commit
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("invalid commit payload: %w", err)
	}
	return &c, nil
}