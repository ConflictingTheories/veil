package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

const codexDir = ".codex"

type Commit struct {
	Message   string   `json:"message"`
	Parent    string   `json:"parent,omitempty"`
	Timestamp int64    `json:"timestamp"`
	Objects   []string `json:"objects"`
}

func ensureRepo() error {
	if _, err := os.Stat(codexDir); os.IsNotExist(err) {
		return fmt.Errorf("not a codex repository (run 'codex init')")
	}
	return nil
}

func initRepo() error {
	if _, err := os.Stat(codexDir); !os.IsNotExist(err) {
		return fmt.Errorf(".codex already exists")
	}
	if err := os.MkdirAll(filepath.Join(codexDir, "objects"), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(codexDir, "refs", "heads"), 0o755); err != nil {
		return err
	}
	// create initial HEAD pointing to empty
	if err := ioutil.WriteFile(filepath.Join(codexDir, "HEAD"), []byte("refs/heads/main"), 0o644); err != nil {
		return err
	}
	// initial branch file
	if err := ioutil.WriteFile(filepath.Join(codexDir, "refs", "heads", "main"), []byte(""), 0o644); err != nil {
		return err
	}
	// empty index
	idx := []string{}
	b, _ := json.Marshal(idx)
	if err := ioutil.WriteFile(filepath.Join(codexDir, "index.json"), b, 0o644); err != nil {
		return err
	}
	return nil
}

func hashObject(r io.Reader) (string, []byte, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return "", nil, err
	}
	h := sha256.Sum256(data)
	key := hex.EncodeToString(h[:])
	return key, data, nil
}

func writeObject(key string, data []byte) error {
	dest := filepath.Join(codexDir, "objects", key+".json")
	return ioutil.WriteFile(dest, data, 0o644)
}

func stageObject(key string) error {
	idxPath := filepath.Join(codexDir, "index.json")
	b, err := ioutil.ReadFile(idxPath)
	if err != nil {
		return err
	}
	var idx []string
	if err := json.Unmarshal(b, &idx); err != nil {
		return err
	}
	idx = append(idx, key)
	nb, _ := json.Marshal(idx)
	return ioutil.WriteFile(idxPath, nb, 0o644)
}

func readIndex() ([]string, error) {
	b, err := ioutil.ReadFile(filepath.Join(codexDir, "index.json"))
	if err != nil {
		return nil, err
	}
	var idx []string
	if err := json.Unmarshal(b, &idx); err != nil {
		return nil, err
	}
	return idx, nil
}

func clearIndex() error {
	b, _ := json.Marshal([]string{})
	return ioutil.WriteFile(filepath.Join(codexDir, "index.json"), b, 0o644)
}

func getHEAD() (string, error) {
	headRef, err := ioutil.ReadFile(filepath.Join(codexDir, "HEAD"))
	if err != nil {
		return "", err
	}
	ref := string(headRef)
	refPath := filepath.Join(codexDir, ref)
	b, err := ioutil.ReadFile(refPath)
	if err != nil {
		return "", nil // no commit yet
	}
	return string(b), nil
}

func updateHeadCommit(commitHash string) error {
	headRef, err := ioutil.ReadFile(filepath.Join(codexDir, "HEAD"))
	if err != nil {
		return err
	}
	ref := string(headRef)
	refPath := filepath.Join(codexDir, ref)
	return ioutil.WriteFile(refPath, []byte(commitHash), 0o644)
}

func writeCommit(c Commit) (string, error) {
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(b)
	key := hex.EncodeToString(h[:])
	if err := writeObject(key, b); err != nil {
		return "", err
	}
	return key, nil
}

// readObject returns raw bytes of an object by key
func readObject(key string) ([]byte, error) {
	p := filepath.Join(codexDir, "objects", key+".json")
	return ioutil.ReadFile(p)
}

// listCommits walks commits from HEAD and returns Commit objects
func listCommits() ([]Commit, error) {
	head, err := getHEAD()
	if err != nil {
		return nil, err
	}
	if head == "" {
		return []Commit{}, nil
	}
	var commits []Commit
	cur := head
	for cur != "" {
		b, err := readObject(cur)
		if err != nil {
			break
		}
		var c Commit
		if err := json.Unmarshal(b, &c); err != nil {
			break
		}
		commits = append(commits, c)
		cur = c.Parent
	}
	return commits, nil
}
