package fs

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"veil/pkg/codex"
)

// FSStorage implements codex.Storage using the local filesystem under a path
// It stores objects under <base>/.codex/objects/<hash>.json
type FSStorage struct {
	base string
}

// New creates a new FSStorage rooted at base
func New(base string) *FSStorage {
	return &FSStorage{base: base}
}

func (fsys *FSStorage) objectsDir() string {
	return filepath.Join(fsys.base, ".codex", "objects")
}

func ensureDir(p string) error {
	return os.MkdirAll(p, 0o755)
}

func (fsys *FSStorage) refsDir() string {
	return filepath.Join(fsys.base, ".codex", "refs")
}

// PutObject writes payload into file named by hash
func (fsys *FSStorage) PutObject(hash string, payload []byte) error {
	if err := ensureDir(fsys.objectsDir()); err != nil {
		return err
	}
	// Legacy behavior: store JSON payload
	path := filepath.Join(fsys.objectsDir(), fmt.Sprintf("%s.json", hash))
	return ioutil.WriteFile(path, payload, 0o644)
}

// GetObject reads the file for hash
func (fsys *FSStorage) GetObject(hash string) ([]byte, error) {
	// Prefer legacy JSON file
	jsonPath := filepath.Join(fsys.objectsDir(), fmt.Sprintf("%s.json", hash))
	if b, err := ioutil.ReadFile(jsonPath); err == nil {
		return b, nil
	}
	// Fallback to raw data file
	dataPath := filepath.Join(fsys.objectsDir(), fmt.Sprintf("%s.data", hash))
	b, err := ioutil.ReadFile(dataPath)
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
		name := e.Name()
		// strip known suffixes
		if strings.HasSuffix(name, ".json") {
			name = strings.TrimSuffix(name, ".json")
		} else if strings.HasSuffix(name, ".data") {
			name = strings.TrimSuffix(name, ".data")
		} else if strings.HasSuffix(name, ".meta.json") {
			// skip metadata files
			continue
		}
		if prefix == "" || strings.HasPrefix(name, prefix) {
			out = append(out, name)
		}
	}
	return out, nil
}

// PutObjectStream stores an object by streaming data from r. It computes the SHA256
// hash of the content which is used as the object id. It writes a `.data` file and
// a `.meta.json` file containing content-type and size.
func (fsys *FSStorage) PutObjectStream(r io.Reader, contentType string) (string, error) {
	if err := ensureDir(fsys.objectsDir()); err != nil {
		return "", err
	}
	// create a temp file to stream into while computing hash
	tmp, err := ioutil.TempFile(fsys.objectsDir(), "tmpobj-*")
	if err != nil {
		return "", err
	}
	defer func() { tmp.Close(); os.Remove(tmp.Name()) }()

	hasher := sha256.New()
	size := int64(0)
	// tee reader that writes to hasher while copying to tmp file
	mw := io.MultiWriter(hasher, tmp)
	n, err := io.Copy(mw, r)
	if err != nil {
		return "", err
	}
	size = n
	hash := hex.EncodeToString(hasher.Sum(nil))

	// move tmp file to final .data path
	finalPath := filepath.Join(fsys.objectsDir(), fmt.Sprintf("%s.data", hash))
	if err := os.Rename(tmp.Name(), finalPath); err != nil {
		return "", err
	}

	// write metadata
	meta := map[string]interface{}{"content_type": contentType, "size": size}
	mb, _ := json.Marshal(meta)
	_ = ioutil.WriteFile(filepath.Join(fsys.objectsDir(), fmt.Sprintf("%s.meta.json", hash)), mb, 0o644)

	return hash, nil
}

// GetObjectStream returns a ReadCloser for the object and its contentType (if available)
func (fsys *FSStorage) GetObjectStream(hash string) (io.ReadCloser, string, error) {
	dataPath := filepath.Join(fsys.objectsDir(), fmt.Sprintf("%s.data", hash))
	if f, err := os.Open(dataPath); err == nil {
		// try to read metadata
		metaPath := filepath.Join(fsys.objectsDir(), fmt.Sprintf("%s.meta.json", hash))
		ct := "application/octet-stream"
		if mb, err := ioutil.ReadFile(metaPath); err == nil {
			var m map[string]interface{}
			if json.Unmarshal(mb, &m) == nil {
				if s, ok := m["content_type"].(string); ok && s != "" {
					ct = s
				}
			}
		}
		return f, ct, nil
	}
	// fallback: try legacy json file
	jsonPath := filepath.Join(fsys.objectsDir(), fmt.Sprintf("%s.json", hash))
	if f, err := os.Open(jsonPath); err == nil {
		return f, "application/json", nil
	}
	return nil, "", fmt.Errorf("object not found: %s", hash)
}

// PutCommit stores commit as an object (using commit.Hash as filename)
func (fsys *FSStorage) PutCommit(c *codex.Commit) error {
	if c.Hash == "" {
		return fmt.Errorf("commit hash required")
	}
	b, err := codex.MarshalCommit(c)
	if err != nil {
		return err
	}
	return fsys.PutObject(c.Hash, b)
}

// GetCommit reads a commit object
func (fsys *FSStorage) GetCommit(hash string) (*codex.Commit, error) {
	b, err := fsys.GetObject(hash)
	if err != nil {
		return nil, err
	}
	return codex.UnmarshalCommit(b)
}

// PutRef writes a ref file with the given hash
func (fsys *FSStorage) PutRef(ref string, hash string) error {
	if err := ensureDir(fsys.refsDir()); err != nil {
		return err
	}
	path := filepath.Join(fsys.refsDir(), ref)
	// ensure parent dirs exist
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return ioutil.WriteFile(path, []byte(hash), 0o644)
}

// GetRef reads a ref file and returns the hash
func (fsys *FSStorage) GetRef(ref string) (string, error) {
	path := filepath.Join(fsys.refsDir(), ref)
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

// ListRefs lists ref names under refsDir optionally filtered by prefix
func (fsys *FSStorage) ListRefs(prefix string) ([]string, error) {
	if err := ensureDir(fsys.refsDir()); err != nil {
		return nil, err
	}
	var out []string
	err := filepath.Walk(fsys.refsDir(), func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(fsys.refsDir(), p)
		if err != nil {
			return err
		}
		if prefix == "" || strings.HasPrefix(rel, prefix) {
			out = append(out, rel)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
