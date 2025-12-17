package codex

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// ExportCommitToZip writes a ZIP archive containing the commit metadata and
// all associated objects (as files under objects/<hash>). It writes commit.json
// at the root with commit details.
func ExportCommitToZip(w io.Writer, repo *Repository, commitHash string) error {
	c, err := repo.storage.GetCommit(commitHash)
	if err != nil {
		return fmt.Errorf("commit lookup: %w", err)
	}
	zw := zip.NewWriter(w)
	defer zw.Close()

	// write commit metadata
	cb, _ := json.MarshalIndent(c, "", "  ")
	if fw, err := zw.Create("commit.json"); err == nil {
		if _, err := fw.Write(cb); err != nil {
			return err
		}
	} else {
		return err
	}

	// write each object
	for _, h := range c.Objects {
		// object data
		rc, ct, err := repo.storage.GetObjectStream(h)
		if err != nil {
			// fallback to legacy get
			if b, err2 := repo.storage.GetObject(h); err2 == nil {
				if fw, err := zw.Create("objects/" + h); err == nil {
					_, _ = fw.Write(b)
				}
				// continue to write metadata if available
			}
			continue
		}
		func() {
			defer rc.Close()
			if fw, err := zw.Create("objects/" + h); err == nil {
				_, _ = io.Copy(fw, rc)
			}
			// write metadata
			meta := map[string]interface{}{"content_type": ct}
			if mb, err := json.Marshal(meta); err == nil {
				if fw, err := zw.Create("objects/" + h + ".meta.json"); err == nil {
					_, _ = fw.Write(mb)
				}
			}
		}()
	}
	return nil
}

// ExportCommitToJSONLD returns a compact JSON-LD style representation of the commit
// with objects embedded when they are JSON-parseable, otherwise base64 is used.
func ExportCommitToJSONLD(repo *Repository, commitHash string) ([]byte, error) {
	c, err := repo.storage.GetCommit(commitHash)
	if err != nil {
		return nil, fmt.Errorf("commit lookup: %w", err)
	}
	out := map[string]interface{}{}
	out["commit"] = c
	objs := []interface{}{}
	for _, h := range c.Objects {
		if rc, ct, err := repo.storage.GetObjectStream(h); err == nil {
			var b bytes.Buffer
			_, _ = io.Copy(&b, rc)
			rc.Close()
			if ct == "application/json" {
				var parsed interface{}
				if json.Unmarshal(b.Bytes(), &parsed) == nil {
					objs = append(objs, map[string]interface{}{"hash": h, "object": parsed})
					continue
				}
			}
			objs = append(objs, map[string]interface{}{"hash": h, "content_base64": b.Bytes()})
		} else if b, err := repo.storage.GetObject(h); err == nil {
			// try to include raw JSON
			var parsed interface{}
			if json.Unmarshal(b, &parsed) == nil {
				objs = append(objs, map[string]interface{}{"hash": h, "object": parsed})
			} else {
				objs = append(objs, map[string]interface{}{"hash": h, "content_base64": b})
			}
		}
	}
	out["objects"] = objs
	out["exported_at"] = time.Now().UTC()
	return json.MarshalIndent(out, "", "  ")
}
