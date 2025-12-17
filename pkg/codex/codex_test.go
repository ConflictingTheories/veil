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

func TestBranchCreateAndList(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "codex-branch-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	fs := fsadapter.New(tmpdir)
	r := codex.NewRepository(fs, tmpdir)

	// create a branch
	if err := r.CreateBranch("main", ""); err != nil {
		t.Fatalf("create branch error: %v", err)
	}

	// set head
	if err := r.SetRef("refs/heads/main", "c1"); err != nil {
		t.Fatalf("setref error: %v", err)
	}

	h, err := r.GetRef("refs/heads/main")
	if err != nil {
		t.Fatalf("getref error: %v", err)
	}
	if h != "c1" {
		t.Fatalf("expected head c1, got %s", h)
	}

	branches, err := r.ListBranches()
	if err != nil {
		t.Fatalf("list branches: %v", err)
	}
	found := false
	for _, b := range branches {
		if b == "main" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected main branch in %v", branches)
	}
}

func TestMergeCommits_NoConflict(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "codex-merge-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	fs := fsadapter.New(tmpdir)

	// create base commit with o1
	_ = fs.PutObject("o1", []byte(`{"urn":"urn:node:1","title":"base"}`))
	c1 := &codex.Commit{Hash: "c1", Author: "a", Timestamp: time.Now().Add(-time.Hour), Message: "base", Objects: []string{"o1"}}
	fs.PutCommit(c1)

	// ours adds o2
	_ = fs.PutObject("o2", []byte(`{"urn":"urn:node:2","title":"ours"}`))
	c2 := &codex.Commit{Hash: "c2", Parents: []string{"c1"}, Author: "a", Timestamp: time.Now().Add(-30 * time.Minute), Message: "ours", Objects: []string{"o1", "o2"}}
	fs.PutCommit(c2)

	// theirs adds o3
	_ = fs.PutObject("o3", []byte(`{"urn":"urn:node:3","title":"theirs"}`))
	c3 := &codex.Commit{Hash: "c3", Parents: []string{"c1"}, Author: "b", Timestamp: time.Now(), Message: "theirs", Objects: []string{"o1", "o3"}}
	fs.PutCommit(c3)

	r := codex.NewRepository(fs, tmpdir)
	mcommit, conflicts, err := r.MergeCommits("c1", "c2", "c3", "merger", "merge msg")
	if err != nil {
		t.Fatalf("merge error: %v", err)
	}
	if len(conflicts) != 0 {
		t.Fatalf("expected no conflicts, got %v", conflicts)
	}
	// merged commit should contain o1,o2,o3
	found := map[string]struct{}{}
	for _, h := range mcommit.Objects {
		found[h] = struct{}{}
	}
	for _, want := range []string{"o1", "o2", "o3"} {
		if _, ok := found[want]; !ok {
			t.Fatalf("expected merged object %s", want)
		}
	}
}

func TestMergeCommits_Conflict(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "codex-merge-test-2-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	fs := fsadapter.New(tmpdir)
	// base: node with urn:urn:node:1 value v1
	_ = fs.PutObject("o1", []byte(`{"urn":"urn:node:1","title":"v1"}`))
	c1 := &codex.Commit{Hash: "c1", Author: "a", Timestamp: time.Now().Add(-time.Hour), Message: "base", Objects: []string{"o1"}}
	fs.PutCommit(c1)

	// ours modifies node -> o1a
	_ = fs.PutObject("o1a", []byte(`{"urn":"urn:node:1","title":"v2"}`))
	c2 := &codex.Commit{Hash: "c2", Parents: []string{"c1"}, Author: "a", Timestamp: time.Now().Add(-30 * time.Minute), Message: "ours", Objects: []string{"o1a"}}
	fs.PutCommit(c2)

	// theirs modifies node differently -> o1b
	_ = fs.PutObject("o1b", []byte(`{"urn":"urn:node:1","title":"v3"}`))
	c3 := &codex.Commit{Hash: "c3", Parents: []string{"c1"}, Author: "b", Timestamp: time.Now(), Message: "theirs", Objects: []string{"o1b"}}
	fs.PutCommit(c3)

	r := codex.NewRepository(fs, tmpdir)
	mcommit, conflicts, err := r.MergeCommits("c1", "c2", "c3", "merger", "merge msg")
	if err != nil {
		t.Fatalf("merge error: %v", err)
	}
	if mcommit != nil {
		t.Fatalf("expected no commit when conflicts, got %v", mcommit)
	}
	if len(conflicts) == 0 {
		t.Fatalf("expected conflicts, got none")
	}
	if conflicts[0].URN != "urn:node:1" {
		t.Fatalf("unexpected conflict urn: %s", conflicts[0].URN)
	}
}
