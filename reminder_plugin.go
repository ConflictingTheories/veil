package main

import (
	"context"
	"fmt"
	"time"
)

// === Reminder System Plugin ===

type ReminderPlugin struct {
	name    string
	version string
}

func NewReminderPlugin() *ReminderPlugin {
	return &ReminderPlugin{
		name:    "reminder",
		version: "1.0.0",
	}
}

func (rp *ReminderPlugin) Name() string {
	return rp.name
}

func (rp *ReminderPlugin) Version() string {
	return rp.version
}

func (rp *ReminderPlugin) Initialize(config map[string]interface{}) error {
	// Ensure reminders table exists
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS reminders (
			id TEXT PRIMARY KEY,
			node_id TEXT,
			title TEXT NOT NULL,
			description TEXT,
			remind_at INTEGER NOT NULL,
			status TEXT DEFAULT 'pending',
			recurrence TEXT,
			notification_sent INTEGER DEFAULT 0,
			created_at INTEGER NOT NULL,
			modified_at INTEGER NOT NULL,
			FOREIGN KEY (node_id) REFERENCES nodes(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to initialize reminders table: %v", err)
	}

	// Index for performance
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_reminders_node_id ON reminders(node_id)`)
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_reminders_remind_at ON reminders(remind_at)`)
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_reminders_status ON reminders(status)`)

	return nil
}

func (rp *ReminderPlugin) Validate() error {
	// No external dependencies required
	return nil
}

func (rp *ReminderPlugin) Execute(ctx context.Context, action string, payload interface{}) (interface{}, error) {
	switch action {
	case "create":
		return rp.createReminder(ctx, payload)
	case "list":
		return rp.listReminders(ctx, payload)
	case "get":
		return rp.getReminder(ctx, payload)
	case "update":
		return rp.updateReminder(ctx, payload)
	case "delete":
		return rp.deleteReminder(ctx, payload)
	case "dismiss":
		return rp.dismissReminder(ctx, payload)
	case "snooze":
		return rp.snoozeReminder(ctx, payload)
	case "pending":
		return rp.pendingReminders(ctx, payload)
	default:
		return nil, fmt.Errorf("unknown action: %s", action)
	}
}

func (rp *ReminderPlugin) Shutdown() error {
	return nil
}

// Actions

type Reminder struct {
	ID               string `json:"id"`
	NodeID           string `json:"node_id"`
	Title            string `json:"title"`
	Description      string `json:"description"`
	RemindAt         int64  `json:"remind_at"`
	Status           string `json:"status"`
	Recurrence       string `json:"recurrence,omitempty"` // none, daily, weekly, monthly, yearly
	NotificationSent int    `json:"notification_sent"`
	CreatedAt        int64  `json:"created_at"`
	ModifiedAt       int64  `json:"modified_at"`
}

func (rp *ReminderPlugin) createReminder(ctx context.Context, payload interface{}) (interface{}, error) {
	req, ok := payload.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid payload")
	}

	reminder := Reminder{
		ID:         fmt.Sprintf("reminder_%d", time.Now().UnixNano()),
		Title:      req["title"].(string),
		Status:     "pending",
		Recurrence: "none",
		CreatedAt:  time.Now().Unix(),
		ModifiedAt: time.Now().Unix(),
	}

	if nodeID, ok := req["node_id"].(string); ok {
		reminder.NodeID = nodeID
	}
	if desc, ok := req["description"].(string); ok {
		reminder.Description = desc
	}
	if remindAt, ok := req["remind_at"].(float64); ok {
		reminder.RemindAt = int64(remindAt)
	} else {
		return nil, fmt.Errorf("remind_at is required")
	}
	if recurrence, ok := req["recurrence"].(string); ok {
		reminder.Recurrence = recurrence
	}

	_, err := db.Exec(`
		INSERT INTO reminders (id, node_id, title, description, remind_at, status, recurrence, created_at, modified_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, reminder.ID, reminder.NodeID, reminder.Title, reminder.Description, reminder.RemindAt,
		reminder.Status, reminder.Recurrence, reminder.CreatedAt, reminder.ModifiedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create reminder: %v", err)
	}

	return reminder, nil
}

func (rp *ReminderPlugin) listReminders(ctx context.Context, payload interface{}) (interface{}, error) {
	req, ok := payload.(map[string]interface{})
	if !ok {
		req = make(map[string]interface{})
	}

	query := `SELECT id, COALESCE(node_id, ''), title, COALESCE(description, ''), remind_at, 
	          status, COALESCE(recurrence, 'none'), notification_sent, created_at, modified_at 
	          FROM reminders WHERE 1=1`
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

	query += " ORDER BY remind_at ASC"

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query reminders: %v", err)
	}
	defer rows.Close()

	var reminders []Reminder
	for rows.Next() {
		var reminder Reminder
		err := rows.Scan(&reminder.ID, &reminder.NodeID, &reminder.Title, &reminder.Description,
			&reminder.RemindAt, &reminder.Status, &reminder.Recurrence, &reminder.NotificationSent,
			&reminder.CreatedAt, &reminder.ModifiedAt)
		if err != nil {
			continue
		}
		reminders = append(reminders, reminder)
	}

	return reminders, nil
}

func (rp *ReminderPlugin) getReminder(ctx context.Context, payload interface{}) (interface{}, error) {
	req, ok := payload.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid payload")
	}

	reminderID, ok := req["id"].(string)
	if !ok {
		return nil, fmt.Errorf("reminder id required")
	}

	var reminder Reminder
	err := db.QueryRow(`
		SELECT id, COALESCE(node_id, ''), title, COALESCE(description, ''), remind_at,
		       status, COALESCE(recurrence, 'none'), notification_sent, created_at, modified_at
		FROM reminders WHERE id = ?
	`, reminderID).Scan(&reminder.ID, &reminder.NodeID, &reminder.Title, &reminder.Description,
		&reminder.RemindAt, &reminder.Status, &reminder.Recurrence, &reminder.NotificationSent,
		&reminder.CreatedAt, &reminder.ModifiedAt)

	if err != nil {
		return nil, fmt.Errorf("reminder not found: %v", err)
	}

	return reminder, nil
}

func (rp *ReminderPlugin) updateReminder(ctx context.Context, payload interface{}) (interface{}, error) {
	req, ok := payload.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid payload")
	}

	reminderID, ok := req["id"].(string)
	if !ok {
		return nil, fmt.Errorf("reminder id required")
	}

	updates := make(map[string]interface{})
	if title, ok := req["title"].(string); ok {
		updates["title"] = title
	}
	if desc, ok := req["description"].(string); ok {
		updates["description"] = desc
	}
	if remindAt, ok := req["remind_at"].(float64); ok {
		updates["remind_at"] = int64(remindAt)
	}
	if status, ok := req["status"].(string); ok {
		updates["status"] = status
	}
	if recurrence, ok := req["recurrence"].(string); ok {
		updates["recurrence"] = recurrence
	}

	if len(updates) == 0 {
		return nil, fmt.Errorf("no updates provided")
	}

	updates["modified_at"] = time.Now().Unix()

	// Build dynamic UPDATE query
	query := "UPDATE reminders SET "
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
	args = append(args, reminderID)

	_, err := db.Exec(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update reminder: %v", err)
	}

	return rp.getReminder(ctx, map[string]interface{}{"id": reminderID})
}

func (rp *ReminderPlugin) deleteReminder(ctx context.Context, payload interface{}) (interface{}, error) {
	req, ok := payload.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid payload")
	}

	reminderID, ok := req["id"].(string)
	if !ok {
		return nil, fmt.Errorf("reminder id required")
	}

	_, err := db.Exec(`DELETE FROM reminders WHERE id = ?`, reminderID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete reminder: %v", err)
	}

	return map[string]interface{}{"status": "deleted", "id": reminderID}, nil
}

func (rp *ReminderPlugin) dismissReminder(ctx context.Context, payload interface{}) (interface{}, error) {
	req, ok := payload.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid payload")
	}

	reminderID, ok := req["id"].(string)
	if !ok {
		return nil, fmt.Errorf("reminder id required")
	}

	now := time.Now().Unix()
	_, err := db.Exec(`
		UPDATE reminders SET status = 'dismissed', modified_at = ? WHERE id = ?
	`, now, reminderID)

	if err != nil {
		return nil, fmt.Errorf("failed to dismiss reminder: %v", err)
	}

	return rp.getReminder(ctx, map[string]interface{}{"id": reminderID})
}

func (rp *ReminderPlugin) snoozeReminder(ctx context.Context, payload interface{}) (interface{}, error) {
	req, ok := payload.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid payload")
	}

	reminderID, ok := req["id"].(string)
	if !ok {
		return nil, fmt.Errorf("reminder id required")
	}

	// Default snooze for 15 minutes
	snoozeMinutes := 15
	if sm, ok := req["snooze_minutes"].(float64); ok {
		snoozeMinutes = int(sm)
	}

	newRemindAt := time.Now().Add(time.Duration(snoozeMinutes) * time.Minute).Unix()
	now := time.Now().Unix()

	_, err := db.Exec(`
		UPDATE reminders SET remind_at = ?, notification_sent = 0, modified_at = ? WHERE id = ?
	`, newRemindAt, now, reminderID)

	if err != nil {
		return nil, fmt.Errorf("failed to snooze reminder: %v", err)
	}

	return rp.getReminder(ctx, map[string]interface{}{"id": reminderID})
}

func (rp *ReminderPlugin) pendingReminders(ctx context.Context, payload interface{}) (interface{}, error) {
	now := time.Now().Unix()

	rows, err := db.Query(`
		SELECT id, COALESCE(node_id, ''), title, COALESCE(description, ''), remind_at,
		       status, COALESCE(recurrence, 'none'), notification_sent, created_at, modified_at
		FROM reminders 
		WHERE status = 'pending' AND remind_at <= ? AND notification_sent = 0
		ORDER BY remind_at ASC
	`, now)

	if err != nil {
		return nil, fmt.Errorf("failed to query pending reminders: %v", err)
	}
	defer rows.Close()

	var reminders []Reminder
	for rows.Next() {
		var reminder Reminder
		err := rows.Scan(&reminder.ID, &reminder.NodeID, &reminder.Title, &reminder.Description,
			&reminder.RemindAt, &reminder.Status, &reminder.Recurrence, &reminder.NotificationSent,
			&reminder.CreatedAt, &reminder.ModifiedAt)
		if err != nil {
			continue
		}
		reminders = append(reminders, reminder)

		// Mark as notified
		db.Exec(`UPDATE reminders SET notification_sent = 1 WHERE id = ?`, reminder.ID)

		// Handle recurrence
		if reminder.Recurrence != "none" && reminder.Recurrence != "" {
			nextRemindAt := calculateNextRecurrence(reminder.RemindAt, reminder.Recurrence)
			newReminderID := fmt.Sprintf("reminder_%d", time.Now().UnixNano())
			db.Exec(`
				INSERT INTO reminders (id, node_id, title, description, remind_at, status, recurrence, created_at, modified_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			`, newReminderID, reminder.NodeID, reminder.Title, reminder.Description, nextRemindAt,
				"pending", reminder.Recurrence, time.Now().Unix(), time.Now().Unix())
		}
	}

	return reminders, nil
}

// Helper function to calculate next recurrence time
func calculateNextRecurrence(currentTime int64, recurrence string) int64 {
	t := time.Unix(currentTime, 0)

	switch recurrence {
	case "daily":
		return t.Add(24 * time.Hour).Unix()
	case "weekly":
		return t.Add(7 * 24 * time.Hour).Unix()
	case "monthly":
		return t.AddDate(0, 1, 0).Unix()
	case "yearly":
		return t.AddDate(1, 0, 0).Unix()
	default:
		return currentTime
	}
}
