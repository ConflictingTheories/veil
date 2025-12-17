package main

import (
	"archive/zip"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestCreateBackupZip(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "migrate-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	// create veil.db
	dbPath := filepath.Join(tmpdir, "veil.db")
	if err := ioutil.WriteFile(dbPath, []byte("dummy"), 0644); err != nil {
		t.Fatal(err)
	}

	// create .codex/objects with one file
	objs := filepath.Join(tmpdir, ".codex", "objects")
	os.MkdirAll(objs, 0755)
	objPath := filepath.Join(objs, "o1.json")
	if err := ioutil.WriteFile(objPath, []byte(`{"urn":"u1"}`), 0644); err != nil {
		t.Fatal(err)
	}

	zipPath, err := createBackupZip(tmpdir)
	if err != nil {
		t.Fatalf("createBackupZip failed: %v", err)
	}
	defer os.Remove(zipPath)

	// open zip and verify entries
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatalf("open zip failed: %v", err)
	}
	defer r.Close()

	foundDB := false
	foundObj := false
	for _, f := range r.File {
		if f.Name == "veil.db" {
			foundDB = true
		}
		if f.Name == ".codex/objects/o1.json" {
			foundObj = true
		}
	}
	if !foundDB {
		t.Fatalf("veil.db not found in zip")
	}
	if !foundObj {
		t.Fatalf("object not found in zip")
	}
}
