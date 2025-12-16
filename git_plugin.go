package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// === Git Plugin ===

type GitPlugin struct {
	name    string
	version string
}

func NewGitPlugin() *GitPlugin {
	return &GitPlugin{
		name:    "git",
		version: "1.0.0",
	}
}

func (gp *GitPlugin) Name() string {
	return gp.name
}

func (gp *GitPlugin) Version() string {
	return gp.version
}

func (gp *GitPlugin) Initialize(config map[string]interface{}) error {
	// Store git config (repo URL, credentials, etc.)
	if repoURL, ok := config["repo_url"].(string); ok {
		if err := saveConfig("git_repo_url", repoURL); err != nil {
			return err
		}
	}

	if branch, ok := config["branch"].(string); ok {
		if err := saveConfig("git_branch", branch); err != nil {
			return err
		}
	}

	return nil
}

func (gp *GitPlugin) Validate() error {
	// Check if git is installed
	_, err := exec.LookPath("git")
	if err != nil {
		return fmt.Errorf("git not found in PATH")
	}
	return nil
}

func (gp *GitPlugin) Execute(ctx context.Context, action string, payload interface{}) (interface{}, error) {
	switch action {
	case "clone":
		return gp.clone(ctx, payload)
	case "push":
		return gp.push(ctx, payload)
	case "pull":
		return gp.pull(ctx, payload)
	case "commit":
		return gp.commit(ctx, payload)
	case "status":
		return gp.status(ctx)
	case "sync":
		return gp.sync(ctx, payload)
	default:
		return nil, fmt.Errorf("unknown action: %s", action)
	}
}

func (gp *GitPlugin) Shutdown() error {
	return nil
}

// Actions

type GitCloneRequest struct {
	RepoURL   string `json:"repo_url"`
	TargetDir string `json:"target_dir"`
	Branch    string `json:"branch"`
}

func (gp *GitPlugin) clone(ctx context.Context, payload interface{}) (interface{}, error) {
	req, ok := payload.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid payload")
	}

	repoURL := req["repo_url"].(string)
	targetDir := req["target_dir"].(string)

	// Ensure directory exists
	os.MkdirAll(targetDir, 0755)

	cmd := exec.CommandContext(ctx, "git", "clone", repoURL, targetDir)
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("clone failed: %v", err)
	}

	// Store repo URL
	credentialMgr.StoreCredential("git_repo_url", repoURL)
	saveConfig("git_local_path", targetDir)

	return map[string]string{"status": "cloned", "path": targetDir}, nil
}

type GitPushRequest struct {
	Message string `json:"message"`
	Branch  string `json:"branch"`
}

func (gp *GitPlugin) push(ctx context.Context, payload interface{}) (interface{}, error) {
	req, ok := payload.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid payload")
	}

	localPath, _ := loadConfig("git_local_path")
	message := req["message"].(string)
	branch := req["branch"].(string)

	if branch == "" {
		branch = "main"
	}

	// Change to repo directory
	os.Chdir(localPath.(string))

	// Add all changes
	cmd := exec.CommandContext(ctx, "git", "add", "-A")
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git add failed: %v", err)
	}

	// Commit
	cmd = exec.CommandContext(ctx, "git", "commit", "-m", message)
	if err := cmd.Run(); err != nil {
		// Might have nothing to commit
		log.Println("git commit info:", err)
	}

	// Push
	cmd = exec.CommandContext(ctx, "git", "push", "origin", branch)
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git push failed: %v", err)
	}

	return map[string]string{"status": "pushed"}, nil
}

func (gp *GitPlugin) pull(ctx context.Context, payload interface{}) (interface{}, error) {
	localPath, _ := loadConfig("git_local_path")

	os.Chdir(localPath.(string))

	cmd := exec.CommandContext(ctx, "git", "pull")
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git pull failed: %v", err)
	}

	return map[string]string{"status": "pulled"}, nil
}

type GitCommitRequest struct {
	Message string `json:"message"`
	NodeID  string `json:"node_id"`
}

func (gp *GitPlugin) commit(ctx context.Context, payload interface{}) (interface{}, error) {
	req, ok := payload.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid payload")
	}

	localPath, _ := loadConfig("git_local_path")
	message := req["message"].(string)
	nodeID := req["node_id"].(string)

	os.Chdir(localPath.(string))

	// Fetch the node from DB
	var node Node
	db.QueryRow(`SELECT id, path, content FROM nodes WHERE id = ?`, nodeID).
		Scan(&node.ID, &node.Path, &node.Content)

	// Write to file
	filePath := filepath.Join(localPath.(string), node.Path)
	os.MkdirAll(filepath.Dir(filePath), 0755)
	os.WriteFile(filePath, []byte(node.Content), 0644)

	// Git operations
	cmd := exec.CommandContext(ctx, "git", "add", node.Path)
	cmd.Run()

	cmd = exec.CommandContext(ctx, "git", "commit", "-m", message)
	if err := cmd.Run(); err != nil {
		log.Println("Commit info:", err)
	}

	// Record in database
	now := time.Now().Unix()
	db.Exec(`
		INSERT INTO git_commits (id, node_id, message, created_at)
		VALUES (?, ?, ?, ?)
	`, fmt.Sprintf("git_commit_%d", time.Now().UnixNano()), nodeID, message, now)

	return map[string]string{"status": "committed"}, nil
}

func (gp *GitPlugin) status(ctx context.Context) (interface{}, error) {
	localPath, _ := loadConfig("git_local_path")

	os.Chdir(localPath.(string))

	cmd := exec.CommandContext(ctx, "git", "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git status failed: %v", err)
	}

	return map[string]interface{}{
		"status": "ok",
		"files":  strings.Split(string(output), "\n"),
	}, nil
}

type GitSyncRequest struct {
	Direction string `json:"direction"` // push or pull
}

func (gp *GitPlugin) sync(ctx context.Context, payload interface{}) (interface{}, error) {
	req, ok := payload.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid payload")
	}

	direction := req["direction"].(string)

	if direction == "pull" {
		return gp.pull(ctx, nil)
	}

	// Default to push
	return gp.push(ctx, map[string]interface{}{
		"message": fmt.Sprintf("Auto-sync from Veil at %s", time.Now().Format(time.RFC3339)),
		"branch":  "main",
	})
}
