package codex_test

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"testing"
	"time"

	codex "veil/pkg/codex"
	fsadapter "veil/pkg/codex/storage/fs"
)

func TestExportCommitToZip_and_JSONLD(t *testing.T) {
	tmp, err := ioutil.TempDir("", "codex-export-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)

	fs := fsadapter.New(tmp)
	// create two objects
	fs.PutObject("o1", []byte(`{"urn":"urn:1","title":"One"}`))
	fs.PutObject("o2", []byte(`{"urn":"urn:2","title":"Two"}`))
	c := &codex.Commit{Hash: "c1", Author: "a", Timestamp: time.Now(), Message: "m", Objects: []string{"o1", "o2"}}
	fs.PutCommit(c)
	r := codex.NewRepository(fs, tmp)

	// zip export
	var buf bytes.Buffer
	if err := codex.ExportCommitToZip(&buf, r, "c1"); err != nil {
		t.Fatalf("zip export failed: %v", err)
	}
	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatalf("invalid zip: %v", err)
	}
	foundCommit := false
	foundObj := map[string]bool{}
	for _, f := range zr.File {
		if f.Name == "commit.json" {
			foundCommit = true
			rc, _ := f.Open()
			b, _ := io.ReadAll(rc)
			var out codex.Commit
			if json.Unmarshal(b, &out) != nil {
				t.Fatalf("commit.json invalid")
			}
			rc.Close()
		}
		if f.Name == "objects/o1" || f.Name == "objects/o2" {
			foundObj[f.Name] = true
		}
	}
	if !foundCommit || !foundObj["objects/o1"] || !foundObj["objects/o2"] {
		t.Fatalf("zip missing expected entries: commit:%v o1:%v o2:%v", foundCommit, foundObj["objects/o1"], foundObj["objects/o2"])
	}

	// jsonld
	b, err := codex.ExportCommitToJSONLD(r, "c1")
	if err != nil {
		t.Fatalf("jsonld export failed: %v", err)
	}
	var j map[string]interface{}
	if json.Unmarshal(b, &j) != nil {
		t.Fatalf("invalid jsonld")
	}
	if _, ok := j["commit"]; !ok {
		t.Fatalf("jsonld missing commit")
	}
}
