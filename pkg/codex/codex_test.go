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

func TestListCommitsAndDiff(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "codex-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	fs := fsadapter.New(tmpdir)
	// create two objects
	_ = fs.PutObject("o1", []byte(`{"urn":"urn:1","type":"text","title":"One"}`))
	_ = fs.PutObject("o2", []byte(`{"urn":"urn:2","type":"text","title":"Two"}`))

	// commit1 with o1
	c1 := &codex.Commit{Hash: "c1", Parents: []string{}, Author: "a", Timestamp: time.Now().Add(-time.Hour), Message: "first", Objects: []string{"o1"}}
	if err := fs.PutCommit(c1); err != nil {
		t.Fatalf("putcommit1: %v", err)
	}

	// commit2 with o1 and o2
	c2 := &codex.Commit{Hash: "c2", Parents: []string{"c1"}, Author: "a", Timestamp: time.Now(), Message: "second", Objects: []string{"o1", "o2"}}
	if err := fs.PutCommit(c2); err != nil {
		t.Fatalf("putcommit2: %v", err)
	}

	r := codex.NewRepository(fs, tmpdir)
	commits, err := r.ListCommits(10, 0)
	if err != nil {
		t.Fatalf("list commits: %v", err)
	}
	if len(commits) != 2 {
		t.Fatalf("expected 2 commits, got %d", len(commits))
	}

	diff, err := r.DiffCommits("c1", "c2")
	if err != nil {
		t.Fatalf("diff commits: %v", err)
	}
	if len(diff.Added) != 1 {
		t.Fatalf("expected 1 added in diff, got %d", len(diff.Added))
	}
}
