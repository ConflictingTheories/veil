package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	codex "veil/pkg/codex"
	fsstorage "veil/pkg/codex/storage/fs"
)

func TestCodexStatusAndObjectEndpoints(t *testing.T) {
	wd, _ := os.Getwd()
	tmp, err := ioutil.TempDir("", "codex-api-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)
	os.Chdir(tmp)
	defer os.Chdir(wd)

	// create .codex/objects and one object
	objs := filepath.Join(tmp, ".codex", "objects")
	os.MkdirAll(objs, 0755)
	objPath := filepath.Join(objs, "o1.json")
	ioutil.WriteFile(objPath, []byte(`{"urn":"urn:test:1"}`), 0644)

	mux := http.NewServeMux()
	registerCodexHandlers(mux)

	// status
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/codex/status", nil)
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status code: %d body:%s", rr.Code, rr.Body.String())
	}

	// query
	rr = httptest.NewRecorder()
	req = httptest.NewRequest("POST", "/api/codex/query", nil)
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("query failed: %d", rr.Code)
	}

	// object
	rr = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/api/codex/object?hash=o1", nil)
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("get object failed: %d body:%s", rr.Code, rr.Body.String())
	}
}

func TestCodexObjectUploadAndDownload(t *testing.T) {
	wd, _ := os.Getwd()
	tmp, err := ioutil.TempDir("", "codex-api-upload-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)
	os.Chdir(tmp)
	defer os.Chdir(wd)

	mux := http.NewServeMux()
	registerCodexHandlers(mux)

	// upload plain text
	body := "hello-world"
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/codex/object", strings.NewReader(body))
	req.Header.Set("Content-Type", "text/plain")
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("upload failed: %d %s", rr.Code, rr.Body.String())
	}
	// extract hash from response
	var res map[string]string
	json.NewDecoder(rr.Body).Decode(&res)
	h := res["hash"]
	if h == "" {
		t.Fatalf("no hash returned")
	}

	// download it back
	rr = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/api/codex/object?hash="+h, nil)
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("download failed: %d %s", rr.Code, rr.Body.String())
	}
	if rr.Header().Get("Content-Type") != "text/plain" {
		t.Fatalf("unexpected content-type: %s", rr.Header().Get("Content-Type"))
	}
	b, _ := ioutil.ReadAll(rr.Body)
	if string(b) != body {
		t.Fatalf("payload mismatch: %s", string(b))
	}
}

func TestCodexMergeAPI(t *testing.T) {
	wd, _ := os.Getwd()
	tmp, err := ioutil.TempDir("", "codex-api-merge-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)
	os.Chdir(tmp)
	defer os.Chdir(wd)

	mux := http.NewServeMux()
	registerCodexHandlers(mux)

	// Prepare base with o1
	objs := filepath.Join(tmp, ".codex", "objects")
	os.MkdirAll(objs, 0755)
	ioutil.WriteFile(filepath.Join(objs, "o1.json"), []byte(`{"urn":"urn:node:1","title":"v1"}`), 0644)
	// commit c1
	fs := fsstorage.New(tmp)
	c1 := &codex.Commit{Hash: "c1", Author: "a", Timestamp: time.Now().Add(-time.Hour), Message: "base", Objects: []string{"o1"}}
	fs.PutCommit(c1)

	// ours modifies -> o1a
	ioutil.WriteFile(filepath.Join(objs, "o1a.json"), []byte(`{"urn":"urn:node:1","title":"v2"}`), 0644)
	c2 := &codex.Commit{Hash: "c2", Parents: []string{"c1"}, Author: "a", Timestamp: time.Now().Add(-30 * time.Minute), Message: "ours", Objects: []string{"o1a"}}
	fs.PutCommit(c2)

	// theirs modifies differently -> o1b
	ioutil.WriteFile(filepath.Join(objs, "o1b.json"), []byte(`{"urn":"urn:node:1","title":"v3"}`), 0644)
	c3 := &codex.Commit{Hash: "c3", Parents: []string{"c1"}, Author: "b", Timestamp: time.Now(), Message: "theirs", Objects: []string{"o1b"}}
	fs.PutCommit(c3)

	// Call merge API
	body := `{"base":"c1","ours":"c2","theirs":"c3","author":"m","message":"merge"}`
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/codex/merge", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusConflict {
		t.Fatalf("expected 409 conflict for conflicting merge, got %d: %s", rr.Code, rr.Body.String())
	}
	// Now test a non-conflicting merge: c2 with a new object o2 and c3 with o3
	// create c4 and c5
	ioutil.WriteFile(filepath.Join(objs, "o2.json"), []byte(`{"urn":"urn:node:2","title":"two"}`), 0644)
	c4 := &codex.Commit{Hash: "c4", Parents: []string{"c1"}, Author: "a", Timestamp: time.Now().Add(-20 * time.Minute), Message: "c4", Objects: []string{"o1", "o2"}}
	fs.PutCommit(c4)
	ioutil.WriteFile(filepath.Join(objs, "o3.json"), []byte(`{"urn":"urn:node:3","title":"three"}`), 0644)
	c5 := &codex.Commit{Hash: "c5", Parents: []string{"c1"}, Author: "b", Timestamp: time.Now().Add(-10 * time.Minute), Message: "c5", Objects: []string{"o1", "o3"}}
	fs.PutCommit(c5)

	rr = httptest.NewRecorder()
	req = httptest.NewRequest("POST", "/api/codex/merge", strings.NewReader(`{"base":"c1","ours":"c4","theirs":"c5","author":"m","message":"merge"}`))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201 created for merge, got %d: %s", rr.Code, rr.Body.String())
	}
	var res map[string]string
	json.NewDecoder(rr.Body).Decode(&res)
	if res["hash"] == "" {
		t.Fatalf("expected merged commit hash, got none: %s", rr.Body.String())
	}
}

func TestCodexExportAPI(t *testing.T) {
	wd, _ := os.Getwd()
	tmp, err := ioutil.TempDir("", "codex-api-export-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)
	os.Chdir(tmp)
	defer os.Chdir(wd)

	// Setup a simple repo
	objs := filepath.Join(tmp, ".codex", "objects")
	os.MkdirAll(objs, 0755)
	ioutil.WriteFile(filepath.Join(objs, "o1.json"), []byte(`{"urn":"urn:x:1","title":"X"}`), 0644)
	fs := fsstorage.New(tmp)
	c1 := &codex.Commit{Hash: "c1", Author: "a", Timestamp: time.Now(), Message: "m", Objects: []string{"o1"}}
	fs.PutCommit(c1)

	mux := http.NewServeMux()
	registerCodexHandlers(mux)

	// zip
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/codex/export?hash=c1&format=zip", nil)
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("export zip failed: %d %s", rr.Code, rr.Body.String())
	}
	if rr.Header().Get("Content-Type") != "application/zip" {
		t.Fatalf("unexpected content-type: %s", rr.Header().Get("Content-Type"))
	}

	// jsonld
	rr = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/api/codex/export?hash=c1&format=jsonld", nil)
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("export jsonld failed: %d %s", rr.Code, rr.Body.String())
	}
	if rr.Header().Get("Content-Type") != "application/ld+json" {
		t.Fatalf("unexpected content-type: %s", rr.Header().Get("Content-Type"))
	}
}

func TestCodexCommitsAndDiffEndpoints(t *testing.T) {
	wd, _ := os.Getwd()
	tmp, err := ioutil.TempDir("", "codex-api-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)
	os.Chdir(tmp)
	defer os.Chdir(wd)

	// setup objects and commits
	objs := filepath.Join(tmp, ".codex", "objects")
	os.MkdirAll(objs, 0755)
	ioutil.WriteFile(filepath.Join(objs, "o1.json"), []byte(`{"urn":"u1"}`), 0644)
	ioutil.WriteFile(filepath.Join(objs, "o2.json"), []byte(`{"urn":"u2"}`), 0644)

	fs := fsstorage.New(tmp)
	c1 := &codex.Commit{Hash: "c1", Author: "a", Timestamp: time.Now().Add(-time.Hour), Message: "one", Objects: []string{"o1"}}
	c2 := &codex.Commit{Hash: "c2", Author: "a", Timestamp: time.Now(), Message: "two", Objects: []string{"o1", "o2"}}
	fs.PutCommit(c1)
	fs.PutCommit(c2)

	mux := http.NewServeMux()
	registerCodexHandlers(mux)

	// list commits
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/codex/commits?limit=10", nil)
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("commits list failed: %d %s", rr.Code, rr.Body.String())
	}

	// diff
	rr = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/api/codex/diff?from=c1&to=c2", nil)
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("diff failed: %d %s", rr.Code, rr.Body.String())
	}
}
