package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func setupTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()
	// Use in-memory SQLite for tests
	testDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}
	db = testDB
	if err := applyMigrations(db); err != nil {
		t.Fatalf("applyMigrations failed: %v", err)
	}

	// Ensure newer tables exist in case migrations didn't apply ALTERs in this SQLite build
	db.Exec(`CREATE TABLE IF NOT EXISTS node_uris (id TEXT PRIMARY KEY, node_id TEXT NOT NULL, uri TEXT NOT NULL UNIQUE, is_primary INTEGER DEFAULT 0, created_at INTEGER NOT NULL)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS plugins_registry (id TEXT PRIMARY KEY, name TEXT NOT NULL, slug TEXT UNIQUE, manifest TEXT, enabled INTEGER DEFAULT 0, created_at INTEGER, updated_at INTEGER)`)

	return testDB, func() { testDB.Close() }
}

func TestNodeURIsCRUD(t *testing.T) {
	testDB, cleanup := setupTestDB(t)
	defer cleanup()

	// Create site and node
	_, err := testDB.Exec(`INSERT INTO sites (id, name, description, type, created_at, modified_at) VALUES (?, ?, ?, ?, ?, ?)`, "site_test", "Test Site", "desc", "project", 1, 1)
	if err != nil {
		t.Fatalf("failed to insert site: %v", err)
	}
	_, err = testDB.Exec(`INSERT INTO nodes (id, type, path, title, content, created_at, modified_at) VALUES (?, ?, ?, ?, ?, ?, ?)`, "node_test", "note", "test.md", "Test", "# Test", 1, 1)
	if err != nil {
		t.Fatalf("failed to insert node: %v", err)
	}

	mux := setupRoutes()

	// Create URI
	payload := map[string]string{"node_id": "node_test", "uri": "/test/uri"}
	b, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/node-uris", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	// List URIs
	req = httptest.NewRequest("GET", "/api/node-uris?node_id=node_test", nil)
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 on list, got %d", rr.Code)
	}
	var uris []NodeURI
	if err := json.NewDecoder(rr.Body).Decode(&uris); err != nil {
		t.Fatalf("failed to decode URIs: %v", err)
	}
	if len(uris) != 1 {
		t.Fatalf("expected 1 uri, got %d", len(uris))
	}
}

func TestPluginsRegistryCRUD(t *testing.T) {
	testDB, cleanup := setupTestDB(t)
	defer cleanup()

	mux := setupRoutes()

	// Create plugin via POST
	payload := map[string]interface{}{"name": "Dummy", "slug": "dummy", "manifest": "{}", "enabled": false}
	b, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/plugins-registry", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Get list and ensure presence
	req = httptest.NewRequest("GET", "/api/plugins-registry", nil)
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 on get, got %d", rr.Code)
	}
	var plugins []PluginManifest
	if err := json.NewDecoder(rr.Body).Decode(&plugins); err != nil {
		t.Fatalf("failed to decode plugins list: %v", err)
	}
	found := false
	for _, p := range plugins {
		if p.Slug == "dummy" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("created plugin not found in list")
	}

	// Enable the plugin via PUT
	updated := plugins[0]
	updated.Enabled = true
	b, _ = json.Marshal(updated)
	req = httptest.NewRequest("PUT", "/api/plugins-registry", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 on update, got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify DB updated
	var enabled int
	err := testDB.QueryRow(`SELECT enabled FROM plugins_registry WHERE slug = ?`, "dummy").Scan(&enabled)
	if err != nil {
		t.Fatalf("db query failed: %v", err)
	}
	if enabled != 1 {
		t.Fatalf("expected enabled=1, got %d", enabled)
	}
}

// Ensure tests run against GOPATH when formatting or linters run
func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
