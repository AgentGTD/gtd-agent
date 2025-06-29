package chat

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"encore.dev/storage/sqldb"
)

// ChatRequest represents the incoming Google Chat webhook request
type ChatRequest struct {
	Message struct {
		Text   string `json:"text"`
		Sender struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		} `json:"sender"`
	} `json:"message"`
}

// ChatResponse represents the response to Google Chat
type ChatResponse struct {
	Text string `json:"text"`
}

//encore:api public method=POST path=/chat
func HandleChat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	text := strings.TrimSpace(req.Message.Text)

	// Get user identifier (prefer email, fallback to name, then default)
	userID := "default"
	if req.Message.Sender.Email != "" {
		userID = req.Message.Sender.Email
	} else if req.Message.Sender.Name != "" {
		userID = req.Message.Sender.Name
	}

	// Parse commands
	addCmd := regexp.MustCompile(`^add\s+(.+)$`)
	listCmd := regexp.MustCompile(`^list$`)
	doneCmd := regexp.MustCompile(`^done\s+(\d+)$`)

	switch {
	case addCmd.MatchString(text):
		matches := addCmd.FindStringSubmatch(text)
		taskContent := strings.TrimSpace(matches[1])
		response, err := addTask(ctx, taskContent, userID)
		if err != nil {
			return nil, fmt.Errorf("failed to add task: %w", err)
		}
		return &ChatResponse{Text: response}, nil

	case listCmd.MatchString(text):
		response, err := listTasks(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("failed to list tasks: %w", err)
		}
		return &ChatResponse{Text: response}, nil

	case doneCmd.MatchString(text):
		matches := doneCmd.FindStringSubmatch(text)
		taskID, _ := strconv.Atoi(matches[1])
		response, err := markTaskDone(ctx, taskID, userID)
		if err != nil {
			return nil, fmt.Errorf("failed to mark task as done: %w", err)
		}
		return &ChatResponse{Text: response}, nil

	default:
		return &ChatResponse{Text: `Available commands:
• add <task> - Add a new task
• list - List all tasks
• done <id> - Mark task as done`}, nil
	}
}

func addTask(ctx context.Context, content string, userID string) (string, error) {
	var id int
	err := sqldb.QueryRow(ctx, `
		INSERT INTO tasks (content, user_id) 
		VALUES ($1, $2) 
		RETURNING id
	`, content, userID).Scan(&id)

	if err != nil {
		return "", err
	}

	return fmt.Sprintf("✅ Task added with ID: %d", id), nil
}

func listTasks(ctx context.Context, userID string) (string, error) {
	rows, err := sqldb.Query(ctx, `
		SELECT id, content, done 
		FROM tasks 
		WHERE user_id = $1
		ORDER BY id
	`, userID)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var tasks []string
	for rows.Next() {
		var id int
		var content string
		var done bool

		if err := rows.Scan(&id, &content, &done); err != nil {
			return "", err
		}

		status := "❌"
		if done {
			status = "✅"
		}

		tasks = append(tasks, fmt.Sprintf("%d. %s %s", id, status, content))
	}

	if err := rows.Err(); err != nil {
		return "", err
	}

	if len(tasks) == 0 {
		return "📝 No tasks found. Use 'add <task>' to create your first task!", nil
	}

	return "📋 Your tasks:\n" + strings.Join(tasks, "\n"), nil
}

func markTaskDone(ctx context.Context, taskID int, userID string) (string, error) {
	result, err := sqldb.Exec(ctx, `
		UPDATE tasks 
		SET done = true 
		WHERE id = $1 AND user_id = $2
	`, taskID, userID)

	if err != nil {
		return "", err
	}

	rowsAffected := result.RowsAffected()

	if rowsAffected == 0 {
		return fmt.Sprintf("❌ Task with ID %d not found or doesn't belong to you", taskID), nil
	}

	return fmt.Sprintf("✅ Task %d marked as done!", taskID), nil
}
