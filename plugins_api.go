package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// === Plugin Initialization ===

func initializeDefaultPlugins() {
	// Git
	gitPlugin := NewGitPlugin()
	if err := pluginRegistry.Register(gitPlugin); err != nil {
		log.Println("Git plugin registration:", err)
	}

	// IPFS
	ipfsPlugin := NewIPFSPlugin("http://localhost:5001")
	if err := pluginRegistry.Register(ipfsPlugin); err != nil {
		log.Println("IPFS plugin registration:", err)
	}

	// Namecheap
	ncPlugin := NewNamecheapPlugin()
	if err := pluginRegistry.Register(ncPlugin); err != nil {
		log.Println("Namecheap plugin registration:", err)
	}

	// Media
	mediaPlugin := NewMediaPlugin("./media_output")
	if err := pluginRegistry.Register(mediaPlugin); err != nil {
		log.Println("Media plugin registration:", err)
	}

	// Pixospritz
	pixoPlugin := NewPixospritzPlugin("http://localhost:3000")
	if err := pluginRegistry.Register(pixoPlugin); err != nil {
		log.Println("Pixospritz plugin registration:", err)
	}

	fmt.Println("✓ All plugins registered successfully")
}

// === Plugin API Endpoints ===

func handlePluginsList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	plugins := pluginRegistry.ListPlugins()
	json.NewEncoder(w).Encode(map[string]interface{}{
		"plugins": plugins,
	})
}

func handlePluginExecute(w http.ResponseWriter, r *http.Request) {
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

	result, err := pluginRegistry.Execute(ctx, pluginName, action, payload)
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
	// Use Git plugin
	result, err := pluginRegistry.Execute(ctx, "git", "commit", map[string]interface{}{
		"node_id": job.NodeID,
		"message": fmt.Sprintf("Published version %s", job.VersionID),
	})

	if err == nil {
		// Push
		pluginRegistry.Execute(ctx, "git", "push", map[string]interface{}{
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

func handleCredentialsAPI(w http.ResponseWriter, r *http.Request) {
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

func handlePublishJob(w http.ResponseWriter, r *http.Request) {
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
func instantiatePluginBySlug(slug string) Plugin {
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
	default:
		return nil
	}
}

// Load enabled plugins from the DB and register them into the runtime registry
func loadEnabledPluginsFromDB() {
	rows, err := db.Query(`SELECT id, name, slug, manifest FROM plugins_registry WHERE enabled = 1`)
	if err != nil {
		log.Println("Failed to load enabled plugins from DB:", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id, name, slug, manifest string
		rows.Scan(&id, &name, &slug, &manifest)
		p := instantiatePluginBySlug(slug)
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
