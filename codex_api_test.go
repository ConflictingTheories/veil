package main

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
