package plugins

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

var db *sql.DB

// SetDB sets the package-level database handle used by plugin helpers
func SetDB(d *sql.DB) {
	db = d
}

// === Plugin Initialization ===

func initializeDefaultPlugins() {
	registry := GetRegistry()

	// Git
	gitPlugin := NewGitPlugin()
	if err := registry.Register(gitPlugin); err != nil {
		log.Println("Git plugin registration:", err)
	}

	// IPFS
	ipfsPlugin := NewIPFSPlugin("http://localhost:5001")
	if err := registry.Register(ipfsPlugin); err != nil {
		log.Println("IPFS plugin registration:", err)
	}

	// Namecheap
	ncPlugin := NewNamecheapPlugin()
	if err := registry.Register(ncPlugin); err != nil {
		log.Println("Namecheap plugin registration:", err)
	}

	// Media
	mediaPlugin := NewMediaPlugin("./media_output")
	if err := registry.Register(mediaPlugin); err != nil {
		log.Println("Media plugin registration:", err)
	}

	// Pixospritz
	pixoPlugin := NewPixospritzPlugin("http://localhost:3000")
	if err := registry.Register(pixoPlugin); err != nil {
		log.Println("Pixospritz plugin registration:", err)
	}

	// Terminal Scripting
	terminalPlugin := NewTerminalScriptingPlugin()
	if err := registry.Register(terminalPlugin); err != nil {
		log.Println("Terminal scripting plugin registration:", err)
	}
	if err := terminalPlugin.Initialize(map[string]interface{}{"safe_mode": true}); err != nil {
		log.Println("Terminal scripting plugin initialization:", err)
	}

	// Reminder
	reminderPlugin := NewReminderPlugin()
	if err := registry.Register(reminderPlugin); err != nil {
		log.Println("Reminder plugin registration:", err)
	}
	if err := reminderPlugin.Initialize(map[string]interface{}{}); err != nil {
		log.Println("Reminder plugin initialization:", err)
	}

	fmt.Println("✓ All plugins registered successfully")
}

// === Plugin API Endpoints ===

// HandlePluginsList handles the plugins list endpoint
func HandlePluginsList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	plugins := GetRegistry().ListPlugins()
	json.NewEncoder(w).Encode(map[string]interface{}{
		"plugins": plugins,
	})
}

// HandlePluginExecute handles plugin execution endpoint
func HandlePluginExecute(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req map[string]interface{}
	json.NewDecoder(r.Body).Decode(&req)

	pluginName := req["plugin"].(string)
	action := req["action"].(string)
	payload := req["payload"]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := GetRegistry().Execute(ctx, pluginName, action, payload)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(result)
}

// === Publishing Channel Endpoints (Enhanced) ===

// Note: handlePublishingChannels and handlePublishHistory are in main.go
// These helper functions support the main handlers

func publishToGit(ctx context.Context, job PublishJob, config map[string]interface{}) (interface{}, error) {
	registry := GetRegistry()
	// Use Git plugin
	result, err := registry.Execute(ctx, "git", "commit", map[string]interface{}{
		"node_id": job.NodeID,
		"message": fmt.Sprintf("Published version %s", job.VersionID),
	})

	if err == nil {
		// Push
		registry.Execute(ctx, "git", "push", map[string]interface{}{
			"message": "Auto-sync published version",
			"branch":  "main",
		})
	}

	return result, err
}

func publishToIPFS(ctx context.Context, job PublishJob, config map[string]interface{}) (interface{}, error) {
	// Use IPFS plugin
	return pluginRegistry.Execute(ctx, "ipfs", "publish", map[string]interface{}{
		"node_id":    job.NodeID,
		"version_id": job.VersionID,
	})
}

func publishToRSS(ctx context.Context, job PublishJob, config map[string]interface{}) (interface{}, error) {
	// Generate RSS feed
	var blog BlogPost
	db.QueryRow(`SELECT id, slug, excerpt FROM blog_posts WHERE node_id = ?`, job.NodeID).
		Scan(&blog.ID, &blog.Slug, &blog.Excerpt)

	return map[string]string{
		"status": "rss_updated",
		"slug":   blog.Slug,
	}, nil
}

func publishAsStatic(ctx context.Context, job PublishJob, config map[string]interface{}) (interface{}, error) {
	// Export as static HTML
	result, _ := handleExportForJob(job.NodeID, "html")
	return result, nil
}

func handleExportForJob(nodeID, format string) (interface{}, error) {
	var node Node
	db.QueryRow(`SELECT id, title, content FROM nodes WHERE id = ?`, nodeID).
		Scan(&node.ID, &node.Title, &node.Content)

	if format == "html" {
		html := `<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<title>` + node.Title + `</title>
<style>
body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto; max-width: 800px; margin: 0 auto; padding: 20px; }
h1 { border-bottom: 2px solid #333; }
</style>
</head>
<body>
<h1>` + node.Title + `</h1>
<div>` + markdownToHTML(node.Content) + `</div>
</body>
</html>`
		return map[string]string{"html": html}, nil
	}

	return nil, fmt.Errorf("unsupported format")
}

func HandleCredentialsAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == "POST" {
		var cred map[string]string
		json.NewDecoder(r.Body).Decode(&cred)

		key := cred["key"]
		value := cred["value"]

		credentialMgr.StoreCredential(key, value)

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"stored": key})
	}
}

func HandlePublishJob(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == "POST" {
		var job PublishJob
		json.NewDecoder(r.Body).Decode(&job)
		job.ID = fmt.Sprintf("job_%d", time.Now().UnixNano())
		job.CreatedAt = time.Now().Unix()
		job.Status = "queued"

		db.Exec(`
			INSERT INTO publish_jobs (id, node_id, version_id, channel_id, status, progress, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, job.ID, job.NodeID, job.VersionID, job.ChannelID, job.Status, 0, job.CreatedAt)

		go processPublishJob(job)

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(job)
	}
}

// Instantiate known plugins by slug. Returns nil if the slug is unknown or instantiation fails.
func InstantiatePluginBySlug(slug string) Plugin {
	switch slug {
	case "git":
		return NewGitPlugin()
	case "ipfs":
		return NewIPFSPlugin("http://localhost:5001")
	case "namecheap":
		return NewNamecheapPlugin()
	case "media":
		return NewMediaPlugin("./media_output")
	case "pixospritz":
		return NewPixospritzPlugin("http://localhost:3000")
	case "shader":
		return NewShaderPlugin()
	case "svg":
		return NewSVGPlugin()
	case "code":
		// Code plugin lives in main package; not available here
		return nil
	case "todo":
		return NewTodoPlugin()
	case "reminder":
		return NewReminderPlugin()
	case "terminal":
		return NewTerminalScriptingPlugin()
	default:
		return nil
	}
}

// PopulatePluginsRegistry ensures all known plugins are in the plugins_registry table
func PopulatePluginsRegistry(db *sql.DB) {
	// List of all known plugins with their metadata
	knownPlugins := []struct {
		name string
		slug string
	}{
		{"Git", "git"},
		{"IPFS", "ipfs"},
		{"Namecheap", "namecheap"},
		{"Media", "media"},
		{"Pixospritz", "pixospritz"},
		{"Shader", "shader"},
		{"SVG", "svg"},
		{"Code", "code"},
		{"Todo", "todo"},
		{"Reminder", "reminder"},
		{"Terminal Scripting", "terminal"},
	}

	now := time.Now().Unix()
	for _, plugin := range knownPlugins {
		// Check if plugin already exists
		var count int
		db.QueryRow(`SELECT COUNT(*) FROM plugins_registry WHERE slug = ?`, plugin.slug).Scan(&count)
		if count == 0 {
			// Insert with enabled=0
			id := fmt.Sprintf("plugin_%d", time.Now().UnixNano())
			_, err := db.Exec(`INSERT INTO plugins_registry (id, name, slug, manifest, enabled, created_at, updated_at) VALUES (?, ?, ?, ?, 0, ?, ?)`,
				id, plugin.name, plugin.slug, "", now, now)
			if err != nil {
				log.Printf("Failed to insert plugin %s: %v\n", plugin.slug, err)
			} else {
				log.Printf("Added plugin to registry: %s (%s)\n", plugin.name, plugin.slug)
			}
		}
	}
}

// LoadEnabledPluginsFromDB loads enabled plugins from the DB and register them into the runtime registry
func LoadEnabledPluginsFromDB(db *sql.DB) {
	// store DB handle for plugin helpers
	SetDB(db)
	rows, err := db.Query(`SELECT id, name, slug, manifest FROM plugins_registry WHERE enabled = 1`)
	if err != nil {
		log.Println("Failed to load enabled plugins from DB:", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id, name, slug, manifest string
		rows.Scan(&id, &name, &slug, &manifest)
		p := InstantiatePluginBySlug(slug)
		if p == nil {
			log.Printf("No runtime plugin implementation for slug: %s\n", slug)
			continue
		}
		// Attempt to initialize with manifest JSON if present
		var cfg map[string]interface{}
		if manifest != "" {
			json.Unmarshal([]byte(manifest), &cfg)
		}
		if err := p.Initialize(cfg); err != nil {
			log.Printf("Failed to initialize plugin %s: %v\n", slug, err)
			continue
		}
		if err := pluginRegistry.Register(p); err != nil {
			log.Printf("Failed to register plugin %s: %v\n", slug, err)
			continue
		}
		log.Printf("Registered plugin from DB: %s (%s)\n", name, slug)
	}
}

// === Processing Functions ===

func processPublishJob(job PublishJob) {
	// Update to publishing status
	db.Exec(`UPDATE publish_jobs SET status = 'publishing', progress = 10 WHERE id = ?`, job.ID)

	// Get channel
	var channel PublishingChannel
	var configJSON sql.NullString
	db.QueryRow(`SELECT type, config FROM publishing_channels WHERE id = ?`, job.ChannelID).
		Scan(&channel.Type, &configJSON)

	if configJSON.Valid {
		json.Unmarshal([]byte(configJSON.String), &channel.Config)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	var result interface{}
	var err error

	switch channel.Type {
	case "git":
		result, err = publishToGit(ctx, job, channel.Config)
	case "ipfs":
		result, err = publishToIPFS(ctx, job, channel.Config)
	case "rss":
		result, err = publishToRSS(ctx, job, channel.Config)
	case "static":
		result, err = publishAsStatic(ctx, job, channel.Config)
	default:
		err = fmt.Errorf("unknown channel type: %s", channel.Type)
	}

	status := "success"
	errorMsg := ""
	if err != nil {
		status = "failed"
		errorMsg = err.Error()
	}

	// Update job status
	resultJSON, _ := json.Marshal(result)
	now := time.Now().Unix()
	db.Exec(`
		UPDATE publish_jobs 
		SET status = ?, progress = 100, result = ?, error = ?, completed_at = ?
		WHERE id = ?
	`, status, string(resultJSON), errorMsg, now, job.ID)
}

// Simple plugin-local types and helpers to avoid tight coupling with main
type BlogPost struct {
	ID      string
	Slug    string
	Excerpt string
}

type Node struct {
	ID      string
	Title   string
	Path    string
	Content string
}

type Version struct {
	ID      string
	NodeID  string
	Content string
	Title   string
}

func markdownToHTML(s string) string {
	// Minimal conversion placeholder for plugin usage
	return s
}

// Configuration helpers used by plugins
func saveConfig(key string, value interface{}) error {
	if db == nil {
		return fmt.Errorf("db not initialized for plugins")
	}
	configID := fmt.Sprintf("config_%s", key)
	now := int64(0)
	_, err := db.Exec(`INSERT OR REPLACE INTO configs (id, key, value, updated_at) VALUES (?, ?, ?, ?)`, configID, key, fmt.Sprintf("%v", value), now)
	return err
}

func loadConfig(key string) (interface{}, error) {
	if db == nil {
		return nil, fmt.Errorf("db not initialized for plugins")
	}
	var value string
	err := db.QueryRow(`SELECT value FROM configs WHERE key = ?`, key).Scan(&value)
	if err != nil {
		return nil, err
	}
	return value, nil
}
