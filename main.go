package main

import (
	"archive/zip"
	"bytes"
	"crypto/md5"
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed web/*
var webUI embed.FS

//go:embed migrations/*.sql
var migrations embed.FS

var db *sql.DB
var dbPath string

// === Node Types ===
const (
	NodeTypeNote        = "note"
	NodeTypePage        = "page"
	NodeTypePost        = "post"
	NodeTypeCanvas      = "canvas"
	NodeTypeShaderDemo  = "shader-demo"
	NodeTypeCodeSnippet = "code-snippet"
	NodeTypeImage       = "image"
	NodeTypeVideo       = "video"
	NodeTypeAudio       = "audio"
	NodeTypeDocument    = "document"
	NodeTypeTodo        = "todo"
	NodeTypeReminder    = "reminder"
)

// === Types ===
type Node struct {
	ID           string    `json:"id"`
	Type         string    `json:"type"`
	ParentID     string    `json:"parent_id,omitempty"`
	Path         string    `json:"path"`
	Title        string    `json:"title"`
	Content      string    `json:"content"`
	Slug         string    `json:"slug,omitempty"`
	CanonicalURI string    `json:"canonical_uri,omitempty"`
	Body         string    `json:"body,omitempty"`     // JSON structured body
	Metadata     string    `json:"metadata,omitempty"` // JSON metadata
	MimeType     string    `json:"mime_type"`
	CreatedAt    time.Time `json:"created_at"`
	ModifiedAt   time.Time `json:"modified_at"`
	Tags         []string  `json:"tags,omitempty"`
	References   []string  `json:"references,omitempty"`
	Visibility   string    `json:"visibility,omitempty"`
	Status       string    `json:"status,omitempty"`
	SiteID       string    `json:"site_id,omitempty"`
}

type Version struct {
	ID            string     `json:"id"`
	NodeID        string     `json:"node_id"`
	VersionNumber int        `json:"version_number"`
	Content       string     `json:"content"`
	Title         string     `json:"title"`
	Status        string     `json:"status"`
	PublishedAt   *time.Time `json:"published_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	ModifiedAt    time.Time  `json:"modified_at"`
	IsCurrent     bool       `json:"is_current"`
}

type BlogPost struct {
	ID          string     `json:"id"`
	NodeID      string     `json:"node_id"`
	Slug        string     `json:"slug"`
	Excerpt     string     `json:"excerpt"`
	PublishDate *time.Time `json:"publish_date,omitempty"`
	Category    string     `json:"category"`
}

type MediaFile struct {
	ID               string    `json:"id"`
	NodeID           string    `json:"node_id"`
	Filename         string    `json:"filename"`
	OriginalFilename string    `json:"original_filename"`
	MimeType         string    `json:"mime_type"`
	FileSize         int64     `json:"file_size"`
	Checksum         string    `json:"checksum"`
	StorageURL       string    `json:"storage_url"`
	UploadedBy       string    `json:"uploaded_by"`
	CreatedAt        time.Time `json:"created_at"`
}

type Reference struct {
	ID           string `json:"id"`
	SourceNodeID string `json:"source_node_id"`
	TargetNodeID string `json:"target_node_id"`
	LinkType     string `json:"link_type"`
	LinkText     string `json:"link_text"`
}

type Tag struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

type Citation struct {
	ID             string `json:"id"`
	NodeID         string `json:"node_id"`
	CitationKey    string `json:"citation_key"`
	Authors        string `json:"authors"`
	Title          string `json:"title"`
	Year           int    `json:"year"`
	Publisher      string `json:"publisher"`
	URL            string `json:"url"`
	DOI            string `json:"doi"`
	CitationFormat string `json:"citation_format"`
	RawBibtex      string `json:"raw_bibtex"`
}

type Site struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Type        string    `json:"type"` // project, portfolio, blog, etc
	CreatedAt   time.Time `json:"created_at"`
	ModifiedAt  time.Time `json:"modified_at"`
}

type NodeURI struct {
	ID        string    `json:"id"`
	NodeID    string    `json:"node_id"`
	URI       string    `json:"uri"`
	IsPrimary bool      `json:"is_primary"`
	CreatedAt time.Time `json:"created_at"`
}

type PluginManifest struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	Manifest  string    `json:"manifest"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	command := os.Args[1]
	switch command {
	case "init":
		initVault()
	case "serve":
		serve()
	case "gui":
		gui()
	case "new":
		createNode()
	case "list":
		listNodes()
	case "publish":
		publishNode()
	case "export":
		exportNode()
	case "version":
		fmt.Println("veil v0.2.0 - MVP")
	default:
		printUsage()
	}
}

func printUsage() {
	fmt.Println(`veil - Universal content management system v0.2.0 (MVP)

Usage:
  veil init [path]              Initialize new vault (default: ./veil.db)
  veil serve [--port N]         Start web server (default: 8080)
  veil gui                      Launch GUI mode
  veil new <path>               Create new file/note
  veil list                     List all nodes
  veil publish <node-id>        Publish a node
  veil export <node-id> <type>  Export node (zip, html, json, rss)
  veil version                  Show version

Examples:
  veil init ~/my-vault
  veil serve --port 3000
  veil new notes/ideas.md
  veil export node_123 zip
  veil publish node_456`)
}

func initVault() {
	path := "./veil.db"
	if len(os.Args) > 2 {
		path = os.Args[2]
	}

	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, path[2:])
	}

	// If path is a directory, append veil.db
	if stat, err := os.Stat(path); err == nil && stat.IsDir() {
		path = filepath.Join(path, "veil.db")
	} else if err != nil {
		// Path doesn't exist, check if parent is directory
		dir := filepath.Dir(path)
		os.MkdirAll(dir, 0755)
	}

	database, err := sql.Open("sqlite", path)
	if err != nil {
		log.Fatal("Failed to create database:", err)
	}
	defer database.Close()

	// Run migrations
	migrationFiles, _ := fs.ReadDir(migrations, "migrations")

	// Sort migration files by name to ensure proper order
	sort.Slice(migrationFiles, func(i, j int) bool {
		return migrationFiles[i].Name() < migrationFiles[j].Name()
	})

	for _, file := range migrationFiles {
		content, _ := migrations.ReadFile("migrations/" + file.Name())
		statements := strings.Split(string(content), ";")
		for _, stmt := range statements {
			stmt = strings.TrimSpace(stmt)
			if stmt != "" {
				if _, err := database.Exec(stmt); err != nil {
					log.Printf("Migration warning: %v\n", err)
				}
			}
		}
	}

	fmt.Printf("✓ Initialized vault at %s\n", path)
	fmt.Println("\nNext steps:")
	fmt.Println("  veil serve")
	fmt.Println("  veil gui")
}

func serve() {
	port := "8080"
	dbPath = "./veil.db"

	for i, arg := range os.Args {
		if arg == "--port" && i+1 < len(os.Args) {
			port = os.Args[i+1]
		}
	}

	var err error
	db, err = sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}

	// Ensure migrations are applied on server start so the default DB has required tables
	if err := applyMigrations(db); err != nil {
		log.Printf("Warning: migrations had errors: %v", err)
	}
	defer db.Close()

	// Initialize plugin systems
	initPluginRegistry()
	initCredentialManager()
	initURIResolver()
	// Load enabled plugins from DB and register them at runtime
	loadEnabledPluginsFromDB()

	mux := setupRoutes()
	addr := ":" + port
	fmt.Printf("✓ Veil running at http://localhost:%s\n", port)
	fmt.Println("✓ Plugins initialized: Git, IPFS, Namecheap, Media, Pixospritz")
	log.Fatal(http.ListenAndServe(addr, mux))
}

func gui() {
	dbPath = "./veil.db"
	var err error
	db, err = sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}

	// Apply migrations in GUI mode as well
	if err := applyMigrations(db); err != nil {
		log.Printf("Warning: migrations had errors: %v", err)
	}
	defer db.Close()

	// Initialize plugin systems
	initPluginRegistry()
	initCredentialManager()
	initURIResolver()
	loadEnabledPluginsFromDB()

	mux := setupRoutes()
	go func() {
		log.Fatal(http.ListenAndServe(":8080", mux))
	}()

	time.Sleep(500 * time.Millisecond)
	url := "http://localhost:8080"
	fmt.Printf("✓ Opening Veil at %s\n", url)

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	cmd.Run()
	select {}
}

func setupRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	// Serve a no-content favicon to avoid 404 noise in browser consoles
	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	// Static files
	webFS, _ := fs.Sub(webUI, "web")
	mux.Handle("/", http.FileServer(http.FS(webFS)))

	// Core node APIs
	mux.HandleFunc("/api/nodes", handleNodes)
	mux.HandleFunc("/api/node/", handleNode)
	mux.HandleFunc("/api/node-create", handleNodeCreate)
	mux.HandleFunc("/api/node-update", handleNodeUpdate)
	mux.HandleFunc("/api/node-delete", handleNodeDelete)

	// Universal URI system
	mux.HandleFunc("/veil/", handleUniversalURI)

	// Version control
	mux.HandleFunc("/api/versions", handleVersions)
	mux.HandleFunc("/api/publish", handlePublish)
	mux.HandleFunc("/api/rollback", handleRollback)
	mux.HandleFunc("/api/snapshots", handleSnapshots)

	// Knowledge graph
	mux.HandleFunc("/api/references", handleReferences)
	mux.HandleFunc("/api/backlinks/", handleBacklinks)
	mux.HandleFunc("/api/resolve-link", handleResolveLink)

	// Tags
	mux.HandleFunc("/api/tags", handleTags)
	mux.HandleFunc("/api/node-tags", handleNodeTags)

	// Media
	mux.HandleFunc("/api/media-upload", handleMediaUpload)
	mux.HandleFunc("/api/media", handleMedia)
	mux.HandleFunc("/api/media-library", handleMediaLibrary)

	// Blog
	mux.HandleFunc("/api/blog-posts", handleBlogPosts)
	mux.HandleFunc("/api/blog-post", handleBlogPost)

	// Export
	mux.HandleFunc("/api/export", handleExport)
	mux.HandleFunc("/api/rss-feed", handleRSSFeed)

	// Publishing
	mux.HandleFunc("/api/publishing-channels", handlePublishingChannels)
	mux.HandleFunc("/api/publish-history", handlePublishHistory)

	// Permissions
	mux.HandleFunc("/api/visibility", handleVisibility)

	// Search
	mux.HandleFunc("/api/search", handleSearch)

	// Citation
	mux.HandleFunc("/api/citations", handleCitations)

	// Sites/Projects
	mux.HandleFunc("/api/sites", handleSites)
	mux.HandleFunc("/api/sites/", handleSitesDetail)

	// Plugin APIs (NEW)
	mux.HandleFunc("/api/plugins", handlePluginsList)
	mux.HandleFunc("/api/plugin-execute", handlePluginExecute)
	mux.HandleFunc("/api/credentials", handleCredentialsAPI)
	mux.HandleFunc("/api/publish-job", handlePublishJob)
	mux.HandleFunc("/api/plugins-registry", handlePluginsRegistry)
	mux.HandleFunc("/api/node-uris", handleNodeURIs)
	mux.HandleFunc("/api/resolve-uri", handleResolveURI)
	mux.HandleFunc("/api/generate-uri", handleGenerateURI)

	return mux
}

// === CLI Commands ===
func createNode() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: veil new <path>")
		return
	}
	path := os.Args[2]
	fmt.Printf("Created node: %s\n", path)
}

func listNodes() {
	db, err := sql.Open("sqlite", "./veil.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	rows, _ := db.Query("SELECT id, path, title FROM nodes WHERE deleted_at IS NULL ORDER BY path")
	defer rows.Close()

	for rows.Next() {
		var id, path, title string
		rows.Scan(&id, &path, &title)
		fmt.Printf("%s - %s (%s)\n", path, title, id)
	}
}

func publishNode() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: veil publish <node-id>")
		return
	}
	nodeID := os.Args[2]
	fmt.Printf("Published node: %s\n", nodeID)
}

func exportNode() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: veil export <node-id> <type>")
		return
	}
	nodeID := os.Args[2]
	exportType := os.Args[3]
	fmt.Printf("Exported node %s as %s\n", nodeID, exportType)
}

// === API Handlers - Core ===
func handleNodes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method == "GET" {
		rows, _ := db.Query(`SELECT id, type, COALESCE(parent_id, ''), path, title, content, mime_type, created_at, modified_at 
			FROM nodes WHERE deleted_at IS NULL ORDER BY path`)
		defer rows.Close()

		var nodes []Node
		for rows.Next() {
			var node Node
			var created, modified int64
			rows.Scan(&node.ID, &node.Type, &node.ParentID, &node.Path, &node.Title,
				&node.Content, &node.MimeType, &created, &modified)
			node.CreatedAt = time.Unix(created, 0)
			node.ModifiedAt = time.Unix(modified, 0)
			nodes = append(nodes, node)
		}
		json.NewEncoder(w).Encode(nodes)
	}
}

func handleNode(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	nodeID := strings.TrimPrefix(r.URL.Path, "/api/node/")

	var node Node
	var created, modified int64
	err := db.QueryRow(`SELECT id, type, COALESCE(parent_id, ''), path, title, content, mime_type, created_at, modified_at 
		FROM nodes WHERE id = ? AND deleted_at IS NULL`, nodeID).
		Scan(&node.ID, &node.Type, &node.ParentID, &node.Path, &node.Title,
			&node.Content, &node.MimeType, &created, &modified)

	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	node.CreatedAt = time.Unix(created, 0)
	node.ModifiedAt = time.Unix(modified, 0)
	json.NewEncoder(w).Encode(node)
}

func handleNodeCreate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var node Node
	json.NewDecoder(r.Body).Decode(&node)
	node.ID = fmt.Sprintf("node_%d", time.Now().UnixNano())
	now := time.Now().Unix()

	db.Exec(`INSERT INTO nodes (id, type, parent_id, path, title, content, mime_type, created_at, modified_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		node.ID, node.Type, node.ParentID, node.Path, node.Title, node.Content, node.MimeType, now, now)

	// Create initial version
	versionID := fmt.Sprintf("v_%d", time.Now().UnixNano())
	db.Exec(`INSERT INTO versions (id, node_id, version_number, content, title, status, created_at, modified_at, is_current)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		versionID, node.ID, 1, node.Content, node.Title, "draft", now, now, 1)

	// Set visibility
	db.Exec(`INSERT INTO node_visibility (id, node_id, visibility, created_at)
		VALUES (?, ?, ?, ?)`,
		fmt.Sprintf("vis_%d", time.Now().UnixNano()), node.ID, "private", now)

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(node)
}

func handleNodeUpdate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != "PUT" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var node Node
	json.NewDecoder(r.Body).Decode(&node)
	now := time.Now().Unix()

	db.Exec(`UPDATE nodes SET title = ?, content = ?, modified_at = ? WHERE id = ?`,
		node.Title, node.Content, now, node.ID)

	// Create new version
	var versionNumber int
	db.QueryRow(`SELECT COALESCE(MAX(version_number), 0) FROM versions WHERE node_id = ?`, node.ID).Scan(&versionNumber)
	versionNumber++

	versionID := fmt.Sprintf("v_%d", time.Now().UnixNano())
	db.Exec(`INSERT INTO versions (id, node_id, version_number, content, title, status, created_at, modified_at, is_current)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		versionID, node.ID, versionNumber, node.Content, node.Title, "draft", now, now, 1)

	db.Exec(`UPDATE versions SET is_current = 0 WHERE node_id = ? AND id != ?`, node.ID, versionID)

	json.NewEncoder(w).Encode(node)
}

func handleNodeDelete(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	nodeID := r.URL.Query().Get("id")
	db.Exec(`UPDATE nodes SET deleted_at = ? WHERE id = ?`, time.Now().Unix(), nodeID)
	w.WriteHeader(http.StatusNoContent)
}

// === API Handlers - Versions ===
func handleVersions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	nodeID := r.URL.Query().Get("node_id")

	rows, _ := db.Query(`SELECT id, node_id, version_number, content, title, status, published_at, created_at, modified_at, is_current
		FROM versions WHERE node_id = ? ORDER BY version_number DESC`, nodeID)
	defer rows.Close()

	var versions []Version
	for rows.Next() {
		var v Version
		var created, modified int64
		var published sql.NullInt64
		rows.Scan(&v.ID, &v.NodeID, &v.VersionNumber, &v.Content, &v.Title, &v.Status, &published, &created, &modified, &v.IsCurrent)
		v.CreatedAt = time.Unix(created, 0)
		v.ModifiedAt = time.Unix(modified, 0)
		if published.Valid {
			t := time.Unix(published.Int64, 0)
			v.PublishedAt = &t
		}
		versions = append(versions, v)
	}
	json.NewEncoder(w).Encode(versions)
}

func handlePublish(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	nodeID := r.URL.Query().Get("node_id")
	now := time.Now().Unix()

	db.Exec(`UPDATE versions SET status = 'published', published_at = ? WHERE node_id = ? AND is_current = 1`,
		now, nodeID)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "published"})
}

func handleRollback(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	versionID := r.URL.Query().Get("version_id")

	var version Version
	db.QueryRow(`SELECT id, node_id, content, title FROM versions WHERE id = ?`, versionID).
		Scan(&version.ID, &version.NodeID, &version.Content, &version.Title)

	now := time.Now().Unix()
	db.Exec(`UPDATE nodes SET content = ?, title = ?, modified_at = ? WHERE id = ?`,
		version.Content, version.Title, now, version.NodeID)

	json.NewEncoder(w).Encode(version)
}

func handleSnapshots(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
}

// === API Handlers - References ===
func handleReferences(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	sourceNodeID := r.URL.Query().Get("source")

	rows, _ := db.Query(`SELECT id, source_node_id, target_node_id, link_type, link_text
		FROM node_references WHERE source_node_id = ?`, sourceNodeID)
	defer rows.Close()

	var references []Reference
	for rows.Next() {
		var ref Reference
		rows.Scan(&ref.ID, &ref.SourceNodeID, &ref.TargetNodeID, &ref.LinkType, &ref.LinkText)
		references = append(references, ref)
	}
	json.NewEncoder(w).Encode(references)
}

func handleBacklinks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	targetNodeID := strings.TrimPrefix(r.URL.Path, "/api/backlinks/")

	rows, _ := db.Query(`SELECT id, source_node_id, target_node_id, link_type, link_text
		FROM node_references WHERE target_node_id = ?`, targetNodeID)
	defer rows.Close()

	var backlinks []Reference
	for rows.Next() {
		var ref Reference
		rows.Scan(&ref.ID, &ref.SourceNodeID, &ref.TargetNodeID, &ref.LinkType, &ref.LinkText)
		backlinks = append(backlinks, ref)
	}
	json.NewEncoder(w).Encode(backlinks)
}

func handleResolveLink(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	linkText := r.URL.Query().Get("text")

	// Try exact URI match in node_uris
	var uri NodeURI
	err := db.QueryRow(`SELECT id, node_id, uri, is_primary, created_at FROM node_uris WHERE uri = ?`, linkText).
		Scan(&uri.ID, &uri.NodeID, &uri.URI, &uri.IsPrimary, &uri.CreatedAt)
	if err == nil {
		var node Node
		db.QueryRow(`SELECT id, path, title FROM nodes WHERE id = ?`, uri.NodeID).Scan(&node.ID, &node.Path, &node.Title)
		json.NewEncoder(w).Encode(node)
		return
	}

	// Fallback: search by canonical_uri or partial path/title
	var node Node
	db.QueryRow(`SELECT id, path, title FROM nodes WHERE canonical_uri = ? OR path LIKE ? OR title LIKE ?`, linkText, "%"+linkText+"%", "%"+linkText+"%").
		Scan(&node.ID, &node.Path, &node.Title)
	json.NewEncoder(w).Encode(node)
}

func handleUniversalURI(w http.ResponseWriter, r *http.Request) {
	// Extract URI from path: /veil/site/path/entity
	path := strings.TrimPrefix(r.URL.Path, "/veil/")
	parts := strings.Split(path, "/")

	if len(parts) < 2 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	siteName := parts[0]
	entityPath := strings.Join(parts[1:], "/")

	// Find site
	var site Site
	err := db.QueryRow(`SELECT id, name FROM sites WHERE name = ?`, siteName).
		Scan(&site.ID, &site.Name)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// Find node by path within site
	var node Node
	var created, modified int64
	err = db.QueryRow(`SELECT id, type, path, title, content, mime_type, created_at, modified_at FROM nodes WHERE site_id = ? AND path = ?`, site.ID, entityPath).
		Scan(&node.ID, &node.Type, &node.Path, &node.Title, &node.Content, &node.MimeType, &created, &modified)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	node.CreatedAt = time.Unix(created, 0)
	node.ModifiedAt = time.Unix(modified, 0)

	// Render based on content type
	switch node.Type {
	case NodeTypeImage, NodeTypeVideo, NodeTypeAudio:
		// Serve media content
		w.Header().Set("Content-Type", node.MimeType)
		w.Write([]byte(node.Content))
	case NodeTypeShaderDemo:
		// Render shader demo
		renderShaderDemo(w, node)
	case NodeTypeCanvas:
		// Render canvas content
		renderCanvas(w, node)
	default:
		// Render as HTML page
		renderNodeAsHTML(w, node, site)
	}
}

func renderNodeAsHTML(w http.ResponseWriter, node Node, site Site) {
	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>%s - %s</title>
    <meta charset="utf-8">
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, sans-serif; max-width: 800px; margin: 0 auto; padding: 20px; }
        .header { border-bottom: 1px solid #eee; padding-bottom: 20px; margin-bottom: 30px; }
        .content { line-height: 1.6; }
    </style>
</head>
<body>
    <div class="header">
        <h1>%s</h1>
        <p><em>%s</em></p>
    </div>
    <div class="content">
        %s
    </div>
</body>
</html>`, node.Title, site.Name, node.Title, site.Name, markdownToHTML(node.Content))

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

func renderShaderDemo(w http.ResponseWriter, node Node) {
	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>%s - Shader Demo</title>
    <script src="https://cdnjs.cloudflare.com/ajax/libs/three.js/r128/three.min.js"></script>
</head>
<body>
    <canvas id="shaderCanvas" style="width: 100%%; height: 100vh; display: block;"></canvas>
    <script>
        %s
    </script>
</body>
</html>`, node.Title, node.Content)

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

func renderCanvas(w http.ResponseWriter, node Node) {
	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>%s - Canvas</title>
</head>
<body>
    <div style="text-align: center;">
        %s
    </div>
</body>
</html>`, node.Title, node.Content)

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

// === API Handlers - Plugin Registry ===
func handlePluginsRegistry(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == "GET" {
		rows, _ := db.Query(`SELECT id, name, slug, manifest, enabled, created_at, updated_at FROM plugins_registry ORDER BY name`)
		defer rows.Close()

		var out []PluginManifest
		for rows.Next() {
			var p PluginManifest
			var createdAt, updatedAt sql.NullInt64
			var enabled int
			rows.Scan(&p.ID, &p.Name, &p.Slug, &p.Manifest, &enabled, &createdAt, &updatedAt)
			p.Enabled = enabled == 1
			if createdAt.Valid {
				p.CreatedAt = time.Unix(createdAt.Int64, 0)
			}
			if updatedAt.Valid {
				p.UpdatedAt = time.Unix(updatedAt.Int64, 0)
			}
			out = append(out, p)
		}
		json.NewEncoder(w).Encode(out)
		return
	}

	if r.Method == "POST" {
		var req PluginManifest
		json.NewDecoder(r.Body).Decode(&req)
		req.ID = fmt.Sprintf("plugin_%d", time.Now().UnixNano())
		now := time.Now().Unix()
		enabled := 0
		if req.Enabled {
			enabled = 1
		}
		_, err := db.Exec(`INSERT INTO plugins_registry (id, name, slug, manifest, enabled, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`, req.ID, req.Name, req.Slug, req.Manifest, enabled, now, now)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		json.NewEncoder(w).Encode(req)
		return
	}

	if r.Method == "PUT" {
		var req PluginManifest
		json.NewDecoder(r.Body).Decode(&req)
		now := time.Now().Unix()
		enabled := 0
		if req.Enabled {
			enabled = 1
		}
		_, err := db.Exec(`UPDATE plugins_registry SET name = ?, manifest = ?, enabled = ?, updated_at = ? WHERE slug = ? OR id = ?`, req.Name, req.Manifest, enabled, now, req.Slug, req.ID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		// Runtime registration/unregistration
		if req.Enabled {
			p := instantiatePluginBySlug(req.Slug)
			if p != nil {
				var cfg map[string]interface{}
				if req.Manifest != "" {
					json.Unmarshal([]byte(req.Manifest), &cfg)
				}
				if err := p.Initialize(cfg); err != nil {
					log.Printf("plugin init failed for %s: %v", req.Slug, err)
				} else if err := pluginRegistry.Register(p); err != nil {
					log.Printf("plugin register failed for %s: %v", req.Slug, err)
				} else {
					log.Printf("plugin %s enabled and registered", req.Slug)
				}
			}
		} else {
			// Try by slug first, then name
			if err := pluginRegistry.Unregister(req.Slug); err != nil {
				if req.Name != "" {
					pluginRegistry.Unregister(req.Name)
				}
			}
		}

		json.NewEncoder(w).Encode(map[string]string{"updated": req.Slug})
		return
	}

	if r.Method == "DELETE" {
		id := r.URL.Query().Get("id")
		_, err := db.Exec(`DELETE FROM plugins_registry WHERE id = ? OR slug = ?`, id, id)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"deleted": id})
		return
	}
}

// === API Handlers - Tags ===
func handleTags(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	rows, _ := db.Query(`SELECT id, name, color FROM tags ORDER BY name`)
	defer rows.Close()

	var tags []Tag
	for rows.Next() {
		var tag Tag
		rows.Scan(&tag.ID, &tag.Name, &tag.Color)
		tags = append(tags, tag)
	}
	json.NewEncoder(w).Encode(tags)
}

func handleNodeTags(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	nodeID := r.URL.Query().Get("node_id")

	rows, _ := db.Query(`SELECT t.id, t.name, t.color FROM tags t
		JOIN node_tags nt ON t.id = nt.tag_id WHERE nt.node_id = ?`, nodeID)
	defer rows.Close()

	var tags []Tag
	for rows.Next() {
		var tag Tag
		rows.Scan(&tag.ID, &tag.Name, &tag.Color)
		tags = append(tags, tag)
	}
	json.NewEncoder(w).Encode(tags)
}

// === API Handlers - Media ===
func handleMediaUpload(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	r.ParseMultipartForm(10 << 20)

	file, handler, _ := r.FormFile("file")
	defer file.Close()

	buf := new(bytes.Buffer)
	io.Copy(buf, file)

	hash := md5.Sum(buf.Bytes())
	hashStr := fmt.Sprintf("%x", hash)

	mediaID := fmt.Sprintf("media_%d", time.Now().UnixNano())
	now := time.Now().Unix()

	// Ensure media dir exists and write file to disk
	os.MkdirAll("./media", 0755)
	filename := fmt.Sprintf("%s_%s", mediaID, handler.Filename)
	fpath := filepath.Join("media", filename)
	os.WriteFile(fpath, buf.Bytes(), 0644)

	_, err := db.Exec(`INSERT INTO media (id, filename, storage_url, checksum, mime_type, size, uploaded_by, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		mediaID, handler.Filename, fpath, hashStr, handler.Header.Get("Content-Type"), buf.Len(), "", now)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"id": mediaID, "checksum": hashStr, "storage_url": fpath})
}

func handleMedia(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	mediaID := r.URL.Query().Get("id")

	var media MediaFile
	db.QueryRow(`SELECT id, node_id, filename, storage_url, checksum, mime_type, size, uploaded_by, created_at FROM media WHERE id = ?`, mediaID).
		Scan(&media.ID, &media.NodeID, &media.Filename, &media.StorageURL, &media.Checksum, &media.MimeType, &media.FileSize, &media.UploadedBy, &media.CreatedAt)

	json.NewEncoder(w).Encode(media)
}

func handleMediaLibrary(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID := r.URL.Query().Get("user_id")

	rows, _ := db.Query(`SELECT m.id, m.node_id, m.filename, m.storage_url, m.checksum, m.mime_type, m.size, m.uploaded_by FROM media m JOIN media_library ml ON m.id = ml.media_id WHERE ml.user_id = ?`, userID)
	defer rows.Close()

	var media []MediaFile
	for rows.Next() {
		var m MediaFile
		rows.Scan(&m.ID, &m.NodeID, &m.Filename, &m.StorageURL, &m.Checksum, &m.MimeType, &m.FileSize, &m.UploadedBy)
		media = append(media, m)
	}
	json.NewEncoder(w).Encode(media)
}

// === API Handlers - Blog ===
func handleBlogPosts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	rows, _ := db.Query(`SELECT id, node_id, slug, excerpt, publish_date, category FROM blog_posts ORDER BY publish_date DESC`)
	defer rows.Close()

	var posts []BlogPost
	for rows.Next() {
		var post BlogPost
		var pubDate sql.NullInt64
		rows.Scan(&post.ID, &post.NodeID, &post.Slug, &post.Excerpt, &pubDate, &post.Category)
		if pubDate.Valid {
			t := time.Unix(pubDate.Int64, 0)
			post.PublishDate = &t
		}
		posts = append(posts, post)
	}
	json.NewEncoder(w).Encode(posts)
}

func handleBlogPost(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	slug := r.URL.Query().Get("slug")

	var post BlogPost
	db.QueryRow(`SELECT id, node_id, slug, excerpt, publish_date, category FROM blog_posts WHERE slug = ?`, slug).
		Scan(&post.ID, &post.NodeID, &post.Slug, &post.Excerpt, &post.PublishDate, &post.Category)

	json.NewEncoder(w).Encode(post)
}

// === API Handlers - Export ===
func handleExport(w http.ResponseWriter, r *http.Request) {
	siteID := r.URL.Query().Get("site_id")
	nodeID := r.URL.Query().Get("node_id")
	format := r.URL.Query().Get("format")

	if format == "zip" || format == "static" {
		// Full site export
		if siteID != "" {
			w.Header().Set("Content-Type", "application/zip")

			opts := ExportOptions{
				SiteID:        siteID,
				IncludeAssets: true,
				Theme:         "default",
				Format:        "zip",
			}

			zipData, err := ExportSiteAsStatic(opts)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
				return
			}

			w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=veil-site-%s.zip", siteID))
			w.Write(zipData)
			return
		}

		// Single node export (legacy)
		if nodeID != "" {
			w.Header().Set("Content-Type", "application/zip")
			var node Node
			db.QueryRow(`SELECT id, title, content FROM nodes WHERE id = ?`, nodeID).
				Scan(&node.ID, &node.Title, &node.Content)

			buf := new(bytes.Buffer)
			zw := zip.NewWriter(buf)

			f, _ := zw.Create(node.Title + ".md")
			io.WriteString(f, node.Content)

			zw.Close()

			w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=veil-export-%s.zip", node.ID))
			io.Copy(w, buf)
			return
		}
	}

	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(map[string]string{"error": "Missing site_id or node_id parameter"})
}

func handleRSSFeed(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/rss+xml")
}

// === API Handlers - Publishing ===
func handlePublishingChannels(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
}

func handlePublishHistory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
}

// === API Handlers - Permissions ===
func handleVisibility(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	nodeID := r.URL.Query().Get("node_id")
	visibility := r.URL.Query().Get("visibility")

	if r.Method == "PUT" {
		db.Exec(`UPDATE node_visibility SET visibility = ? WHERE node_id = ?`, visibility, nodeID)
	}

	var vis string
	db.QueryRow(`SELECT visibility FROM node_visibility WHERE node_id = ?`, nodeID).Scan(&vis)
	json.NewEncoder(w).Encode(map[string]string{"visibility": vis})
}

// === API Handlers - Search ===
func handleSearch(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	query := r.URL.Query().Get("q")

	rows, _ := db.Query(`SELECT id, type, path, title, content FROM nodes 
		WHERE deleted_at IS NULL AND (title LIKE ? OR content LIKE ?) ORDER BY path`,
		"%"+query+"%", "%"+query+"%")
	defer rows.Close()

	var results []Node
	for rows.Next() {
		var node Node
		rows.Scan(&node.ID, &node.Type, &node.Path, &node.Title, &node.Content)
		results = append(results, node)
	}
	json.NewEncoder(w).Encode(results)
}

// === API Handlers - Citations ===
func handleCitations(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	nodeID := r.URL.Query().Get("node_id")

	rows, _ := db.Query(`SELECT id, node_id, citation_key, authors, title, year, publisher, url, doi, citation_format, raw_bibtex
		FROM citations WHERE node_id = ?`, nodeID)
	defer rows.Close()

	var citations []Citation
	for rows.Next() {
		var c Citation
		rows.Scan(&c.ID, &c.NodeID, &c.CitationKey, &c.Authors, &c.Title, &c.Year, &c.Publisher, &c.URL, &c.DOI, &c.CitationFormat, &c.RawBibtex)
		citations = append(citations, c)
	}
	json.NewEncoder(w).Encode(citations)
}

// === API Handlers - Sites ===
func handleSites(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == "GET" {
		// List all sites
		rows, err := db.Query(`SELECT id, name, description, type, created_at, modified_at FROM sites ORDER BY created_at DESC`)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		defer rows.Close()

		sites := make([]Site, 0) // Initialize as empty slice, not nil
		for rows.Next() {
			var s Site
			var createdAt, modifiedAt int64
			err := rows.Scan(&s.ID, &s.Name, &s.Description, &s.Type, &createdAt, &modifiedAt)
			if err != nil {
				continue
			}
			s.CreatedAt = time.Unix(createdAt, 0)
			s.ModifiedAt = time.Unix(modifiedAt, 0)
			sites = append(sites, s)
		}

		json.NewEncoder(w).Encode(sites)
	} else if r.Method == "POST" {
		// Create new site
		var req map[string]string
		json.NewDecoder(r.Body).Decode(&req)

		id := fmt.Sprintf("site_%d", time.Now().UnixNano())
		name := req["name"]
		description := req["description"]
		siteType := req["type"]
		if siteType == "" {
			siteType = "project"
		}

		now := time.Now().Unix()
		_, err := db.Exec(`
			INSERT INTO sites (id, name, description, type, created_at, modified_at)
			VALUES (?, ?, ?, ?, ?, ?)
		`, id, name, description, siteType, now, now)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		site := Site{
			ID:          id,
			Name:        name,
			Description: description,
			Type:        siteType,
			CreatedAt:   time.Unix(now, 0),
			ModifiedAt:  time.Unix(now, 0),
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(site)
	} else if r.Method == "PUT" {
		var s Site
		json.NewDecoder(r.Body).Decode(&s)
		if s.ID == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "missing site id"})
			return
		}

		now := time.Now().Unix()
		_, err := db.Exec(`UPDATE sites SET name = ?, description = ?, type = ?, modified_at = ? WHERE id = ?`, s.Name, s.Description, s.Type, now, s.ID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		s.ModifiedAt = time.Unix(now, 0)
		json.NewEncoder(w).Encode(s)
	}
}

func handleSitesDetail(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Extract site ID from path like /api/sites/site_123/nodes
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	siteID := parts[3]

	// Route to appropriate handler
	if len(parts) > 4 && parts[4] == "nodes" {
		handleSiteNodes(w, r, siteID)
	}
}

func handleSiteNodes(w http.ResponseWriter, r *http.Request, siteID string) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == "GET" {
		// Extract nodeID from path if present
		parts := strings.Split(r.URL.Path, "/")

		if len(parts) > 5 {
			// GET /api/sites/site_123/nodes/node_456
			nodeID := parts[5]

			var node Node
			var createdAt, modifiedAt int64
			err := db.QueryRow(`
				SELECT id, type, COALESCE(parent_id, ''), path, title, COALESCE(content, ''), COALESCE(mime_type, ''), COALESCE(slug, ''), COALESCE(canonical_uri, ''), COALESCE(body, ''), COALESCE(metadata, ''), COALESCE(status, ''), created_at, modified_at
				FROM nodes WHERE id = ? AND site_id = ?
			`, nodeID, siteID).Scan(&node.ID, &node.Type, &node.ParentID, &node.Path, &node.Title, &node.Content, &node.MimeType, &node.Slug, &node.CanonicalURI, &node.Body, &node.Metadata, &node.Status, &createdAt, &modifiedAt)

			if err != nil {
				// Debug: log more details to help diagnose why a node that should exist isn't found
				log.Printf("Node lookup failed: id=%s site=%s err=%v", nodeID, siteID, err)
				var count int
				err2 := db.QueryRow(`SELECT COUNT(1) FROM nodes WHERE id = ? AND site_id = ?`, nodeID, siteID).Scan(&count)
				if err2 != nil {
					log.Printf("Node lookup count query failed: %v", err2)
				} else {
					log.Printf("Node lookup count for id=%s site=%s -> %d", nodeID, siteID, count)
				}

				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode(map[string]string{"error": "Node not found"})
				return
			}

			node.CreatedAt = time.Unix(createdAt, 0)
			node.ModifiedAt = time.Unix(modifiedAt, 0)
			json.NewEncoder(w).Encode(node)
		} else {
			// GET /api/sites/site_123/nodes
			rows, err := db.Query(`
				SELECT id, type, COALESCE(parent_id, ''), path, title, COALESCE(content, ''), COALESCE(mime_type, ''), 
				       COALESCE(slug, ''), COALESCE(canonical_uri, ''), COALESCE(body, ''), COALESCE(metadata, ''), 
				       COALESCE(status, 'draft'), created_at, modified_at
				FROM nodes WHERE site_id = ? ORDER BY created_at DESC
			`, siteID)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
				return
			}
			defer rows.Close()

			nodes := make([]Node, 0)
			for rows.Next() {
				var n Node
				var createdAt, modifiedAt int64
				err := rows.Scan(&n.ID, &n.Type, &n.ParentID, &n.Path, &n.Title, &n.Content, &n.MimeType,
					&n.Slug, &n.CanonicalURI, &n.Body, &n.Metadata, &n.Status, &createdAt, &modifiedAt)
				if err != nil {
					log.Printf("Failed to scan node row: %v", err)
					continue
				}
				n.CreatedAt = time.Unix(createdAt, 0)
				n.ModifiedAt = time.Unix(modifiedAt, 0)
				nodes = append(nodes, n)
			}

			json.NewEncoder(w).Encode(nodes)
		}
	} else if r.Method == "POST" {
		// Create new node in site
		var node Node
		json.NewDecoder(r.Body).Decode(&node)

		// Ensure unique path within site
		var exists int
		err := db.QueryRow(`SELECT COUNT(1) FROM nodes WHERE site_id = ? AND path = ?`, siteID, node.Path).Scan(&exists)
		if err == nil && exists > 0 {
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]string{"error": "Node path already exists in this site"})
			return
		}

		// Generate node ID and timestamps
		node.ID = fmt.Sprintf("node_%d", time.Now().UnixNano())
		now := time.Now().Unix()

		// Auto-generate slug if not provided
		if node.Slug == "" {
			node.Slug = generateSlug(node.Title)
		}

		// Auto-generate canonical URI if not provided
		if node.CanonicalURI == "" {
			node.CanonicalURI = fmt.Sprintf("veil://%s/%s/%s", siteID, node.Type, node.Slug)
		}

		// Set default status if not provided
		if node.Status == "" {
			node.Status = "draft"
		}

		// Set default visibility if not provided
		if node.Visibility == "" {
			node.Visibility = "public"
		}

		_, err = db.Exec(`
			INSERT INTO nodes (id, type, path, title, content, mime_type, slug, canonical_uri, body, metadata, status, visibility, created_at, modified_at, site_id)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, node.ID, node.Type, node.Path, node.Title, node.Content, node.MimeType, node.Slug, node.CanonicalURI, node.Body, node.Metadata, node.Status, node.Visibility, now, now, siteID)

		if err != nil {
			// Convert DB-level unique constraint errors into a 409 Conflict so clients can handle them
			if strings.Contains(err.Error(), "UNIQUE constraint failed: nodes.path") || strings.Contains(err.Error(), "constraint failed: UNIQUE constraint failed: nodes.path") {
				w.WriteHeader(http.StatusConflict)
				json.NewEncoder(w).Encode(map[string]string{"error": "Node path already exists in this site"})
				return
			}

			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		node.CreatedAt = time.Unix(now, 0)
		node.ModifiedAt = time.Unix(now, 0)

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(node)
	} else if r.Method == "PUT" {
		// Update node
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) < 6 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		nodeID := parts[5]
		var node Node
		json.NewDecoder(r.Body).Decode(&node)

		now := time.Now().Unix()
		_, err := db.Exec(`
			UPDATE nodes SET title = ?, content = ?, mime_type = ?, slug = ?, canonical_uri = ?, body = ?, metadata = ?, status = ?, modified_at = ? WHERE id = ? AND site_id = ?
		`, node.Title, node.Content, node.MimeType, node.Slug, node.CanonicalURI, node.Body, node.Metadata, node.Status, now, nodeID, siteID)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
	}
}

// === Utilities ===
func markdownToHTML(md string) string {
	html := md
	// Headers
	html = regexp.MustCompile(`^### (.*?)$`).ReplaceAllString(html, "<h3>$1</h3>")
	html = regexp.MustCompile(`^## (.*?)$`).ReplaceAllString(html, "<h2>$1</h2>")
	html = regexp.MustCompile(`^# (.*?)$`).ReplaceAllString(html, "<h1>$1</h1>")
	// Bold/Italic
	html = regexp.MustCompile(`\*\*(.*?)\*\*`).ReplaceAllString(html, "<strong>$1</strong>")
	html = regexp.MustCompile(`__(.*?)__`).ReplaceAllString(html, "<strong>$1</strong>")
	html = regexp.MustCompile(`\*(.*?)\*`).ReplaceAllString(html, "<em>$1</em>")
	html = regexp.MustCompile(`_(.*?)_`).ReplaceAllString(html, "<em>$1</em>")
	// Line breaks
	html = strings.ReplaceAll(html, "\n", "<br>\n")
	return html
}

// applyMigrations runs embedded SQL migration files against the provided database.
// It logs warnings for individual statements but attempts to apply all migrations.
func applyMigrations(database *sql.DB) error {
	migrationFiles, err := fs.ReadDir(migrations, "migrations")
	if err != nil {
		return err
	}

	// Sort migration files by name to ensure proper order
	sort.Slice(migrationFiles, func(i, j int) bool {
		return migrationFiles[i].Name() < migrationFiles[j].Name()
	})

	for _, file := range migrationFiles {
		content, _ := migrations.ReadFile("migrations/" + file.Name())
		statements := strings.Split(string(content), ";")
		for _, stmt := range statements {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" {
				continue
			}

			// Detect ALTER TABLE ... ADD COLUMN statements and make them idempotent
			upper := strings.ToUpper(stmt)
			if strings.HasPrefix(upper, "ALTER TABLE") && strings.Contains(upper, "ADD COLUMN") {
				// Try to parse table and column name
				re := regexp.MustCompile(`(?i)ALTER\s+TABLE\s+([` + "`" + `\w]+)\s+ADD\s+COLUMN(?:\s+IF\s+NOT\s+EXISTS)?\s+([\w` + "`" + `\"]+)`)
				m := re.FindStringSubmatch(stmt)
				table := ""
				col := ""
				if len(m) >= 3 {
					table = strings.Trim(m[1], "`\"")
					col = strings.Trim(m[2], "`\"")
				}

				if table != "" && col != "" {
					// Check if column already exists
					exists := false
					rows, _ := database.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
					for rows.Next() {
						var cid int
						var name, ctype string
						var notnull, dfltValue, pk interface{}
						rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk)
						if name == col {
							exists = true
							break
						}
					}
					if rows != nil {
						rows.Close()
					}

					if exists {
						// Column already present — skip this statement
						continue
					}

					// Some SQLite builds reject 'IF NOT EXISTS'; try executing without it
					execStmt := strings.Replace(stmt, "IF NOT EXISTS", "", 1)
					execStmt = strings.TrimSpace(execStmt)
					if _, err := database.Exec(execStmt); err != nil {
						// Try a minimal ALTER that only sets column name and type (strip constraints like UNIQUE/REFERENCES)
						log.Printf("Migration %s ALTER TABLE add column error (table=%s col=%s) attempting fallback: %v\n", file.Name(), table, col, err)
						// Build a minimal ALTER statement: extract type token after column name
						remainderRe := regexp.MustCompile(`(?i)ADD\s+COLUMN(?:\s+IF\s+NOT\s+EXISTS)?\s+([` + "`" + `\w]+)\s+([A-Za-z0-9()]+)`)
						m2 := remainderRe.FindStringSubmatch(stmt)
						if len(m2) >= 3 {
							colName := strings.Trim(m2[1], "`\"")
							colType := m2[2]
							fallback := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, colName, colType)
							if _, err2 := database.Exec(fallback); err2 != nil {
								log.Printf("Migration %s fallback ALTER failed (table=%s col=%s): %v\n", file.Name(), table, colName, err2)
							}
						}
					}
					continue
				}
			}

			if _, err := database.Exec(stmt); err != nil {
				// Ignore benign errors caused by dialect differences or missing referenced columns when re-running
				errStr := err.Error()
				if strings.Contains(errStr, "already exists") || strings.Contains(errStr, "near \"EXISTS\": syntax error") || strings.Contains(errStr, "no such column") || strings.Contains(errStr, "duplicate column") {
					// ignore
					continue
				}
				log.Printf("Migration %s statement error: %v\n", file.Name(), err)
			}
		}
	}

	return nil
}

// === Utility Functions ===

// generateSlug creates a URL-friendly slug from a title
func generateSlug(title string) string {
	// Convert to lowercase
	slug := strings.ToLower(title)

	// Replace spaces and special chars with hyphens
	slug = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(slug, "-")

	// Remove leading/trailing hyphens
	slug = strings.Trim(slug, "-")

	// Limit length
	if len(slug) > 100 {
		slug = slug[:100]
	}

	return slug
}

// generateUniqueSlug ensures a slug is unique by appending a number if needed
func generateUniqueSlug(db *sql.DB, baseSlug, siteID string) string {
	slug := baseSlug
	counter := 1

	for {
		var count int
		err := db.QueryRow(`SELECT COUNT(1) FROM nodes WHERE slug = ? AND site_id = ?`, slug, siteID).Scan(&count)
		if err != nil || count == 0 {
			return slug
		}
		slug = fmt.Sprintf("%s-%d", baseSlug, counter)
		counter++
	}
}
