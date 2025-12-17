// Package core provides the unified Codex knowledge graph, storage, and plugin interfaces.
package core

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"sync"
	"time"
)

// Helper: deserialize commit from JSON
func UnmarshalCommit(b []byte) (*Commit, error) {
	var c Commit
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

// Helper: generic object unmarshal
func UnmarshalObject(b []byte, v interface{}) error {
	return json.Unmarshal(b, v)
}

// Commit represents a repository commit (unified model)
type Commit struct {
	Hash      string    `json:"hash"`
	Parents   []string  `json:"parents,omitempty"`
	Author    string    `json:"author,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message,omitempty"`
	Objects   []string  `json:"objects,omitempty"`
}

// DiffResult is a simple structured diff between two commits
type DiffResult struct {
	Added    []map[string]interface{} `json:"added"`
	Removed  []map[string]interface{} `json:"removed"`
	Modified []map[string]interface{} `json:"modified"`
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
	// ...existing code for status...
	return m, nil
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

// DiffCommits computes differences between two commits by object hashes
func (r *Repository) DiffCommits(fromHash, toHash string) (*DiffResult, error) {
	fromC, err := r.storage.GetCommit(fromHash)
	if err != nil {
		return nil, err
	}
	toC, err := r.storage.GetCommit(toHash)
	if err != nil {
		return nil, err
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
				if UnmarshalObject(b, &v) == nil {
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
				if UnmarshalObject(b, &v) == nil {
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

// Plugin is the interface all integrations must implement (unified)
type Plugin interface {
	Name() string
	Version() string
	Initialize(config map[string]interface{}) error
	Execute(ctx context.Context, action string, payload interface{}) (interface{}, error)
	Validate() error
	Shutdown() error
}

// PluginRegistry manages all plugins
type PluginRegistry struct {
	plugins map[string]Plugin
	mu      sync.RWMutex
}

var pluginRegistry *PluginRegistry

func initPluginRegistry() {
	pluginRegistry = &PluginRegistry{
		plugins: make(map[string]Plugin),
	}
}

// GetRegistry returns the global plugin registry
func GetRegistry() *PluginRegistry {
	if pluginRegistry == nil {
		initPluginRegistry()
	}
	return pluginRegistry
}

func (pr *PluginRegistry) Register(plugin Plugin) error {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	if err := plugin.Validate(); err != nil {
		return err
	}

	name := plugin.Name()
	if _, exists := pr.plugins[name]; exists {
		return errors.New("plugin already registered")
	}

	pr.plugins[name] = plugin
	return nil
}

func (pr *PluginRegistry) Get(name string) (Plugin, error) {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	plugin, exists := pr.plugins[name]
	if !exists {
		return nil, errors.New("plugin not found")
	}
	return plugin, nil
}

func (pr *PluginRegistry) Execute(ctx context.Context, pluginName, action string, payload interface{}) (interface{}, error) {
	plugin, err := pr.Get(pluginName)
	if err != nil {
		return nil, err
	}
	return plugin.Execute(ctx, action, payload)
}

func (pr *PluginRegistry) ListPlugins() []string {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	var names []string
	for name := range pr.plugins {
		names = append(names, name)
	}
	return names
}

func (pr *PluginRegistry) Unregister(name string) error {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	plugin, exists := pr.plugins[name]
	if !exists {
		return errors.New("plugin not registered")
	}

	if err := plugin.Shutdown(); err != nil {
		// Log and continue to remove it
	}

	delete(pr.plugins, name)
	return nil
}

// More models and API logic will be migrated here.
