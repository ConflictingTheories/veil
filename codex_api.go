package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	codexpkg "veil/pkg/codex"
	fsstorage "veil/pkg/codex/storage/fs"
)

// GET /api/codex/status
func handleCodexStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	repo := codexpkg.NewRepository(fsstorage.New("."), ".")
	st, err := repo.Status()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	st["checked_at"] = time.Now().UTC().Format(time.RFC3339)
	json.NewEncoder(w).Encode(st)
}

// GET /api/codex/object?hash=...
func handleCodexObject(w http.ResponseWriter, r *http.Request) {
	fs := fsstorage.New(".")
	switch r.Method {
	case "GET":
		h := r.URL.Query().Get("hash")
		if h == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "hash required"})
			return
		}
		rc, ct, err := fs.GetObjectStream(h)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		defer rc.Close()
		// If JSON, attempt to decode and pretty-print
		if ct == "application/json" {
			var out interface{}
			if err := json.NewDecoder(rc).Decode(&out); err == nil {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(out)
				return
			}
			// fallthrough to raw
		}
		w.Header().Set("Content-Type", ct)
		// stream raw bytes
		_, _ = io.Copy(w, rc)
	case "POST":
		// Upload raw body as object
		contentType := r.Header.Get("Content-Type")
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		hash, err := fs.PutObjectStream(r.Body, contentType)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"status": "created", "hash": hash})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// POST /api/codex/query  { prefix: "" }
func handleCodexQuery(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var req struct {
		Prefix string `json:"prefix"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	fs := fsstorage.New(".")
	list, err := fs.ListObjects(req.Prefix)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"count": len(list), "objects": list})
}

// POST /api/codex/commit  (Commit JSON)
func handleCodexCommit(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var c codexpkg.Commit
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid commit payload"})
		return
	}
	if c.Hash == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "commit hash required"})
		return
	}
	fs := fsstorage.New(".")
	if err := fs.PutCommit(&c); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "created", "hash": c.Hash})
}

// GET /api/codex/commits?limit=&offset=
func handleCodexCommitsList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	q := r.URL.Query()
	limit := 50
	offset := 0
	if l := q.Get("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	if o := q.Get("offset"); o != "" {
		fmt.Sscanf(o, "%d", &offset)
	}
	repo := codexpkg.NewRepository(fsstorage.New("."), ".")
	commits, err := repo.ListCommits(limit, offset)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(commits)
}

// GET /api/codex/commit?hash=...
func handleCodexCommitGet(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	h := r.URL.Query().Get("hash")
	if h == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "hash required"})
		return
	}
	fs := fsstorage.New(".")
	c, err := fs.GetCommit(h)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(c)
}

// GET /api/codex/diff?from=&to=
func handleCodexDiff(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	q := r.URL.Query()
	from := q.Get("from")
	to := q.Get("to")
	if from == "" || to == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "from and to required"})
		return
	}
	repo := codexpkg.NewRepository(fsstorage.New("."), ".")
	diff, err := repo.DiffCommits(from, to)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(diff)
}

// POST /api/codex/merge  { base:, ours:, theirs:, author:, message: }
func handleCodexMerge(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var req struct {
		Base    string `json:"base"`
		Ours    string `json:"ours"`
		Theirs  string `json:"theirs"`
		Author  string `json:"author"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid payload"})
		return
	}
	repo := codexpkg.NewRepository(fsstorage.New("."), ".")
	mcommit, conflicts, err := repo.MergeCommits(req.Base, req.Ours, req.Theirs, req.Author, req.Message)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	if len(conflicts) > 0 {
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]interface{}{"conflicts": conflicts})
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"hash": mcommit.Hash})
}

// GET /api/codex/export?hash=&format=zip|jsonld
func handleCodexExport(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	h := q.Get("hash")
	if h == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "hash required"})
		return
	}
	format := q.Get("format")
	if format == "" {
		format = "zip"
	}
	repo := codexpkg.NewRepository(fsstorage.New("."), ".")
	switch format {
	case "zip":
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", "attachment; filename=codex-"+h+".zip")
		if err := codexpkg.ExportCommitToZip(w, repo, h); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
	case "jsonld":
		w.Header().Set("Content-Type", "application/ld+json")
		b, err := codexpkg.ExportCommitToJSONLD(repo, h)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		w.Write(b)
	default:
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "unsupported format"})
	}
}

func registerCodexHandlers(mux *http.ServeMux) {
	mux.HandleFunc("/api/codex/status", handleCodexStatus)
	mux.HandleFunc("/api/codex/object", handleCodexObject)
	mux.HandleFunc("/api/codex/query", handleCodexQuery)
	mux.HandleFunc("/api/codex/commit", handleCodexCommit)
	mux.HandleFunc("/api/codex/commits", handleCodexCommitsList)
	mux.HandleFunc("/api/codex/commit/get", handleCodexCommitGet)
	mux.HandleFunc("/api/codex/diff", handleCodexDiff)
	mux.HandleFunc("/api/codex/merge", handleCodexMerge)
	mux.HandleFunc("/api/codex/export", handleCodexExport)
}
