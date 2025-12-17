package plugins

import (
	"context"
	"fmt"
	"sync"
	"veil/pkg/codex"
)

// === Plugin Architecture ===

// Plugin is the interface all integrations must implement
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
		return fmt.Errorf("plugin validation failed: %v", err)
	}

	name := plugin.Name()
	if _, exists := pr.plugins[name]; exists {
		return fmt.Errorf("plugin %s already registered", name)
	}

	pr.plugins[name] = plugin
	return nil
}

func (pr *PluginRegistry) Get(name string) (Plugin, error) {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	plugin, exists := pr.plugins[name]
	if !exists {
		return nil, fmt.Errorf("plugin %s not found", name)
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

// Unregister removes a plugin by name and invokes its Shutdown method if present
func (pr *PluginRegistry) Unregister(name string) error {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	plugin, exists := pr.plugins[name]
	if !exists {
		return fmt.Errorf("plugin %s not registered", name)
	}

	// Attempt graceful shutdown
	if err := plugin.Shutdown(); err != nil {
		// Log and continue to remove it
		fmt.Printf("plugin %s shutdown error: %v\n", name, err)
	}

	delete(pr.plugins, name)
	return nil
}

// RepositoryAware is an optional interface that plugins can implement
// to receive a reference to the core codex Repository. The plugin manager
// will call AttachRepository when a repository is available.
type RepositoryAware interface {
	AttachRepository(*codex.Repository) error
}

// AttachRepositoryToAll iterates over registered plugins and calls AttachRepository
// for those implementing RepositoryAware. This allows dependency injection of the
// codex Repository into plugins at runtime.
func (pr *PluginRegistry) AttachRepositoryToAll(repo *codex.Repository) error {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	for name, p := range pr.plugins {
		if ra, ok := p.(RepositoryAware); ok {
			if err := ra.AttachRepository(repo); err != nil {
				return fmt.Errorf("failed to attach repository to plugin %s: %w", name, err)
			}
		}
	}
	return nil
}

// === Publishing Channel System ===

type PublishingChannel struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Type      string                 `json:"type"` // git, ipfs, rss, static, ftp, scp, etc
	Config    map[string]interface{} `json:"config"`
	Active    bool                   `json:"active"`
	CreatedAt int64                  `json:"created_at"`
}

type PublishJob struct {
	ID          string      `json:"id"`
	NodeID      string      `json:"node_id"`
	VersionID   string      `json:"version_id"`
	ChannelID   string      `json:"channel_id"`
	Status      string      `json:"status"` // queued, publishing, success, failed
	Progress    int         `json:"progress"`
	Result      interface{} `json:"result,omitempty"`
	Error       string      `json:"error,omitempty"`
	CreatedAt   int64       `json:"created_at"`
	CompletedAt *int64      `json:"completed_at,omitempty"`
}

// CredentialManager handles encrypted storage of API keys
type CredentialManager struct {
	credentials map[string][]byte // Will be encrypted in production
	mu          sync.RWMutex
}

var credentialMgr *CredentialManager

func initCredentialManager() {
	credentialMgr = &CredentialManager{
		credentials: make(map[string][]byte),
	}
}

// GetCredentialManager returns the global credential manager
func GetCredentialManager() *CredentialManager {
	if credentialMgr == nil {
		initCredentialManager()
	}
	return credentialMgr
}

func (cm *CredentialManager) StoreCredential(key string, value string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// In production, encrypt this
	cm.credentials[key] = []byte(value)
	return nil
}

func (cm *CredentialManager) GetCredential(key string) (string, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	value, exists := cm.credentials[key]
	if !exists {
		return "", fmt.Errorf("credential not found: %s", key)
	}

	return string(value), nil
}

func (cm *CredentialManager) DeleteCredential(key string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	delete(cm.credentials, key)
	return nil
}

// === Configuration Storage ===

type Config struct {
	ID    string      `json:"id"`
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}
