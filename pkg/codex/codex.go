package codex

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
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

// ListObjects returns a list of object hashes optionally filtered by prefix and paginated
func (r *Repository) ListObjects(prefix string, limit, offset int) ([]string, error) {
	objs, err := r.storage.ListObjects(prefix)
	if err != nil {
		return nil, err
	}
	// simple pagination
	if offset >= len(objs) {
		return []string{}, nil
	}
	end := len(objs)
	if limit > 0 && offset+limit < end {
		end = offset + limit
	}
	return objs[offset:end], nil
}

// ListCommits returns commits stored in the repository, sorted by timestamp desc
func (r *Repository) ListCommits(limit, offset int) ([]*Commit, error) {
	objs, err := r.storage.ListObjects("")
	if err != nil {
		return nil, err
	}
	var commits []*Commit
	for _, h := range objs {
		b, err := r.storage.GetObject(h)
		if err != nil {
			continue
		}
		c, err := UnmarshalCommit(b)
		if err != nil {
			continue
		}
		commits = append(commits, c)
	}
	// sort by timestamp desc
	sort.Slice(commits, func(i, j int) bool { return commits[i].Timestamp.After(commits[j].Timestamp) })
	if offset >= len(commits) {
		return []*Commit{}, nil
	}
	end := len(commits)
	if limit > 0 && offset+limit < end {
		end = offset + limit
	}
	return commits[offset:end], nil
}

// DiffResult is a simple structured diff between two commits
type DiffResult struct {
	Added    []map[string]interface{} `json:"added"`
	Removed  []map[string]interface{} `json:"removed"`
	Modified []map[string]interface{} `json:"modified"`
}

// DiffCommits computes differences between two commits by object hashes
func (r *Repository) DiffCommits(fromHash, toHash string) (*DiffResult, error) {
	fromC, err := r.storage.GetCommit(fromHash)
	if err != nil {
		return nil, fmt.Errorf("from commit error: %w", err)
	}
	toC, err := r.storage.GetCommit(toHash)
	if err != nil {
		return nil, fmt.Errorf("to commit error: %w", err)
	}
	fromSet := make(map[string]struct{})
	toSet := make(map[string]struct{})
	for _, h := range fromC.Objects {
		fromSet[h] = struct{}{}
	}
	for _, h := range toC.Objects {
		toSet[h] = struct{}{}
	}
	res := &DiffResult{}
	// Added: in toSet not in fromSet
	for _, h := range toC.Objects {
		if _, ok := fromSet[h]; !ok {
			// attempt preview
			if b, err := r.storage.GetObject(h); err == nil {
				var v map[string]interface{}
				if json.Unmarshal(b, &v) == nil {
					res.Added = append(res.Added, map[string]interface{}{"hash": h, "preview": v})
					continue
				}
			}
			res.Added = append(res.Added, map[string]interface{}{"hash": h})
		}
	}
	// Removed
	for _, h := range fromC.Objects {
		if _, ok := toSet[h]; !ok {
			if b, err := r.storage.GetObject(h); err == nil {
				var v map[string]interface{}
				if json.Unmarshal(b, &v) == nil {
					res.Removed = append(res.Removed, map[string]interface{}{"hash": h, "preview": v})
					continue
				}
			}
			res.Removed = append(res.Removed, map[string]interface{}{"hash": h})
		}
	}
	// Modified: none in content-addressed store (hash change considered add/remove)
	return res, nil
}
