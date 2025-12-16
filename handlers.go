package main

import (
	"archive/zip"
	"bytes"
	"crypto/md5"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

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

	db.Exec(`INSERT INTO nodes (id, type, parent_id, path, title, content, mime_type, site_id, created_at, modified_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		node.ID, node.Type, node.ParentID, node.Path, node.Title, node.Content, node.MimeType, node.SiteID, now, now)

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
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseMultipartForm(32 << 20) // 32 MB max
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to parse form"})
		return
	}

	file, handler, err := r.FormFile("file")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "no file uploaded"})
		return
	}
	defer file.Close()

	buf := new(bytes.Buffer)
	io.Copy(buf, file)
	content := buf.Bytes()

	hash := md5.Sum(content)
	hashStr := fmt.Sprintf("%x", hash)

	mediaID := fmt.Sprintf("media_%d", time.Now().UnixNano())
	now := time.Now().Unix()

	// Ensure media dir exists and write file to disk
	os.MkdirAll("./media", 0755)
	filename := fmt.Sprintf("%s_%s", mediaID, handler.Filename)
	fpath := filepath.Join("media", filename)
	os.WriteFile(fpath, content, 0644)

	_, err = db.Exec(`INSERT INTO media (id, filename, storage_url, checksum, mime_type, size, uploaded_by, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		mediaID, handler.Filename, fpath, hashStr, handler.Header.Get("Content-Type"), len(content), "", now)
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

	// Route to appropriate handler based on path structure
	if len(parts) > 4 && parts[4] == "nodes" {
		if len(parts) > 6 {
			// Sub-resources: /api/sites/site_123/nodes/node_456/versions or /publish etc
			nodeID := parts[5]
			action := parts[6]

			switch action {
			case "versions":
				handleNodeVersions(w, r, siteID, nodeID)
			case "publish":
				handleNodePublish(w, r, siteID, nodeID)
			case "tags":
				handleNodeTagsNested(w, r, siteID, nodeID)
			case "media":
				handleNodeMedia(w, r, siteID, nodeID)
			case "backlinks":
				handleNodeBacklinks(w, r, siteID, nodeID)
			case "references":
				handleNodeReferences(w, r, siteID, nodeID)
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		} else {
			// /api/sites/site_123/nodes or /api/sites/site_123/nodes/node_456
			handleSiteNodes(w, r, siteID)
		}
	}
}

// Handle node versions for a specific site/node
func handleNodeVersions(w http.ResponseWriter, r *http.Request, siteID, nodeID string) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == "GET" {
		rows, err := db.Query(`
			SELECT id, node_id, version_number, content, title, status, published_at, created_at, modified_at, is_current
			FROM versions WHERE node_id = ? ORDER BY version_number DESC
		`, nodeID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		defer rows.Close()

		versions := []Version{}
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
}

// Handle node publish for a specific site/node
func handleNodePublish(w http.ResponseWriter, r *http.Request, siteID, nodeID string) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == "POST" {
		var req map[string]interface{}
		json.NewDecoder(r.Body).Decode(&req)

		visibility := "public"
		if v, ok := req["visibility"].(string); ok {
			visibility = v
		}

		now := time.Now().Unix()

		// Update node status
		_, err := db.Exec(`
			UPDATE nodes 
			SET status = 'published', visibility = ?, modified_at = ?
			WHERE id = ? AND site_id = ?
		`, visibility, now, nodeID, siteID)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		// Update current version
		db.Exec(`
			UPDATE versions 
			SET status = 'published', published_at = ?
			WHERE node_id = ? AND is_current = 1
		`, now, nodeID)

		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":     "published",
			"visibility": visibility,
		})
	}
}

// Handle node tags nested route
func handleNodeTagsNested(w http.ResponseWriter, r *http.Request, siteID, nodeID string) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == "POST" {
		var req map[string]string
		json.NewDecoder(r.Body).Decode(&req)

		tagName := req["name"]
		if tagName == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "tag name required"})
			return
		}

		// Get or create tag
		var tagID string
		err := db.QueryRow(`SELECT id FROM tags WHERE name = ?`, tagName).Scan(&tagID)
		if err != nil {
			tagID = fmt.Sprintf("tag_%d", time.Now().UnixNano())
			db.Exec(`INSERT INTO tags (id, name) VALUES (?, ?)`, tagID, tagName)
		}

		// Link tag to node
		ntID := fmt.Sprintf("nt_%d", time.Now().UnixNano())
		_, err = db.Exec(`
			INSERT OR IGNORE INTO node_tags (id, node_id, tag_id) VALUES (?, ?, ?)
		`, ntID, nodeID, tagID)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		json.NewEncoder(w).Encode(map[string]string{
			"status": "added",
			"tag":    tagName,
		})
	}
}

// Handle node media for a specific site/node
func handleNodeMedia(w http.ResponseWriter, r *http.Request, siteID, nodeID string) {
w.Header().Set("Content-Type", "application/json")
// TODO: Implement media handling for nodes
json.NewEncoder(w).Encode(map[string]interface{}{
"node_id": nodeID,
"site_id": siteID,
"media": []interface{}{},
})
}

// Handle node backlinks for a specific site/node  
func handleNodeBacklinks(w http.ResponseWriter, r *http.Request, siteID, nodeID string) {
w.Header().Set("Content-Type", "application/json")
rows, err := db.Query(`
SELECT DISTINCT n.id, n.title, n.type, n.path
FROM nodes n
INNER JOIN node_references nr ON n.id = nr.source_node_id
WHERE nr.target_node_id = ? AND n.deleted_at IS NULL
`, nodeID)

if err != nil {
w.WriteHeader(http.StatusInternalServerError)
json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
return
}
defer rows.Close()

var backlinks []map[string]interface{}
for rows.Next() {
var id, title, nodeType, path string
rows.Scan(&id, &title, &nodeType, &path)
backlinks = append(backlinks, map[string]interface{}{
"id": id,
"title": title,
"type": nodeType,
"path": path,
})
}

json.NewEncoder(w).Encode(map[string]interface{}{
"node_id": nodeID,
"backlinks": backlinks,
})
}

// Handle node references (forward links) for a specific site/node
func handleNodeReferences(w http.ResponseWriter, r *http.Request, siteID, nodeID string) {
w.Header().Set("Content-Type", "application/json")
rows, err := db.Query(`
SELECT DISTINCT n.id, n.title, n.type, n.path, nr.link_type, nr.link_text
FROM nodes n
INNER JOIN node_references nr ON n.id = nr.target_node_id
WHERE nr.source_node_id = ? AND n.deleted_at IS NULL
`, nodeID)

if err != nil {
w.WriteHeader(http.StatusInternalServerError)
json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
return
}
defer rows.Close()

var references []map[string]interface{}
for rows.Next() {
var id, title, nodeType, path, linkType, linkText string
rows.Scan(&id, &title, &nodeType, &path, &linkType, &linkText)
references = append(references, map[string]interface{}{
"id": id,
"title": title,
"type": nodeType,
"path": path,
"link_type": linkType,
"link_text": linkText,
})
}

json.NewEncoder(w).Encode(map[string]interface{}{
"node_id": nodeID,
"references": references,
})
}

// Handle site nodes listing
func handleSiteNodes(w http.ResponseWriter, r *http.Request, siteID string) {
w.Header().Set("Content-Type", "application/json")

rows, err := db.Query(`
SELECT id, type, COALESCE(parent_id, ''), path, title, content, 
       COALESCE(slug, ''), mime_type, created_at, modified_at
FROM nodes 
WHERE site_id = ? AND deleted_at IS NULL 
ORDER BY created_at DESC
`, siteID)

if err != nil {
w.WriteHeader(http.StatusInternalServerError)
json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
return
}
defer rows.Close()

var nodes []Node
for rows.Next() {
var node Node
var created, modified int64
err := rows.Scan(&node.ID, &node.Type, &node.ParentID, &node.Path, &node.Title,
&node.Content, &node.Slug, &node.MimeType, &created, &modified)
if err != nil {
continue
}
node.CreatedAt = time.Unix(created, 0)
node.ModifiedAt = time.Unix(modified, 0)
nodes = append(nodes, node)
}

json.NewEncoder(w).Encode(map[string]interface{}{
"site_id": siteID,
"nodes": nodes,
})
}
