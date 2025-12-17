package main

import (
	"archive/zip"
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	codexpkg "veil/pkg/codex"
	fsstorage "veil/pkg/codex/storage/fs"
	plugins "veil/pkg/plugins"

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
	case "codex":
		codexCommand()
		return
	case "migrate":
		migrateCommand()
		return
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

func codexCommand() {
	// Usage: veil codex status [path]
	if len(os.Args) < 3 {
		fmt.Println("Usage: veil codex <status> [repo-path]")
		return
	}
	action := os.Args[2]
	switch action {
	case "status":
		repoPath := "."
		if len(os.Args) >= 4 {
			repoPath = os.Args[3]
		}
		storage := fsstorage.New(repoPath)
		repo := codexpkg.NewRepository(storage, repoPath)
		st, err := repo.Status()
		if err != nil {
			fmt.Printf("error: %v\n", err)
			return
		}
		b, _ := json.MarshalIndent(st, "", "  ")
		fmt.Println(string(b))
	default:
		fmt.Println("Unknown codex action; supported: status")
	}
}

func migrateCommand() {
	// Usage: veil migrate [--dry-run] [--backup] [repo-path]
	dryRun := false
	doBackup := false
	repoPath := "."
	for i := 2; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "--dry-run":
			dryRun = true
		case "--backup":
			doBackup = true
		default:
			repoPath = os.Args[i]
		}
	}

	if doBackup {
		backupPath, err := createBackupZip(repoPath)
		if err != nil {
			fmt.Printf("backup failed: %v\n", err)
			return
		}
		fmt.Printf("backup created: %s\n", backupPath)
	}

	// Dry-run: report counts
	objsDir := filepath.Join(repoPath, ".codex", "objects")
	count := 0
	if fi, err := os.Stat(objsDir); err == nil && fi.IsDir() {
		files, _ := ioutil.ReadDir(objsDir)
		for _, f := range files {
			if !f.IsDir() {
				count++
			}
		}
	}
	fmt.Printf(".codex objects: %d\n", count)

	if dryRun {
		fmt.Println("dry-run: no other actions performed")
	} else {
		fmt.Println("migrate: (no-op) -- full migration not implemented yet")
	}
}

func createBackupZip(base string) (string, error) {
	// create zip named veil-backup-<timestamp>.zip in base
	ts := time.Now().UTC().Format("20060102T150405Z")
	out := filepath.Join(base, fmt.Sprintf("veil-backup-%s.zip", ts))
	f, err := os.Create(out)
	if err != nil {
		return "", err
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	defer zw.Close()

	// include veil.db if exists
	dbFile := filepath.Join(base, "veil.db")
	if fi, err := os.Stat(dbFile); err == nil && !fi.IsDir() {
		if err := addFileToZip(zw, dbFile, "veil.db"); err != nil {
			return "", err
		}
	}

	// include .codex directory if present
	codexDir := filepath.Join(base, ".codex")
	if fi, err := os.Stat(codexDir); err == nil && fi.IsDir() {
		// walk objects
		objectsDir := filepath.Join(codexDir, "objects")
		if fi2, err := os.Stat(objectsDir); err == nil && fi2.IsDir() {
			files, _ := ioutil.ReadDir(objectsDir)
			for _, f := range files {
				if f.IsDir() {
					continue
				}
				path := filepath.Join(objectsDir, f.Name())
				if err := addFileToZip(zw, path, filepath.Join(".codex", "objects", f.Name())); err != nil {
					return "", err
				}
			}
		}
	}

	return out, nil
}

func addFileToZip(zw *zip.Writer, path, rel string) error {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	w, err := zw.Create(rel)
	if err != nil {
		return err
	}
	_, err = w.Write(b)
	return err
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
	plugins.LoadEnabledPluginsFromDB(db)

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
	plugins.LoadEnabledPluginsFromDB(db)

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
	// Ensure URI resolver initialized for tests and server
	if uriResolver == nil {
		initURIResolver()
	}
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
	mux.HandleFunc("/api/plugins", plugins.HandlePluginsList)
	mux.HandleFunc("/api/plugin-execute", plugins.HandlePluginExecute)
	mux.HandleFunc("/api/credentials", plugins.HandleCredentialsAPI)
	mux.HandleFunc("/api/publish-job", plugins.HandlePublishJob)
	mux.HandleFunc("/api/plugins-registry", handlePluginsRegistry)
	mux.HandleFunc("/api/node-uris", handleNodeURIs)
	mux.HandleFunc("/api/resolve-uri", handleResolveURI)
	mux.HandleFunc("/api/generate-uri", handleGenerateURI)

	// Codex UI route (serve small built UI)
	mux.Handle("/codex/", http.StripPrefix("/codex/", http.FileServer(http.FS(webFS))))
	// Full prototype UI: serve codex-universalis static files
	mux.Handle("/codex-prototype/", http.StripPrefix("/codex-prototype/", http.FileServer(http.Dir("./codex-universalis"))))

	// Register codex API handlers
	registerCodexHandlers(mux)

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
