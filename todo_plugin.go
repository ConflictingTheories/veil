package main

import (
	"context"
	"fmt"
	"time"
)

// === Todo System Plugin ===

type TodoPlugin struct {
	name    string
	version string
}

func NewTodoPlugin() *TodoPlugin {
	return &TodoPlugin{
		name:    "todo",
		version: "1.0.0",
	}
}

func (tp *TodoPlugin) Name() string {
	return tp.name
}

func (tp *TodoPlugin) Version() string {
	return tp.version
}

func (tp *TodoPlugin) Initialize(config map[string]interface{}) error {
	// Ensure todos table exists
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS todos (
			id TEXT PRIMARY KEY,
			node_id TEXT,
			title TEXT NOT NULL,
			description TEXT,
			status TEXT DEFAULT 'pending',
			priority TEXT DEFAULT 'medium',
			due_date INTEGER,
			assigned_to TEXT,
			completed_at INTEGER,
			created_at INTEGER NOT NULL,
			modified_at INTEGER NOT NULL,
			FOREIGN KEY (node_id) REFERENCES nodes(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to initialize todos table: %v", err)
	}

	// Index for performance
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_todos_node_id ON todos(node_id)`)
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_todos_status ON todos(status)`)
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_todos_due_date ON todos(due_date)`)

	return nil
}

func (tp *TodoPlugin) Validate() error {
	// No external dependencies required
	return nil
}

func (tp *TodoPlugin) Execute(ctx context.Context, action string, payload interface{}) (interface{}, error) {
	switch action {
	case "create":
		return tp.createTodo(ctx, payload)
	case "list":
		return tp.listTodos(ctx, payload)
	case "get":
		return tp.getTodo(ctx, payload)
	case "update":
		return tp.updateTodo(ctx, payload)
	case "delete":
		return tp.deleteTodo(ctx, payload)
	case "complete":
		return tp.completeTodo(ctx, payload)
	case "reopen":
		return tp.reopenTodo(ctx, payload)
	default:
		return nil, fmt.Errorf("unknown action: %s", action)
	}
}

func (tp *TodoPlugin) Shutdown() error {
	return nil
}

// Actions

type Todo struct {
	ID          string `json:"id"`
	NodeID      string `json:"node_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
	Priority    string `json:"priority"`
	DueDate     int64  `json:"due_date,omitempty"`
	AssignedTo  string `json:"assigned_to,omitempty"`
	CompletedAt int64  `json:"completed_at,omitempty"`
	CreatedAt   int64  `json:"created_at"`
	ModifiedAt  int64  `json:"modified_at"`
}

func (tp *TodoPlugin) createTodo(ctx context.Context, payload interface{}) (interface{}, error) {
	req, ok := payload.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid payload")
	}

	todo := Todo{
		ID:         fmt.Sprintf("todo_%d", time.Now().UnixNano()),
		Title:      req["title"].(string),
		Status:     "pending",
		Priority:   "medium",
		CreatedAt:  time.Now().Unix(),
		ModifiedAt: time.Now().Unix(),
	}

	if nodeID, ok := req["node_id"].(string); ok {
		todo.NodeID = nodeID
	}
	if desc, ok := req["description"].(string); ok {
		todo.Description = desc
	}
	if priority, ok := req["priority"].(string); ok {
		todo.Priority = priority
	}
	if dueDate, ok := req["due_date"].(float64); ok {
		todo.DueDate = int64(dueDate)
	}
	if assignedTo, ok := req["assigned_to"].(string); ok {
		todo.AssignedTo = assignedTo
	}

	_, err := db.Exec(`
		INSERT INTO todos (id, node_id, title, description, status, priority, due_date, assigned_to, created_at, modified_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, todo.ID, todo.NodeID, todo.Title, todo.Description, todo.Status, todo.Priority, todo.DueDate, todo.AssignedTo, todo.CreatedAt, todo.ModifiedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create todo: %v", err)
	}

	return todo, nil
}

func (tp *TodoPlugin) listTodos(ctx context.Context, payload interface{}) (interface{}, error) {
	req, ok := payload.(map[string]interface{})
	if !ok {
		req = make(map[string]interface{})
	}

	query := `SELECT id, COALESCE(node_id, ''), title, COALESCE(description, ''), status, priority, 
	          COALESCE(due_date, 0), COALESCE(assigned_to, ''), COALESCE(completed_at, 0), created_at, modified_at 
	          FROM todos WHERE 1=1`
	args := []interface{}{}

	// Filter by node_id
	if nodeID, ok := req["node_id"].(string); ok && nodeID != "" {
		query += " AND node_id = ?"
		args = append(args, nodeID)
	}

	// Filter by status
	if status, ok := req["status"].(string); ok && status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}

	// Filter by priority
	if priority, ok := req["priority"].(string); ok && priority != "" {
		query += " AND priority = ?"
		args = append(args, priority)
	}

	query += " ORDER BY due_date ASC, created_at DESC"

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query todos: %v", err)
	}
	defer rows.Close()

	var todos []Todo
	for rows.Next() {
		var todo Todo
		err := rows.Scan(&todo.ID, &todo.NodeID, &todo.Title, &todo.Description, &todo.Status,
			&todo.Priority, &todo.DueDate, &todo.AssignedTo, &todo.CompletedAt, &todo.CreatedAt, &todo.ModifiedAt)
		if err != nil {
			continue
		}
		todos = append(todos, todo)
	}

	return todos, nil
}

func (tp *TodoPlugin) getTodo(ctx context.Context, payload interface{}) (interface{}, error) {
	req, ok := payload.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid payload")
	}

	todoID, ok := req["id"].(string)
	if !ok {
		return nil, fmt.Errorf("todo id required")
	}

	var todo Todo
	err := db.QueryRow(`
		SELECT id, COALESCE(node_id, ''), title, COALESCE(description, ''), status, priority,
		       COALESCE(due_date, 0), COALESCE(assigned_to, ''), COALESCE(completed_at, 0), created_at, modified_at
		FROM todos WHERE id = ?
	`, todoID).Scan(&todo.ID, &todo.NodeID, &todo.Title, &todo.Description, &todo.Status,
		&todo.Priority, &todo.DueDate, &todo.AssignedTo, &todo.CompletedAt, &todo.CreatedAt, &todo.ModifiedAt)

	if err != nil {
		return nil, fmt.Errorf("todo not found: %v", err)
	}

	return todo, nil
}

func (tp *TodoPlugin) updateTodo(ctx context.Context, payload interface{}) (interface{}, error) {
	req, ok := payload.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid payload")
	}

	todoID, ok := req["id"].(string)
	if !ok {
		return nil, fmt.Errorf("todo id required")
	}

	updates := make(map[string]interface{})
	if title, ok := req["title"].(string); ok {
		updates["title"] = title
	}
	if desc, ok := req["description"].(string); ok {
		updates["description"] = desc
	}
	if status, ok := req["status"].(string); ok {
		updates["status"] = status
	}
	if priority, ok := req["priority"].(string); ok {
		updates["priority"] = priority
	}
	if dueDate, ok := req["due_date"].(float64); ok {
		updates["due_date"] = int64(dueDate)
	}
	if assignedTo, ok := req["assigned_to"].(string); ok {
		updates["assigned_to"] = assignedTo
	}

	if len(updates) == 0 {
		return nil, fmt.Errorf("no updates provided")
	}

	updates["modified_at"] = time.Now().Unix()

	// Build dynamic UPDATE query
	query := "UPDATE todos SET "
	args := []interface{}{}
	i := 0
	for k, v := range updates {
		if i > 0 {
			query += ", "
		}
		query += k + " = ?"
		args = append(args, v)
		i++
	}
	query += " WHERE id = ?"
	args = append(args, todoID)

	_, err := db.Exec(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update todo: %v", err)
	}

	return tp.getTodo(ctx, map[string]interface{}{"id": todoID})
}

func (tp *TodoPlugin) deleteTodo(ctx context.Context, payload interface{}) (interface{}, error) {
	req, ok := payload.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid payload")
	}

	todoID, ok := req["id"].(string)
	if !ok {
		return nil, fmt.Errorf("todo id required")
	}

	_, err := db.Exec(`DELETE FROM todos WHERE id = ?`, todoID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete todo: %v", err)
	}

	return map[string]interface{}{"status": "deleted", "id": todoID}, nil
}

func (tp *TodoPlugin) completeTodo(ctx context.Context, payload interface{}) (interface{}, error) {
	req, ok := payload.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid payload")
	}

	todoID, ok := req["id"].(string)
	if !ok {
		return nil, fmt.Errorf("todo id required")
	}

	now := time.Now().Unix()
	_, err := db.Exec(`
		UPDATE todos SET status = 'completed', completed_at = ?, modified_at = ? WHERE id = ?
	`, now, now, todoID)

	if err != nil {
		return nil, fmt.Errorf("failed to complete todo: %v", err)
	}

	return tp.getTodo(ctx, map[string]interface{}{"id": todoID})
}

func (tp *TodoPlugin) reopenTodo(ctx context.Context, payload interface{}) (interface{}, error) {
	req, ok := payload.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid payload")
	}

	todoID, ok := req["id"].(string)
	if !ok {
		return nil, fmt.Errorf("todo id required")
	}

	now := time.Now().Unix()
	_, err := db.Exec(`
		UPDATE todos SET status = 'pending', completed_at = NULL, modified_at = ? WHERE id = ?
	`, now, todoID)

	if err != nil {
		return nil, fmt.Errorf("failed to reopen todo: %v", err)
	}

	return tp.getTodo(ctx, map[string]interface{}{"id": todoID})
}
