package core

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// FSStorage implements core.Storage using the local filesystem under a path
// It stores objects under <base>/.codex/objects/<hash>.json
type FSStorage struct {
	base string
}

// NewFSStorage creates a new FSStorage rooted at base
func NewFSStorage(base string) *FSStorage {
	return &FSStorage{base: base}
}

func (fsys *FSStorage) objectsDir() string {
	return filepath.Join(fsys.base, ".codex", "objects")
}

func ensureDir(p string) error {
	return os.MkdirAll(p, 0o755)
}

// PutObject writes payload into file named by hash
func (fsys *FSStorage) PutObject(hash string, payload []byte) error {
	if err := ensureDir(fsys.objectsDir()); err != nil {
		return err
	}
	path := filepath.Join(fsys.objectsDir(), fmt.Sprintf("%s.json", hash))
	return ioutil.WriteFile(path, payload, 0o644)
}

// GetObject reads the file for hash
func (fsys *FSStorage) GetObject(hash string) ([]byte, error) {
	path := filepath.Join(fsys.objectsDir(), fmt.Sprintf("%s.json", hash))
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// ListObjects lists filenames in objectsDir optionally filtered by prefix
func (fsys *FSStorage) ListObjects(prefix string) ([]string, error) {
	if err := ensureDir(fsys.objectsDir()); err != nil {
		return nil, err
	}
	entries, err := ioutil.ReadDir(fsys.objectsDir())
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".json")
		if prefix == "" || strings.HasPrefix(name, prefix) {
			out = append(out, name)
		}
	}
	return out, nil
}

// PutCommit stores commit as an object (using commit.Hash as filename)
func (fsys *FSStorage) PutCommit(c *Commit) error {
	if c.Hash == "" {
		return fmt.Errorf("commit hash required")
	}
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return fsys.PutObject(c.Hash, b)
}

// GetCommit reads a commit object
func (fsys *FSStorage) GetCommit(hash string) (*Commit, error) {
	b, err := fsys.GetObject(hash)
	if err != nil {
		return nil, err
	}
	return UnmarshalCommit(b)
}
