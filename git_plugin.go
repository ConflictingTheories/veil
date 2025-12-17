package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
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
	case "create_pr":
		return gp.createPR(ctx, payload)
	case "list_issues":
		return gp.listIssues(ctx, payload)
	case "create_issue":
		return gp.createIssue(ctx, payload)
	case "fork":
		return gp.forkRepo(ctx, payload)
	case "star":
		return gp.starRepo(ctx, payload)
	case "get_repos":
		return gp.getRepos(ctx, payload)
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

// GitHub API Integration Methods

type GitHubPRRequest struct {
	Title string `json:"title"`
	Head  string `json:"head"`
	Base  string `json:"base"`
	Body  string `json:"body"`
	Draft bool   `json:"draft,omitempty"`
}

func (gp *GitPlugin) createPR(ctx context.Context, payload interface{}) (interface{}, error) {
	req, ok := payload.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid payload")
	}

	token, _ := loadConfig("github_token")
	if token == nil {
		return nil, fmt.Errorf("GitHub token not configured")
	}

	repoURL, _ := loadConfig("git_repo_url")
	if repoURL == nil {
		return nil, fmt.Errorf("repository URL not configured")
	}

	// Extract owner/repo from URL
	repoPath := strings.TrimPrefix(repoURL.(string), "https://github.com/")
	parts := strings.Split(repoPath, "/")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid GitHub repository URL")
	}
	owner, repo := parts[0], parts[1]

	prData := map[string]interface{}{
		"title": req["title"].(string),
		"head":  req["head"].(string),
		"base":  req["base"].(string),
		"body":  req["body"].(string),
	}

	if draft, ok := req["draft"].(bool); ok {
		prData["draft"] = draft
	}

	jsonData, _ := json.Marshal(prData)

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls", owner, repo)
	httpReq, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	httpReq.Header.Set("Authorization", "token "+token.(string))
	httpReq.Header.Set("Accept", "application/vnd.github.v3+json")
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("GitHub API request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error: %s", string(body))
	}

	var prResponse map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&prResponse)

	return map[string]interface{}{
		"status": "created",
		"pr":     prResponse,
	}, nil
}

type GitHubIssueRequest struct {
	Title  string   `json:"title"`
	Body   string   `json:"body"`
	Labels []string `json:"labels,omitempty"`
}

func (gp *GitPlugin) createIssue(ctx context.Context, payload interface{}) (interface{}, error) {
	req, ok := payload.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid payload")
	}

	token, _ := loadConfig("github_token")
	if token == nil {
		return nil, fmt.Errorf("GitHub token not configured")
	}

	repoURL, _ := loadConfig("git_repo_url")
	if repoURL == nil {
		return nil, fmt.Errorf("repository URL not configured")
	}

	repoPath := strings.TrimPrefix(repoURL.(string), "https://github.com/")
	parts := strings.Split(repoPath, "/")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid GitHub repository URL")
	}
	owner, repo := parts[0], parts[1]

	issueData := map[string]interface{}{
		"title": req["title"].(string),
		"body":  req["body"].(string),
	}

	if labels, ok := req["labels"].([]interface{}); ok {
		var labelStrings []string
		for _, label := range labels {
			labelStrings = append(labelStrings, label.(string))
		}
		issueData["labels"] = labelStrings
	}

	jsonData, _ := json.Marshal(issueData)

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/issues", owner, repo)
	httpReq, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	httpReq.Header.Set("Authorization", "token "+token.(string))
	httpReq.Header.Set("Accept", "application/vnd.github.v3+json")
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("GitHub API request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error: %s", string(body))
	}

	var issueResponse map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&issueResponse)

	return map[string]interface{}{
		"status": "created",
		"issue":  issueResponse,
	}, nil
}

func (gp *GitPlugin) listIssues(ctx context.Context, payload interface{}) (interface{}, error) {
	req, ok := payload.(map[string]interface{})
	if !ok {
		req = map[string]interface{}{}
	}

	token, _ := loadConfig("github_token")
	if token == nil {
		return nil, fmt.Errorf("GitHub token not configured")
	}

	repoURL, _ := loadConfig("git_repo_url")
	if repoURL == nil {
		return nil, fmt.Errorf("repository URL not configured")
	}

	repoPath := strings.TrimPrefix(repoURL.(string), "https://github.com/")
	parts := strings.Split(repoPath, "/")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid GitHub repository URL")
	}
	owner, repo := parts[0], parts[1]

	state := "open"
	if s, ok := req["state"].(string); ok {
		state = s
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/issues?state=%s", owner, repo, state)
	httpReq, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	httpReq.Header.Set("Authorization", "token "+token.(string))
	httpReq.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("GitHub API request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error: %s", string(body))
	}

	var issues []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&issues)

	return map[string]interface{}{
		"issues": issues,
	}, nil
}

func (gp *GitPlugin) forkRepo(ctx context.Context, payload interface{}) (interface{}, error) {
	token, _ := loadConfig("github_token")
	if token == nil {
		return nil, fmt.Errorf("GitHub token not configured")
	}

	repoURL, _ := loadConfig("git_repo_url")
	if repoURL == nil {
		return nil, fmt.Errorf("repository URL not configured")
	}

	repoPath := strings.TrimPrefix(repoURL.(string), "https://github.com/")
	parts := strings.Split(repoPath, "/")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid GitHub repository URL")
	}
	owner, repo := parts[0], parts[1]

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/forks", owner, repo)
	httpReq, _ := http.NewRequestWithContext(ctx, "POST", url, nil)
	httpReq.Header.Set("Authorization", "token "+token.(string))
	httpReq.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("GitHub API request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 202 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error: %s", string(body))
	}

	var forkResponse map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&forkResponse)

	return map[string]interface{}{
		"status": "forked",
		"fork":   forkResponse,
	}, nil
}

func (gp *GitPlugin) starRepo(ctx context.Context, payload interface{}) (interface{}, error) {
	token, _ := loadConfig("github_token")
	if token == nil {
		return nil, fmt.Errorf("GitHub token not configured")
	}

	req, ok := payload.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid payload")
	}

	owner, ok := req["owner"].(string)
	if !ok {
		return nil, fmt.Errorf("owner is required")
	}

	repo, ok := req["repo"].(string)
	if !ok {
		return nil, fmt.Errorf("repo is required")
	}

	url := fmt.Sprintf("https://api.github.com/user/starred/%s/%s", owner, repo)
	httpReq, _ := http.NewRequestWithContext(ctx, "PUT", url, nil)
	httpReq.Header.Set("Authorization", "token "+token.(string))
	httpReq.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("GitHub API request failed: %v", err)
	}
	defer resp.Body.Close()

	success := resp.StatusCode == 204

	return map[string]interface{}{
		"starred": success,
		"owner":   owner,
		"repo":    repo,
	}, nil
}

func (gp *GitPlugin) getRepos(ctx context.Context, payload interface{}) (interface{}, error) {
	token, _ := loadConfig("github_token")
	if token == nil {
		return nil, fmt.Errorf("GitHub token not configured")
	}

	req, ok := payload.(map[string]interface{})
	if !ok {
		req = make(map[string]interface{})
	}

	// Default to authenticated user's repos if no username specified
	username, _ := req["username"].(string)
	var reposURL string
	if username != "" {
		reposURL = fmt.Sprintf("https://api.github.com/users/%s/repos", username)
	} else {
		reposURL = "https://api.github.com/user/repos"
	}

	// Add query parameters
	params := ""
	if visibility, ok := req["visibility"].(string); ok {
		params += "&visibility=" + visibility
	}
	if sort, ok := req["sort"].(string); ok {
		params += "&sort=" + sort
	}
	if direction, ok := req["direction"].(string); ok {
		params += "&direction=" + direction
	}
	if perPage, ok := req["per_page"].(float64); ok {
		params += fmt.Sprintf("&per_page=%.0f", perPage)
	}

	if params != "" {
		reposURL += "?" + params[1:] // Remove leading &
	}

	httpReq, _ := http.NewRequestWithContext(ctx, "GET", reposURL, nil)
	httpReq.Header.Set("Authorization", "token "+token.(string))
	httpReq.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("GitHub API request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error: %s", string(body))
	}

	var repos []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&repos)

	return map[string]interface{}{
		"repos": repos,
		"count": len(repos),
	}, nil
}
