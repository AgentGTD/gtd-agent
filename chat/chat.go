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
	Action *Action `json:"action,omitempty"`
}

// Action represents a card action (button click)
type Action struct {
	ActionMethodName string `json:"actionMethodName"`
	Parameters       []struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	} `json:"parameters"`
}

// ChatResponse represents the response to Google Chat webhook
type ChatResponse struct {
	Text  string `json:"text,omitempty"`
	Cards []Card `json:"cards,omitempty"`
}

// Alternative response format for debugging
type ChatResponseV2 struct {
	Text  string `json:"text,omitempty"`
	Cards []Card `json:"cards,omitempty"`
}

// Card represents a Google Chat card
type Card struct {
	Header   *CardHeader   `json:"header,omitempty"`
	Sections []CardSection `json:"sections"`
}

// CardHeader represents a card header
type CardHeader struct {
	Title    string `json:"title"`
	Subtitle string `json:"subtitle,omitempty"`
}

// CardSection represents a card section
type CardSection struct {
	Widgets []Widget `json:"widgets"`
}

// Widget represents a card widget
type Widget struct {
	TextParagraph *TextParagraph `json:"textParagraph,omitempty"`
	ButtonList    *ButtonList    `json:"buttonList,omitempty"`
	Divider       *Divider       `json:"divider,omitempty"`
}

// TextParagraph represents a text paragraph widget
type TextParagraph struct {
	Text string `json:"text"`
}

// ButtonList represents a button list widget
type ButtonList struct {
	Buttons []Button `json:"buttons"`
}

// Button represents a button
type Button struct {
	TextButton *TextButton `json:"textButton,omitempty"`
}

// TextButton represents a text button
type TextButton struct {
	Text    string  `json:"text"`
	OnClick OnClick `json:"onClick"`
}

// OnClick represents a button click action
type OnClick struct {
	Action CardAction `json:"action"`
}

// CardAction represents an action for card buttons
type CardAction struct {
	ActionMethodName string            `json:"actionMethodName"`
	Parameters       map[string]string `json:"parameters"`
}

// Divider represents a divider widget
type Divider struct{}

//encore:api public method=POST path=/chat
func HandleChat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	// Handle card actions (button clicks)
	if req.Action != nil {
		return handleCardAction(ctx, req)
	}

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
	editCmd := regexp.MustCompile(`^edit\s+(\d+)\s+(.+)$`)
	testCmd := regexp.MustCompile(`^test$`)

	switch {
	case addCmd.MatchString(text):
		matches := addCmd.FindStringSubmatch(text)
		taskContent := strings.TrimSpace(matches[1])
		response, err := addTask(ctx, taskContent, userID)
		if err != nil {
			return nil, fmt.Errorf("failed to add task: %w", err)
		}
		return response, nil

	case listCmd.MatchString(text):
		response, err := listTasks(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("failed to list tasks: %w", err)
		}
		return response, nil

	case doneCmd.MatchString(text):
		matches := doneCmd.FindStringSubmatch(text)
		taskID, _ := strconv.Atoi(matches[1])
		response, err := markTaskDone(ctx, taskID, userID)
		if err != nil {
			return nil, fmt.Errorf("failed to mark task as done: %w", err)
		}
		return response, nil

	case editCmd.MatchString(text):
		matches := editCmd.FindStringSubmatch(text)
		taskID, _ := strconv.Atoi(matches[1])
		newContent := strings.TrimSpace(matches[2])
		response, err := editTask(ctx, taskID, newContent, userID)
		if err != nil {
			return nil, fmt.Errorf("failed to edit task: %w", err)
		}
		return response, nil

	case testCmd.MatchString(text):
		return &ChatResponse{Text: "🧪 Test command working! The bot is responding correctly."}, nil

	default:
		return &ChatResponse{Text: `Available commands:
• add <task> - Add a new task
• list - List all tasks
• done <id> - Mark task as done
• edit <id> <new content> - Edit a task
• test - Test bot functionality`}, nil
	}
}

//encore:api public method=POST path=/card-action
func HandleCardAction(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	return handleCardAction(ctx, req)
}

func handleCardAction(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	if req.Action == nil {
		return &ChatResponse{Text: "❌ Invalid action"}, nil
	}

	// Get user identifier
	userID := "default"
	if req.Message.Sender.Email != "" {
		userID = req.Message.Sender.Email
	} else if req.Message.Sender.Name != "" {
		userID = req.Message.Sender.Name
	}

	// Extract parameters
	params := make(map[string]string)
	for _, param := range req.Action.Parameters {
		params[param.Key] = param.Value
	}

	switch req.Action.ActionMethodName {
	case "markDone":
		taskIDStr := params["taskId"]
		taskID, err := strconv.Atoi(taskIDStr)
		if err != nil {
			return &ChatResponse{Text: "❌ Invalid task ID"}, nil
		}
		return markTaskDone(ctx, taskID, userID)

	case "deleteTask":
		taskIDStr := params["taskId"]
		taskID, err := strconv.Atoi(taskIDStr)
		if err != nil {
			return &ChatResponse{Text: "❌ Invalid task ID"}, nil
		}
		return deleteTask(ctx, taskID, userID)

	case "editTask":
		taskIDStr := params["taskId"]
		taskID, err := strconv.Atoi(taskIDStr)
		if err != nil {
			return &ChatResponse{Text: "❌ Invalid task ID"}, nil
		}

		// If content is provided, update the task
		if newContent := params["content"]; newContent != "" {
			return editTask(ctx, taskID, newContent, userID)
		}

		// Otherwise, show the edit form
		return showEditForm(ctx, taskID, userID)

	case "list":
		return listTasks(ctx, userID)

	default:
		return &ChatResponse{Text: "❌ Unknown action"}, nil
	}
}

func showEditForm(ctx context.Context, taskID int, userID string) (*ChatResponse, error) {
	// Get the current task content
	var content string
	err := sqldb.QueryRow(ctx, `
		SELECT content FROM tasks WHERE id = $1 AND user_id = $2
	`, taskID, userID).Scan(&content)

	if err != nil {
		return &ChatResponse{Text: fmt.Sprintf("❌ Task with ID %d not found or doesn't belong to you", taskID)}, nil
	}

	return &ChatResponse{Text: fmt.Sprintf("✏️ Edit Task #%d\nCurrent content: %s\n\nTo edit this task, use the command:\nedit %d <new content>", taskID, content, taskID)}, nil
}

func addTask(ctx context.Context, content string, userID string) (*ChatResponse, error) {
	var id int
	err := sqldb.QueryRow(ctx, `
		INSERT INTO tasks (content, user_id) 
		VALUES ($1, $2) 
		RETURNING id
	`, content, userID).Scan(&id)

	if err != nil {
		return nil, err
	}

	// Return a simple text response for now
	return &ChatResponse{Text: fmt.Sprintf("✅ Task added with ID: %d\nContent: %s", id, content)}, nil
}

func listTasks(ctx context.Context, userID string) (*ChatResponse, error) {
	rows, err := sqldb.Query(ctx, `
		SELECT id, content, done 
		FROM tasks 
		WHERE user_id = $1
		ORDER BY id
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []string
	for rows.Next() {
		var id int
		var content string
		var done bool

		if err := rows.Scan(&id, &content, &done); err != nil {
			return nil, err
		}

		status := "❌"
		if done {
			status = "✅"
		}

		tasks = append(tasks, fmt.Sprintf("%d. %s %s", id, status, content))
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(tasks) == 0 {
		return &ChatResponse{Text: "📝 No tasks found. Use 'add <task>' to create your first task!"}, nil
	}

	return &ChatResponse{Text: "📋 Your tasks:\n" + strings.Join(tasks, "\n")}, nil
}

func markTaskDone(ctx context.Context, taskID int, userID string) (*ChatResponse, error) {
	result, err := sqldb.Exec(ctx, `
		UPDATE tasks 
		SET done = true 
		WHERE id = $1 AND user_id = $2
	`, taskID, userID)

	if err != nil {
		return nil, err
	}

	rowsAffected := result.RowsAffected()

	if rowsAffected == 0 {
		return &ChatResponse{Text: fmt.Sprintf("❌ Task with ID %d not found or doesn't belong to you", taskID)}, nil
	}

	return &ChatResponse{Text: fmt.Sprintf("✅ Task %d marked as done!", taskID)}, nil
}

func deleteTask(ctx context.Context, taskID int, userID string) (*ChatResponse, error) {
	result, err := sqldb.Exec(ctx, `
		DELETE FROM tasks 
		WHERE id = $1 AND user_id = $2
	`, taskID, userID)

	if err != nil {
		return nil, err
	}

	rowsAffected := result.RowsAffected()

	if rowsAffected == 0 {
		return &ChatResponse{Text: fmt.Sprintf("❌ Task with ID %d not found or doesn't belong to you", taskID)}, nil
	}

	return &ChatResponse{Text: fmt.Sprintf("🗑️ Task %d deleted!", taskID)}, nil
}

func editTask(ctx context.Context, taskID int, newContent string, userID string) (*ChatResponse, error) {
	if newContent == "" {
		return &ChatResponse{Text: "❌ Task content cannot be empty"}, nil
	}

	result, err := sqldb.Exec(ctx, `
		UPDATE tasks 
		SET content = $1 
		WHERE id = $2 AND user_id = $3
	`, newContent, taskID, userID)

	if err != nil {
		return nil, err
	}

	rowsAffected := result.RowsAffected()

	if rowsAffected == 0 {
		return &ChatResponse{Text: fmt.Sprintf("❌ Task with ID %d not found or doesn't belong to you", taskID)}, nil
	}

	return &ChatResponse{Text: fmt.Sprintf("✏️ Task %d updated to: %s", taskID, newContent)}, nil
}
