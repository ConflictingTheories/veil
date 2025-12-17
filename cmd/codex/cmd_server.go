package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
)

func runServer(args []string) {
	http.HandleFunc("/datasets", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"datasets": []string{"urn:codex:example/mvp@v1"}})
	})

	http.HandleFunc("/push", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		// store commit object
		// compute key
		key, data, err := hashObject(bytesReader(body))
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if err := os.MkdirAll(".codex/objects", 0o755); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if err := writeObject(key, data); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		// update branch to this commit
		_ = os.MkdirAll(".codex/refs/heads", 0o755)
		_ = ioutil.WriteFile(".codex/refs/heads/main", []byte(key), 0o644)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok", "commit": key})
	})

	fmt.Println("Starting Codex mock server on :8080")
	http.ListenAndServe(":8080", nil)
}
