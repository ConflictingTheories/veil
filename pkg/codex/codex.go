package codex

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
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
// Storage is the interface for pluggable storage backends.
//
// NOTE: New backends should implement the streaming methods for
// efficient handling of large/binary objects. The legacy PutObject/GetObject
// methods are kept for backward compatibility.
type Storage interface {
	// legacy, convenience methods
	PutObject(hash string, payload []byte) error
	GetObject(hash string) ([]byte, error)
	ListObjects(prefix string) ([]string, error)
	PutCommit(c *Commit) error
	GetCommit(hash string) (*Commit, error)

	// streaming methods for large/binary objects
	// PutObjectStream stores content read from r and returns the computed content-hash
	PutObjectStream(r io.Reader, contentType string) (string, error)
	// GetObjectStream returns a reader for the object and its contentType
	GetObjectStream(hash string) (io.ReadCloser, string, error)

	// Reference (ref) operations: used for branches, tags, heads
	PutRef(ref string, hash string) error
	GetRef(ref string) (string, error)
	ListRefs(prefix string) ([]string, error)
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
	// Basic sanity: a commit must have a hash and a timestamp
	if c.Hash == "" || c.Timestamp.IsZero() {
		return nil, fmt.Errorf("invalid commit: missing hash or timestamp")
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

// Ref operations on repository
func (r *Repository) SetRef(ref, hash string) error {
	return r.storage.PutRef(ref, hash)
}

func (r *Repository) GetRef(ref string) (string, error) {
	return r.storage.GetRef(ref)
}

// CreateBranch creates/updates a branch under refs/heads/<name>
func (r *Repository) CreateBranch(name, targetHash string) error {
	ref := filepath.Join("refs", "heads", name)
	return r.SetRef(ref, targetHash)
}

// ListBranches returns branch names under refs/heads
func (r *Repository) ListBranches() ([]string, error) {
	refs, err := r.storage.ListRefs("refs/heads")
	if err != nil {
		return nil, err
	}
	var out []string
	for _, p := range refs {
		// strip prefixes like refs/heads/
		out = append(out, strings.TrimPrefix(p, "refs/heads/"))
	}
	return out, nil
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

// Conflict represents a detected conflict during merge for a logical entity (URN)
type Conflict struct {
	URN    string `json:"urn"`
	Base   string `json:"base"`
	Ours   string `json:"ours"`
	Theirs string `json:"theirs"`
}

// FindCommonAncestor finds a common ancestor commit between two commits by
// walking parent links. Returns empty string if none found.
func (r *Repository) FindCommonAncestor(a, b string) (string, error) {
	// collect ancestors of a
	seen := map[string]struct{}{}
	queue := []string{a}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur == "" {
			continue
		}
		if _, ok := seen[cur]; ok {
			continue
		}
		seen[cur] = struct{}{}
		c, err := r.storage.GetCommit(cur)
		if err != nil || c == nil {
			continue
		}
		for _, p := range c.Parents {
			queue = append(queue, p)
		}
	}

	// walk ancestors of b and return first seen
	queue = []string{b}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur == "" {
			continue
		}
		if _, ok := seen[cur]; ok {
			return cur, nil
		}
		c, err := r.storage.GetCommit(cur)
		if err != nil || c == nil {
			continue
		}
		for _, p := range c.Parents {
			queue = append(queue, p)
		}
	}
	return "", nil
}

// parseURN attempts to read an "urn" field from a JSON object payload.
func parseURN(b []byte) (string, bool) {
	var m map[string]interface{}
	if json.Unmarshal(b, &m) == nil {
		if u, ok := m["urn"].(string); ok && u != "" {
			return u, true
		}
	}
	return "", false
}

// MergeCommits performs a three-way merge between base, ours and theirs commits.
// If base is empty, the repository will attempt to discover a common ancestor.
// It returns the merged Commit (stored in repo) or a list of Conflicts if any were detected.
func (r *Repository) MergeCommits(baseHash, oursHash, theirsHash, author, message string) (*Commit, []Conflict, error) {
	if baseHash == "" {
		a, err := r.FindCommonAncestor(oursHash, theirsHash)
		if err != nil {
			return nil, nil, err
		}
		baseHash = a
	}

	baseC, _ := r.storage.GetCommit(baseHash)
	oursC, err := r.storage.GetCommit(oursHash)
	if err != nil {
		return nil, nil, err
	}
	theirsC, err := r.storage.GetCommit(theirsHash)
	if err != nil {
		return nil, nil, err
	}

	// Build URN maps for base/ours/theirs when available
	baseURN := map[string]string{}
	oursURN := map[string]string{}
	theirsURN := map[string]string{}
	addURN := func(m map[string]string, hash string) {
		if hash == "" {
			return
		}
		if b, err := r.storage.GetObject(hash); err == nil {
			if urn, ok := parseURN(b); ok {
				m[urn] = hash
			}
		}
	}
	if baseC != nil {
		for _, h := range baseC.Objects {
			addURN(baseURN, h)
		}
	}
	for _, h := range oursC.Objects {
		addURN(oursURN, h)
	}
	for _, h := range theirsC.Objects {
		addURN(theirsURN, h)
	}

	// collect union of urns
	urnSet := map[string]struct{}{}
	for u := range baseURN {
		urnSet[u] = struct{}{}
	}
	for u := range oursURN {
		urnSet[u] = struct{}{}
	}
	for u := range theirsURN {
		urnSet[u] = struct{}{}
	}

	conflicts := []Conflict{}
	mergedObjects := map[string]struct{}{}

	// Resolve URN-based objects
	for u := range urnSet {
		b := baseURN[u]
		o := oursURN[u]
		t := theirsURN[u]
		switch {
		case o == t && o != "":
			mergedObjects[o] = struct{}{}
		case b == o && t != "":
			mergedObjects[t] = struct{}{}
		case b == t && o != "":
			mergedObjects[o] = struct{}{}
		case o != "" && t == "":
			mergedObjects[o] = struct{}{}
		case t != "" && o == "":
			mergedObjects[t] = struct{}{}
		default:
			// conflict if both changed and differ
			if o != t {
				conflicts = append(conflicts, Conflict{URN: u, Base: b, Ours: o, Theirs: t})
			}
		}
	}

	// Include non-URN objects by union
	addHashUnion := func(list []string) {
		for _, h := range list {
			// skip if already included by URN resolution
			// simple check: if h matches a URN mapped value, skip
			skip := false
			for _, m := range []map[string]string{baseURN, oursURN, theirsURN} {
				for _, v := range m {
					if v == h {
						skip = true
						break
					}
				}
				if skip {
					break
				}
			}
			if !skip {
				mergedObjects[h] = struct{}{}
			}
		}
	}
	if baseC != nil {
		addHashUnion(baseC.Objects)
	}
	addHashUnion(oursC.Objects)
	addHashUnion(theirsC.Objects)

	if len(conflicts) > 0 {
		return nil, conflicts, nil
	}

	// create merged commit
	objs := []string{}
	for h := range mergedObjects {
		objs = append(objs, h)
	}
	// deterministic sort
	sort.Strings(objs)
	mcommit := &Commit{
		Hash:      "", // hash may be set externally; leave for PutCommit
		Parents:   []string{oursHash, theirsHash},
		Author:    author,
		Timestamp: time.Now().UTC(),
		Message:   message,
		Objects:   objs,
	}
	// compute commit hash deterministically and set it
	mcommit.Hash = computeCommitHash(mcommit)
	if err := r.storage.PutCommit(mcommit); err != nil {
		return nil, nil, err
	}
	return mcommit, nil, nil
}

// computeCommitHash computes a deterministic SHA256 hash for a commit's content
func computeCommitHash(c *Commit) string {
	tmp := *c
	tmp.Hash = ""
	b, _ := json.Marshal(tmp)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

// Convenience wrappers that expose storage operations via Repository.
// These provide a safe, exported surface so callers outside the `codex`
// package can interact with the storage backend without accessing internals.
func (r *Repository) PutObject(hash string, payload []byte) error {
	return r.storage.PutObject(hash, payload)
}

func (r *Repository) PutObjectStream(rd io.Reader, contentType string) (string, error) {
	return r.storage.PutObjectStream(rd, contentType)
}

// PutObjectStreamWithFilename streams content into storage and records filename
// metadata in a sidecar meta JSON located under `<repo.path>/.codex/objects/<hash>.meta.json`.
// This method does not require storage backends to implement filename handling.
func (r *Repository) PutObjectStreamWithFilename(rd io.Reader, contentType, filename string) (string, error) {
	hash, err := r.storage.PutObjectStream(rd, contentType)
	if err != nil {
		return "", err
	}
	// If repository has a filesystem path, attempt to write/merge a sidecar meta file
	if r.path != "" {
		metaPath := filepath.Join(r.path, ".codex", "objects", fmt.Sprintf("%s.meta.json", hash))
		// read existing meta if present
		meta := map[string]interface{}{"content_type": contentType}
		if b, err := ioutil.ReadFile(metaPath); err == nil {
			var existing map[string]interface{}
			if json.Unmarshal(b, &existing) == nil {
				for k, v := range existing {
					meta[k] = v
				}
			}
		}
		if filename != "" {
			meta["filename"] = filename
		}
		mb, _ := json.Marshal(meta)
		_ = ioutil.WriteFile(metaPath, mb, 0o644)
	}
	return hash, nil
}

func (r *Repository) GetObject(hash string) ([]byte, error) {
	return r.storage.GetObject(hash)
}

func (r *Repository) GetObjectStream(hash string) (io.ReadCloser, string, error) {
	return r.storage.GetObjectStream(hash)
}

func (r *Repository) PutCommit(c *Commit) error {
	if c.Hash == "" {
		c.Hash = computeCommitHash(c)
	}
	return r.storage.PutCommit(c)
}

func (r *Repository) GetCommit(hash string) (*Commit, error) {
	return r.storage.GetCommit(hash)
}

func (r *Repository) ListRefs(prefix string) ([]string, error) {
	return r.storage.ListRefs(prefix)
}
