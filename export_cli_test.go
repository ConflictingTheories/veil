package main

import (
	"archive/zip"
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	codex "veil/pkg/codex"
	fsstorage "veil/pkg/codex/storage/fs"
)

func captureStdout(f func()) []byte {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	f()
	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	os.Stdout = old
	return buf.Bytes()
}

func TestCLIExportCommit_ToStdoutZipAndJSONLD(t *testing.T) {
	tmp, _ := ioutil.TempDir("", "cli-export-test-")
	defer os.RemoveAll(tmp)
	wd, _ := os.Getwd()
	os.Chdir(tmp)
	defer os.Chdir(wd)

	objs := filepath.Join(tmp, ".codex", "objects")
	os.MkdirAll(objs, 0755)
	ioutil.WriteFile(filepath.Join(objs, "o1.json"), []byte(`{"urn":"urn:x:1","title":"X"}`), 0644)
	fs := fsstorage.New(tmp)
	c1 := &codex.Commit{Hash: "c1", Author: "a", Timestamp: time.Now(), Objects: []string{"o1"}}
	fs.PutCommit(c1)

	// zip to stdout
	os.Args = []string{"veil", "export", "commit", "c1"}
	b := captureStdout(func() { exportNode() })
	if len(b) == 0 {
		t.Fatalf("expected zip bytes on stdout")
	}
	_, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		t.Fatalf("invalid zip from stdout: %v", err)
	}

	// jsonld to stdout
	os.Args = []string{"veil", "export", "commit", "c1", "--format", "jsonld"}
	b = captureStdout(func() { exportNode() })
	if len(b) == 0 {
		t.Fatalf("expected jsonld bytes on stdout")
	}
	if b[0] != '{' {
		t.Fatalf("expected JSON object from jsonld, got: %s", string(b[:20]))
	}
}

func TestCLIExportCommit_ToFile(t *testing.T) {
	tmp, _ := ioutil.TempDir("", "cli-export-test-2-")
	defer os.RemoveAll(tmp)
	wd, _ := os.Getwd()
	os.Chdir(tmp)
	defer os.Chdir(wd)

	objs := filepath.Join(tmp, ".codex", "objects")
	os.MkdirAll(objs, 0755)
	ioutil.WriteFile(filepath.Join(objs, "o1.json"), []byte(`{"urn":"urn:x:1","title":"X"}`), 0644)
	fs := fsstorage.New(tmp)
	c1 := &codex.Commit{Hash: "c1", Author: "a", Timestamp: time.Now(), Objects: []string{"o1"}}
	fs.PutCommit(c1)

	outFile := filepath.Join(tmp, "out.zip")
	os.Args = []string{"veil", "export", "commit", "c1", "--out", outFile}
	// capture stdout should be empty
	b := captureStdout(func() { exportNode() })
	if len(b) != 0 {
		t.Fatalf("expected no stdout when --out specified, got %d bytes", len(b))
	}
	if _, err := os.Stat(outFile); err != nil {
		t.Fatalf("expected out file created: %v", err)
	}
}
