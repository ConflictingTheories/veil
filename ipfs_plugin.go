package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// === IPFS Plugin ===

type IPFSPlugin struct {
	name       string
	version    string
	gatewayURL string
}

func NewIPFSPlugin(gatewayURL string) *IPFSPlugin {
	if gatewayURL == "" {
		gatewayURL = "http://localhost:5001"
	}
	return &IPFSPlugin{
		name:       "ipfs",
		version:    "1.0.0",
		gatewayURL: gatewayURL,
	}
}

func (ip *IPFSPlugin) Name() string {
	return ip.name
}

func (ip *IPFSPlugin) Version() string {
	return ip.version
}

func (ip *IPFSPlugin) Initialize(config map[string]interface{}) error {
	if url, ok := config["gateway_url"].(string); ok {
		ip.gatewayURL = url
		saveConfig("ipfs_gateway_url", url)
	}

	if pinService, ok := config["pin_service"].(string); ok {
		saveConfig("ipfs_pin_service", pinService)
	}

	return nil
}

func (ip *IPFSPlugin) Validate() error {
	// Test connection to IPFS gateway
	resp, err := http.Get(ip.gatewayURL + "/api/v0/version")
	if err != nil {
		return fmt.Errorf("cannot connect to IPFS gateway at %s: %v", ip.gatewayURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("IPFS gateway returned status %d", resp.StatusCode)
	}

	return nil
}

func (ip *IPFSPlugin) Execute(ctx context.Context, action string, payload interface{}) (interface{}, error) {
	switch action {
	case "add":
		return ip.addContent(ctx, payload)
	case "get":
		return ip.getContent(ctx, payload)
	case "publish":
		return ip.publishVersion(ctx, payload)
	case "pin":
		return ip.pinContent(ctx, payload)
	case "unpin":
		return ip.unpinContent(ctx, payload)
	case "status":
		return ip.status(ctx)
	default:
		return nil, fmt.Errorf("unknown action: %s", action)
	}
}

func (ip *IPFSPlugin) Shutdown() error {
	return nil
}

// Actions

type IPFSAddRequest struct {
	Content string `json:"content"`
	Name    string `json:"name"`
}

type IPFSAddResponse struct {
	Hash string `json:"hash"`
	Size int64  `json:"size"`
	Name string `json:"name"`
}

func (ip *IPFSPlugin) addContent(ctx context.Context, payload interface{}) (interface{}, error) {
	req, ok := payload.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid payload")
	}

	content := req["content"].(string)
	name := req["name"].(string)

	// Create form for IPFS add
	body := strings.NewReader(content)

	// Call IPFS add endpoint
	httpReq, _ := http.NewRequestWithContext(ctx, "POST", ip.gatewayURL+"/api/v0/add?wrap-with-directory=true", body)
	httpReq.Header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`, name))

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("ipfs add failed: %v", err)
	}
	defer resp.Body.Close()

	// Parse response
	respBody, _ := io.ReadAll(resp.Body)

	// Extract hash from response (simplified)
	responseStr := string(respBody)
	hash := extractIPFSHash(responseStr)

	if hash == "" {
		return nil, fmt.Errorf("failed to extract IPFS hash from response")
	}

	// Store in database
	now := int64(0) // time.Now().Unix() in context
	db.Exec(`
		INSERT INTO ipfs_content (id, hash, name, content, pinned, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, fmt.Sprintf("ipfs_%d", now), hash, name, content, false, now)

	return map[string]interface{}{
		"hash": hash,
		"name": name,
		"url":  fmt.Sprintf("ipfs://%s", hash),
	}, nil
}

type IPFSGetRequest struct {
	Hash string `json:"hash"`
}

func (ip *IPFSPlugin) getContent(ctx context.Context, payload interface{}) (interface{}, error) {
	req, ok := payload.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid payload")
	}

	hash := req["hash"].(string)

	resp, err := http.Get(fmt.Sprintf("%s/ipfs/%s", ip.gatewayURL, hash))
	if err != nil {
		return nil, fmt.Errorf("ipfs get failed: %v", err)
	}
	defer resp.Body.Close()

	content, _ := io.ReadAll(resp.Body)

	return map[string]interface{}{
		"hash":    hash,
		"content": string(content),
	}, nil
}

type IPFSPublishRequest struct {
	VersionID string `json:"version_id"`
	NodeID    string `json:"node_id"`
}

func (ip *IPFSPlugin) publishVersion(ctx context.Context, payload interface{}) (interface{}, error) {
	req, ok := payload.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid payload")
	}

	versionID := req["version_id"].(string)
	nodeID := req["node_id"].(string)

	// Fetch version from database
	var version Version
	var content string
	db.QueryRow(`
		SELECT id, node_id, title, content FROM versions WHERE id = ?
	`, versionID).Scan(&version.ID, &version.NodeID, &version.Title, &content)

	// Add to IPFS
	result, err := ip.addContent(ctx, map[string]interface{}{
		"content": content,
		"name":    version.Title + ".md",
	})
	if err != nil {
		return nil, err
	}

	resultMap := result.(map[string]interface{})
	hash := resultMap["hash"].(string)

	// Store IPFS record
	now := int64(0)
	db.Exec(`
		INSERT INTO ipfs_publications (id, version_id, node_id, ipfs_hash, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, fmt.Sprintf("pub_%d", now), versionID, nodeID, hash, now)

	return map[string]interface{}{
		"hash":      hash,
		"gateway":   fmt.Sprintf("https://gateway.pinata.cloud/ipfs/%s", hash),
		"version":   versionID,
		"published": true,
	}, nil
}

type IPFSPinRequest struct {
	Hash string `json:"hash"`
}

func (ip *IPFSPlugin) pinContent(ctx context.Context, payload interface{}) (interface{}, error) {
	req, ok := payload.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid payload")
	}

	hash := req["hash"].(string)

	// Call IPFS pin add
	httpReq, _ := http.NewRequestWithContext(ctx, "POST",
		fmt.Sprintf("%s/api/v0/pin/add?arg=%s", ip.gatewayURL, hash), nil)

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("pin add failed: %v", err)
	}
	defer resp.Body.Close()

	// Update database
	db.Exec(`UPDATE ipfs_content SET pinned = 1 WHERE hash = ?`, hash)

	return map[string]string{"status": "pinned"}, nil
}

type IPFSUnpinRequest struct {
	Hash string `json:"hash"`
}

func (ip *IPFSPlugin) unpinContent(ctx context.Context, payload interface{}) (interface{}, error) {
	req, ok := payload.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid payload")
	}

	hash := req["hash"].(string)

	// Call IPFS pin rm
	httpReq, _ := http.NewRequestWithContext(ctx, "POST",
		fmt.Sprintf("%s/api/v0/pin/rm?arg=%s", ip.gatewayURL, hash), nil)

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("pin rm failed: %v", err)
	}
	defer resp.Body.Close()

	// Update database
	db.Exec(`UPDATE ipfs_content SET pinned = 0 WHERE hash = ?`, hash)

	return map[string]string{"status": "unpinned"}, nil
}

func (ip *IPFSPlugin) status(ctx context.Context) (interface{}, error) {
	resp, err := http.Get(ip.gatewayURL + "/api/v0/stats/repo")
	if err != nil {
		return nil, fmt.Errorf("status check failed: %v", err)
	}
	defer resp.Body.Close()

	return map[string]string{"status": "ok"}, nil
}

// Utility to extract IPFS hash from response
func extractIPFSHash(response string) string {
	// Simplified - in production use JSON parser
	if idx := strings.Index(response, `"Hash"`); idx != -1 {
		start := strings.Index(response[idx:], `":"`) + idx + 3
		end := strings.Index(response[start:], `"`) + start
		return response[start:end]
	}
	return ""
}
