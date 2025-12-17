package codex_test

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	codex "veil/pkg/codex"
	fsadapter "veil/pkg/codex/storage/fs"
)

func TestFSStoragePutGetListObject(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "codex-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	fs := fsadapter.New(tmpdir)
	payload := []byte(`{"urn":"urn:test:1"}`)
	hash := "deadbeef"

	if err := fs.PutObject(hash, payload); err != nil {
		t.Fatalf("PutObject error: %v", err)
	}

	b, err := fs.GetObject(hash)
	if err != nil {
		t.Fatalf("GetObject error: %v", err)
	}
	if string(b) != string(payload) {
		t.Fatalf("payload mismatch: got %s", string(b))
	}

	list, err := fs.ListObjects("")
	if err != nil {
		t.Fatalf("ListObjects error: %v", err)
	}
	found := false
	for _, n := range list {
		if n == hash {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected hash %s in list: %v", hash, list)
	}

	// Test commit
	commit := &codex.Commit{Hash: "c123", Author: "tester", Timestamp: time.Now(), Message: "test", Objects: []string{hash}}
	if err := fs.PutCommit(commit); err != nil {
		t.Fatalf("PutCommit error: %v", err)
	}
	c2, err := fs.GetCommit("c123")
	if err != nil {
		t.Fatalf("GetCommit error: %v", err)
	}
	if c2.Author != "tester" {
		t.Fatalf("commit mismatch: %v", c2)
	}

	// Repository status
	r := codex.NewRepository(fs, tmpdir)
	st, err := r.Status()
	if err != nil {
		t.Fatalf("Status error: %v", err)
	}
	if st["objects_count"].(int) == 0 {
		t.Fatalf("expected objects_count > 0")
	}
}
