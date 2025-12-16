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

// === Types ===
type Node struct {
	ID         string    `json:"id"`
	Type       string    `json:"type"`
	ParentID   string    `json:"parent_id,omitempty"`
	Path       string    `json:"path"`
	Title      string    `json:"title"`
	Content    string    `json:"content"`
	MimeType   string    `json:"mime_type"`
	CreatedAt  time.Time `json:"created_at"`
	ModifiedAt time.Time `json:"modified_at"`
	Tags       []string  `json:"tags,omitempty"`
	References []string  `json:"references,omitempty"`
	Visibility string    `json:"visibility,omitempty"`
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
	ID               string `json:"id"`
	NodeID           string `json:"node_id"`
	Filename         string `json:"filename"`
	OriginalFilename string `json:"original_filename"`
	MimeType         string `json:"mime_type"`
	FileSize         int64  `json:"file_size"`
	Hash             string `json:"hash"`
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

	dir := filepath.Dir(path)
	os.MkdirAll(dir, 0755)

	database, err := sql.Open("sqlite", path)
	if err != nil {
		log.Fatal("Failed to create database:", err)
	}
	defer database.Close()

	// Run migrations
	migrationFiles, _ := fs.ReadDir(migrations, "migrations")
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
	defer db.Close()

	setupRoutes()
	addr := ":" + port
	fmt.Printf("✓ Veil running at http://localhost:%s\n", port)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func gui() {
	dbPath = "./veil.db"
	var err error
	db, err = sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer db.Close()

	setupRoutes()
	go func() {
		log.Fatal(http.ListenAndServe(":8080", nil))
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

func setupRoutes() {
	mux := http.NewServeMux()

	// Static files
	webFS, _ := fs.Sub(webUI, "web")
	mux.Handle("/", http.FileServer(http.FS(webFS)))

	// Core node APIs
	mux.HandleFunc("/api/nodes", handleNodes)
	mux.HandleFunc("/api/node/", handleNode)
	mux.HandleFunc("/api/node-create", handleNodeCreate)
	mux.HandleFunc("/api/node-update", handleNodeUpdate)
	mux.HandleFunc("/api/node-delete", handleNodeDelete)

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

	http.ListenAndServe("", mux)
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

	var node Node
	db.QueryRow(`SELECT id, path FROM nodes WHERE path LIKE ? OR title LIKE ?`, "%"+linkText+"%", "%"+linkText+"%").
		Scan(&node.ID, &node.Path)

	json.NewEncoder(w).Encode(node)
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

	db.Exec(`INSERT INTO media (id, node_id, filename, original_filename, mime_type, file_size, blob_data, hash, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		mediaID, "", handler.Filename, handler.Filename, handler.Header.Get("Content-Type"),
		buf.Len(), buf.Bytes(), hashStr, now)

	json.NewEncoder(w).Encode(map[string]string{"id": mediaID, "hash": hashStr})
}

func handleMedia(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	mediaID := r.URL.Query().Get("id")

	var media MediaFile
	db.QueryRow(`SELECT id, node_id, filename, original_filename, mime_type, file_size, hash FROM media WHERE id = ?`, mediaID).
		Scan(&media.ID, &media.NodeID, &media.Filename, &media.OriginalFilename, &media.MimeType, &media.FileSize, &media.Hash)

	json.NewEncoder(w).Encode(media)
}

func handleMediaLibrary(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID := r.URL.Query().Get("user_id")

	rows, _ := db.Query(`SELECT m.id, m.node_id, m.filename, m.mime_type, m.file_size 
		FROM media m JOIN media_library ml ON m.id = ml.media_id WHERE ml.user_id = ?`, userID)
	defer rows.Close()

	var media []MediaFile
	for rows.Next() {
		var m MediaFile
		rows.Scan(&m.ID, &m.NodeID, &m.Filename, &m.MimeType, &m.FileSize)
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
	w.Header().Set("Content-Type", "application/zip")
	nodeID := r.URL.Query().Get("node_id")
	format := r.URL.Query().Get("format")

	if format != "zip" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

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
	case "version":
		fmt.Println("veil v0.1.0")
	default:
		printUsage()
	}
}

func printUsage() {
	fmt.Println(`veil - Universal content management system

Usage:
  veil init [path]       Initialize new vault (default: ./veil.db)
  veil serve [--port N]  Start web server (default: 8080)
  veil gui               Launch GUI mode
  veil new <path>        Create new file/note
  veil list              List all nodes
  veil version           Show version

Examples:
  veil init ~/my-vault
  veil serve --port 3000
  veil new notes/ideas.md`)
}

func initVault() {
	path := "./veil.db"
	if len(os.Args) > 2 {
		path = os.Args[2]
	}

	if len(path) > 2 && path[:2] == "~/" {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, path[2:])
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Fatal("Failed to create directory:", err)
	}

	database, err := sql.Open("sqlite", path)
	if err != nil {
		log.Fatal("Failed to create database:", err)
	}
	defer database.Close()

	migrationFiles, _ := fs.ReadDir(migrations, "migrations")
	for _, file := range migrationFiles {
		content, _ := migrations.ReadFile("migrations/" + file.Name())
		if _, err := database.Exec(string(content)); err != nil {
			log.Fatal("Migration failed:", err)
		}
	}

	sampleID := fmt.Sprintf("node_%d", time.Now().UnixNano())
	now := time.Now().Unix()
	database.Exec(`
		INSERT INTO nodes (id, type, path, title, content, mime_type, created_at, modified_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, sampleID, "note", "welcome.md", "Welcome to Veil",
		"# Welcome to Veil\n\nStart writing!\n\n## Features\n- Single binary\n- SQLite storage\n- Clean UI",
		"text/markdown", now, now)

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
	defer db.Close()

	mux := http.NewServeMux()
	webFS, _ := fs.Sub(webUI, "web")
	mux.Handle("/", http.FileServer(http.FS(webFS)))
	mux.HandleFunc("/api/nodes", handleNodes)
	mux.HandleFunc("/api/node/", handleNode)

	addr := "localhost:" + port
	fmt.Printf("✓ Veil running at http://%s\n", addr)
	http.ListenAndServe(addr, mux)
}

func gui() {
	dbPath = "./veil.db"
	db, _ = sql.Open("sqlite", dbPath)
	defer db.Close()

	go func() {
		mux := http.NewServeMux()
		webFS, _ := fs.Sub(webUI, "web")
		mux.Handle("/", http.FileServer(http.FS(webFS)))
		mux.HandleFunc("/api/nodes", handleNodes)
		mux.HandleFunc("/api/node/", handleNode)
		http.ListenAndServe("localhost:8080", mux)
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
	cmd.Start()
	select {}
}

func createNode() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: veil new <path>")
		return
	}
	path := os.Args[2]
	dbPath = "./veil.db"
	db, _ = sql.Open("sqlite", dbPath)
	defer db.Close()

	id := fmt.Sprintf("node_%d", time.Now().UnixNano())
	db.Exec(`INSERT INTO nodes (id, type, path, title, content, mime_type, created_at, modified_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		id, "note", path, filepath.Base(path), "", "text/markdown", time.Now().Unix(), time.Now().Unix())
	fmt.Printf("✓ Created %s\n", path)
}

func listNodes() {
	dbPath = "./veil.db"
	db, _ = sql.Open("sqlite", dbPath)
	defer db.Close()

	rows, _ := db.Query(`SELECT path, title FROM nodes WHERE deleted_at IS NULL ORDER BY path`)
	defer rows.Close()

	fmt.Println("Nodes:")
	for rows.Next() {
		var path, title string
		rows.Scan(&path, &title)
		fmt.Printf("  %s - %s\n", path, title)
	}
}

func handleNodes(w http.ResponseWriter, r *http.Request) {
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
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(nodes)
	} else if r.Method == "POST" {
		var node Node
		json.NewDecoder(r.Body).Decode(&node)
		node.ID = fmt.Sprintf("node_%d", time.Now().UnixNano())
		now := time.Now().Unix()
		db.Exec(`INSERT INTO nodes (id, type, parent_id, path, title, content, mime_type, created_at, modified_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			node.ID, node.Type, node.ParentID, node.Path, node.Title, node.Content, node.MimeType, now, now)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(node)
	}
}

func handleNode(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/api/node/"):]
	if r.Method == "GET" {
		var node Node
		var created, modified int64
		db.QueryRow(`SELECT id, type, COALESCE(parent_id, ''), path, title, content, mime_type, created_at, modified_at 
			FROM nodes WHERE id = ?`, id).Scan(&node.ID, &node.Type, &node.ParentID, &node.Path,
			&node.Title, &node.Content, &node.MimeType, &created, &modified)
		node.CreatedAt = time.Unix(created, 0)
		node.ModifiedAt = time.Unix(modified, 0)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(node)
	} else if r.Method == "PUT" {
		var node Node
		json.NewDecoder(r.Body).Decode(&node)
		db.Exec(`UPDATE nodes SET title = ?, content = ?, modified_at = ? WHERE id = ?`,
			node.Title, node.Content, time.Now().Unix(), id)
		w.WriteHeader(204)
	}
}
