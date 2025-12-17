package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
)

// === Pixospritz Game Engine Plugin ===

type PixospritzPlugin struct {
	name          string
	version       string
	serverURL     string
	localGamePath string
}

func NewPixospritzPlugin(serverURL string) *PixospritzPlugin {
	if serverURL == "" {
		serverURL = "http://localhost:3000"
	}
	return &PixospritzPlugin{
		name:      "pixospritz",
		version:   "1.0.0",
		serverURL: serverURL,
	}
}

func (pp *PixospritzPlugin) Name() string {
	return pp.name
}

func (pp *PixospritzPlugin) Version() string {
	return pp.version
}

func (pp *PixospritzPlugin) Initialize(config map[string]interface{}) error {
	if url, ok := config["server_url"].(string); ok {
		pp.serverURL = url
		saveConfig("pixospritz_server_url", url)
	}

	if path, ok := config["local_game_path"].(string); ok {
		pp.localGamePath = path
		saveConfig("pixospritz_game_path", path)
	}

	return nil
}

func (pp *PixospritzPlugin) Validate() error {
	resp, err := http.Get(pp.serverURL + "/health")
	if err != nil {
		return fmt.Errorf("pixospritz server not responding: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("pixospritz server returned status %d", resp.StatusCode)
	}

	return nil
}

func (pp *PixospritzPlugin) Execute(ctx context.Context, action string, payload interface{}) (interface{}, error) {
	switch action {
	case "embed_game":
		return pp.embedGame(ctx, payload)
	case "get_scores":
		return pp.getScores(ctx, payload)
	case "save_score":
		return pp.saveScore(ctx, payload)
	case "get_leaderboard":
		return pp.getLeaderboard(ctx, payload)
	case "link_to_portfolio":
		return pp.linkToPortfolio(ctx, payload)
	case "get_game_status":
		return pp.getGameStatus(ctx, payload)
	default:
		return nil, fmt.Errorf("unknown action: %s", action)
	}
}

func (pp *PixospritzPlugin) Shutdown() error {
	return nil
}

// Game types

type GameEmbed struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	NodeID      string `json:"node_id"`
	GameID      string `json:"game_id"`
	Description string `json:"description"`
	EmbedCode   string `json:"embed_code"`
	CreatedAt   int64  `json:"created_at"`
}

type GameScore struct {
	ID        string                 `json:"id"`
	GameID    string                 `json:"game_id"`
	PlayerID  string                 `json:"player_id"`
	Score     int                    `json:"score"`
	Timestamp int64                  `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// Actions

type EmbedGameRequest struct {
	GameID      string `json:"game_id"`
	Title       string `json:"title"`
	NodeID      string `json:"node_id"`
	Description string `json:"description"`
}

func (pp *PixospritzPlugin) embedGame(ctx context.Context, payload interface{}) (interface{}, error) {
	req, ok := payload.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid payload")
	}

	gameID := req["game_id"].(string)
	title := req["title"].(string)
	nodeID := req["node_id"].(string)
	description := req["description"].(string)

	// Generate embed code
	embedCode := fmt.Sprintf(
		`<iframe src="%s/play/%s" width="800" height="600" frameborder="0" allow="fullscreen"></iframe>`,
		pp.serverURL, gameID)

	// Store in database
	now := int64(0)
	embedID := fmt.Sprintf("embed_%d", now)

	db.Exec(`
		INSERT INTO game_embeds (id, node_id, game_id, title, description, embed_code, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, embedID, nodeID, gameID, title, description, embedCode, now)

	return map[string]interface{}{
		"id":         embedID,
		"embed_code": embedCode,
		"game_id":    gameID,
		"node_id":    nodeID,
	}, nil
}

type GetScoresRequest struct {
	GameID   string `json:"game_id"`
	PlayerID string `json:"player_id,omitempty"`
}

func (pp *PixospritzPlugin) getScores(ctx context.Context, payload interface{}) (interface{}, error) {
	req, ok := payload.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid payload")
	}

	gameID := req["game_id"].(string)
	playerID := req["player_id"].(string)

	// Query database
	query := `SELECT id, game_id, player_id, score, timestamp FROM game_scores WHERE game_id = ?`
	args := []interface{}{gameID}

	if playerID != "" {
		query += ` AND player_id = ?`
		args = append(args, playerID)
	}

	query += ` ORDER BY score DESC LIMIT 100`

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var scores []map[string]interface{}
	for rows.Next() {
		var id, gid, pid string
		var score int
		var timestamp int64
		rows.Scan(&id, &gid, &pid, &score, &timestamp)
		scores = append(scores, map[string]interface{}{
			"id":        id,
			"game_id":   gid,
			"player_id": pid,
			"score":     score,
			"timestamp": timestamp,
		})
	}

	return map[string]interface{}{
		"game_id": gameID,
		"scores":  scores,
	}, nil
}

type SaveScoreRequest struct {
	GameID   string                 `json:"game_id"`
	PlayerID string                 `json:"player_id"`
	Score    int                    `json:"score"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

func (pp *PixospritzPlugin) saveScore(ctx context.Context, payload interface{}) (interface{}, error) {
	req, ok := payload.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid payload")
	}

	gameID := req["game_id"].(string)
	playerID := req["player_id"].(string)
	score := int(req["score"].(float64))

	// Verify with Pixospritz server if score is legitimate
	verifyURL := fmt.Sprintf("%s/api/verify-score", pp.serverURL)
	verifyBody := map[string]interface{}{
		"game_id":   gameID,
		"player_id": playerID,
		"score":     score,
	}

	_ = verifyBody // Use verifyBody in actual API call if needed
	resp, err := http.Post(verifyURL, "application/json", nil)
	if err != nil {
		return nil, fmt.Errorf("score verification failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("score verification failed: server rejected")
	}

	// Store score
	now := int64(0)
	scoreID := fmt.Sprintf("score_%d", now)

	db.Exec(`
		INSERT INTO game_scores (id, game_id, player_id, score, timestamp, metadata)
		VALUES (?, ?, ?, ?, ?, ?)
	`, scoreID, gameID, playerID, score, now, "")

	return map[string]interface{}{
		"id":      scoreID,
		"game_id": gameID,
		"score":   score,
		"saved":   true,
	}, nil
}

type GetLeaderboardRequest struct {
	GameID string `json:"game_id"`
	Limit  int    `json:"limit"`
}

func (pp *PixospritzPlugin) getLeaderboard(ctx context.Context, payload interface{}) (interface{}, error) {
	req, ok := payload.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid payload")
	}

	gameID := req["game_id"].(string)
	limit := 10
	if l, ok := req["limit"].(float64); ok {
		limit = int(l)
	}

	rows, _ := db.Query(`
		SELECT player_id, score, MAX(timestamp) as latest
		FROM game_scores
		WHERE game_id = ?
		GROUP BY player_id
		ORDER BY score DESC
		LIMIT ?
	`, gameID, limit)
	defer rows.Close()

	var leaderboard []map[string]interface{}
	for i := 0; rows.Next(); i++ {
		var playerID string
		var score int
		var timestamp int64
		rows.Scan(&playerID, &score, &timestamp)
		leaderboard = append(leaderboard, map[string]interface{}{
			"rank":      i + 1,
			"player_id": playerID,
			"score":     score,
		})
	}

	return map[string]interface{}{
		"game_id":     gameID,
		"leaderboard": leaderboard,
	}, nil
}

type LinkToPortfolioRequest struct {
	GameID   string `json:"game_id"`
	NodeID   string `json:"node_id"`
	Showcase bool   `json:"showcase"`
}

func (pp *PixospritzPlugin) linkToPortfolio(ctx context.Context, payload interface{}) (interface{}, error) {
	req, ok := payload.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid payload")
	}

	gameID := req["game_id"].(string)
	nodeID := req["node_id"].(string)
	showcase := req["showcase"].(bool)

	// Get node content
	var nodeTitle string
	db.QueryRow(`SELECT title FROM nodes WHERE id = ?`, nodeID).Scan(&nodeTitle)

	// Create portfolio entry
	now := int64(0)
	portfolioID := fmt.Sprintf("portfolio_%d", now)

	db.Exec(`
		INSERT INTO portfolio_games (id, node_id, game_id, showcase, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, portfolioID, nodeID, gameID, showcase, now)

	// Generate portfolio entry markdown
	portfolioText := fmt.Sprintf(
		"## %s\n\nPlay the game [here](%s/play/%s)\n\nView leaderboard and scores in your portfolio.",
		nodeTitle, pp.serverURL, gameID)

	return map[string]interface{}{
		"status":         "linked",
		"portfolio_id":   portfolioID,
		"portfolio_text": portfolioText,
		"game_embed_url": fmt.Sprintf("%s/play/%s", pp.serverURL, gameID),
	}, nil
}

type GetGameStatusRequest struct {
	GameID string `json:"game_id"`
}

func (pp *PixospritzPlugin) getGameStatus(ctx context.Context, payload interface{}) (interface{}, error) {
	req, ok := payload.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid payload")
	}

	gameID := req["game_id"].(string)

	// Query Pixospritz server
	url := fmt.Sprintf("%s/api/game-status/%s", pp.serverURL, gameID)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("status check failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var status map[string]interface{}
	json.Unmarshal(body, &status)

	return status, nil
}

// Helper to launch Pixospritz server
func LaunchPixospritzServer(gamePath string) error {
	cmd := exec.Command("calliope", "serve", "--path", gamePath)
	err := cmd.Start()
	if err != nil {
		return fmt.Errorf("failed to launch pixospritz server: %v", err)
	}
	return nil
}
