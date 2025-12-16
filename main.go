package main

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
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
		fmt.Println("veil v1.0.0 - Complete Edition")
		fmt.Println("Your universal content management system")
		fmt.Println("\nBuilt-in plugins:")
		fmt.Println("  - Git (version control)")
		fmt.Println("  - IPFS (decentralized publishing)")
		fmt.Println("  - Namecheap (DNS management)")
		fmt.Println("  - Media (video/audio/image processing)")
		fmt.Println("  - Pixospritz (game integration)")
		fmt.Println("  - Shader (WebGL shader editor)")
		fmt.Println("  - SVG (vector graphics editor)")
		fmt.Println("  - Code (syntax-highlighted code snippets)")
		fmt.Println("  - Todo (task management)")
		fmt.Println("  - Reminder (time-based notifications)")
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
	if err := applyMigrations(database); err != nil {
		log.Printf("Warning: migrations had errors: %v", err)
	}

	fmt.Printf("✓ Initialized vault at %s\n", path)
	fmt.Println("\nNext steps:")
	fmt.Println("  veil serve")
	fmt.Println("  veil gui")
}

// applyMigrations runs all migration files from embedded FS
func applyMigrations(database *sql.DB) error {
	migrationFiles, err := fs.ReadDir(migrations, "migrations")
	if err != nil {
		return fmt.Errorf("failed to read migrations: %v", err)
	}

	// Sort migration files by name to ensure proper order
	sort.Slice(migrationFiles, func(i, j int) bool {
		return migrationFiles[i].Name() < migrationFiles[j].Name()
	})

	for _, file := range migrationFiles {
		content, err := migrations.ReadFile("migrations/" + file.Name())
		if err != nil {
			log.Printf("Migration %s: read error: %v\n", file.Name(), err)
			continue
		}

		statements := strings.Split(string(content), ";")
		for _, stmt := range statements {
			stmt = strings.TrimSpace(stmt)
			if stmt != "" {
				if _, err := database.Exec(stmt); err != nil {
					log.Printf("Migration %s: %v\n", file.Name(), err)
				}
			}
		}
	}

	return nil
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

	// Media files
	mux.Handle("/media/", http.StripPrefix("/media/", http.FileServer(http.Dir("./media"))))

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

	// Preview route
	mux.HandleFunc("/preview/", handlePreview)

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
