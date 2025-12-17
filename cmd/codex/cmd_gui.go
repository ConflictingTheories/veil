package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

// runGUI starts a static-file server serving a small PWA and provides simple API endpoints
func runGUI(args []string) {
	// ensure repo exists (create if absent)
	_ = initRepo()

	staticDir := filepath.Join("cmd", "codex", "static")

	// API endpoints
	http.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		head, _ := getHEAD()
		idx, _ := readIndex()
		json.NewEncoder(w).Encode(map[string]interface{}{"head": head, "staged": len(idx)})
	})

	http.HandleFunc("/api/commits", func(w http.ResponseWriter, r *http.Request) {
		commits, _ := listCommits()
		json.NewEncoder(w).Encode(commits)
	})

	http.HandleFunc("/api/objects", func(w http.ResponseWriter, r *http.Request) {
		// list object filenames
		dir := filepath.Join(codexDir, "objects")
		files, _ := ioutil.ReadDir(dir)
		var keys []string
		for _, f := range files {
			name := f.Name()
			if len(name) > 5 && name[len(name)-5:] == ".json" {
				keys = append(keys, name[:len(name)-5])
			}
		}
		json.NewEncoder(w).Encode(keys)
	})

	http.HandleFunc("/api/add", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		data, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		key, _, err := hashObject(bytesReader(data))
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		_ = writeObject(key, data)
		_ = stageObject(key)
		json.NewEncoder(w).Encode(map[string]string{"object": key})
	})

	http.HandleFunc("/api/entity", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		data, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		key, _, err := hashObject(bytesReader(data))
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		_ = writeObject(key, data)
		_ = stageObject(key)
		json.NewEncoder(w).Encode(map[string]string{"entity_object": key})
	})

	http.HandleFunc("/api/annotate", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		data, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		key, _, err := hashObject(bytesReader(data))
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		_ = writeObject(key, data)
		_ = stageObject(key)
		json.NewEncoder(w).Encode(map[string]string{"annotation_object": key})
	})

	http.HandleFunc("/api/export", func(w http.ResponseWriter, r *http.Request) {
		// Very small exporter: list commits and objects
		commits, _ := listCommits()
		dir := filepath.Join(codexDir, "objects")
		files, _ := ioutil.ReadDir(dir)
		var objs = map[string]json.RawMessage{}
		for _, f := range files {
			name := f.Name()
			if len(name) > 5 && name[len(name)-5:] == ".json" {
				key := name[:len(name)-5]
				b, _ := ioutil.ReadFile(filepath.Join(dir, name))
				objs[key] = b
			}
		}
		out := map[string]interface{}{"commits": commits, "objects": objs}
		json.NewEncoder(w).Encode(out)
	})

	http.HandleFunc("/api/commit", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var payload struct {
			Message string `json:"message"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || payload.Message == "" {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		idx, _ := readIndex()
		parent, _ := getHEAD()
		c := Commit{Message: payload.Message, Parent: parent, Timestamp: time.Now().Unix(), Objects: idx}
		hash, err := writeCommit(c)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		_ = updateHeadCommit(hash)
		_ = clearIndex()
		json.NewEncoder(w).Encode(map[string]string{"commit": hash})
	})

	// serve static files
	fs := http.FileServer(http.Dir(staticDir))
	http.Handle("/", fs)

	addr := ":3000"
	url := "http://localhost" + addr
	fmt.Println("Starting Codex GUI on", url)
	// open browser
	go func() {
		// small delay to allow server startup
		time.Sleep(200 * time.Millisecond)
		openBrowser(url)
	}()
	http.ListenAndServe(addr, nil)
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}
