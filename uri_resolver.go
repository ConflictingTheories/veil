package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// === URI Resolution System ===
// Veil uses a universal URI scheme: veil://site_id/type/slug
// This allows all entities to be addressable and linkable

type URIResolver struct {
	db *sql.DB
}

func NewURIResolver(database *sql.DB) *URIResolver {
	return &URIResolver{db: database}
}

// ResolveURI takes a veil:// URI and returns the corresponding node
func (ur *URIResolver) ResolveURI(uri string) (*Node, error) {
	// Parse veil:// URI
	re := regexp.MustCompile(`^veil://([^/]+)/([^/]+)/(.+)$`)
	matches := re.FindStringSubmatch(uri)

	if len(matches) < 4 {
		return nil, fmt.Errorf("invalid URI format: %s", uri)
	}

	siteID := matches[1]
	nodeType := matches[2]
	slug := matches[3]

	// Query database for node
	var node Node
	var createdAt, modifiedAt int64
	err := ur.db.QueryRow(`
		SELECT id, type, path, title, COALESCE(content, ''), COALESCE(slug, ''), 
		       COALESCE(canonical_uri, ''), COALESCE(body, ''), COALESCE(metadata, ''), 
		       COALESCE(status, 'draft'), COALESCE(visibility, 'public'), created_at, modified_at
		FROM nodes 
		WHERE site_id = ? AND type = ? AND slug = ?
	`, siteID, nodeType, slug).Scan(
		&node.ID, &node.Type, &node.Path, &node.Title, &node.Content,
		&node.Slug, &node.CanonicalURI, &node.Body, &node.Metadata,
		&node.Status, &node.Visibility, &createdAt, &modifiedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("node not found: %v", err)
	}

	node.CreatedAt = time.Unix(createdAt, 0)
	node.ModifiedAt = time.Unix(modifiedAt, 0)
	node.SiteID = siteID

	return &node, nil
}

// GetNodeURI generates a veil:// URI for a node
func (ur *URIResolver) GetNodeURI(nodeID string) (string, error) {
	var siteID, nodeType, slug string
	err := ur.db.QueryRow(`
		SELECT COALESCE(site_id, ''), type, COALESCE(slug, id)
		FROM nodes WHERE id = ?
	`, nodeID).Scan(&siteID, &nodeType, &slug)

	if err != nil {
		return "", err
	}

	if siteID == "" {
		siteID = "default"
	}

	return fmt.Sprintf("veil://%s/%s/%s", siteID, nodeType, slug), nil
}

// RegisterNodeURI creates a custom URI alias for a node
func (ur *URIResolver) RegisterNodeURI(nodeID, customURI string, isPrimary bool) error {
	id := fmt.Sprintf("uri_%d", time.Now().UnixNano())
	now := time.Now().Unix()

	primary := 0
	if isPrimary {
		primary = 1
	}

	_, err := ur.db.Exec(`
		INSERT INTO node_uris (id, node_id, uri, is_primary, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, id, nodeID, customURI, primary, now)

	return err
}

// GetAllNodeURIs returns all URIs for a node
func (ur *URIResolver) GetAllNodeURIs(nodeID string) ([]NodeURI, error) {
	rows, err := ur.db.Query(`
		SELECT id, node_id, uri, is_primary, created_at
		FROM node_uris WHERE node_id = ?
	`, nodeID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var uris []NodeURI
	for rows.Next() {
		var u NodeURI
		var createdAt int64
		var isPrimary int

		rows.Scan(&u.ID, &u.NodeID, &u.URI, &isPrimary, &createdAt)
		u.IsPrimary = isPrimary == 1
		u.CreatedAt = time.Unix(createdAt, 0)
		uris = append(uris, u)
	}

	return uris, nil
}

var uriResolver *URIResolver

func initURIResolver() {
	uriResolver = NewURIResolver(db)
}

// === API Handlers for URI System ===

func handleNodeURIs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == "GET" {
		nodeID := r.URL.Query().Get("node_id")

		if nodeID != "" {
			// Get all URIs for a node
			uris, err := uriResolver.GetAllNodeURIs(nodeID)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
				return
			}
			json.NewEncoder(w).Encode(uris)
		} else {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "node_id required"})
		}
	} else if r.Method == "POST" {
		// Create new URI alias
		var req struct {
			NodeID    string `json:"node_id"`
			URI       string `json:"uri"`
			IsPrimary bool   `json:"is_primary"`
		}

		json.NewDecoder(r.Body).Decode(&req)

		err := uriResolver.RegisterNodeURI(req.NodeID, req.URI, req.IsPrimary)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"status": "created"})
	} else if r.Method == "DELETE" {
		// Delete URI
		id := r.URL.Query().Get("id")
		_, err := db.Exec(`DELETE FROM node_uris WHERE id = ?`, id)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
	}
}

// handleResolveURI resolves a veil:// URI to a node
func handleResolveURI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	uri := r.URL.Query().Get("uri")
	if uri == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "uri parameter required"})
		return
	}

	// Support both veil:// and short URIs
	if !strings.HasPrefix(uri, "veil://") {
		uri = "veil://" + uri
	}

	node, err := uriResolver.ResolveURI(uri)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(node)
}

// handleGenerateURI generates a veil:// URI for a node
func handleGenerateURI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	nodeID := r.URL.Query().Get("node_id")
	if nodeID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "node_id parameter required"})
		return
	}

	uri, err := uriResolver.GetNodeURI(nodeID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"uri": uri})
}
