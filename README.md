# GTD Agent - Google Chat Bot

A Getting Things Done (GTD) bot for Google Chat that helps you manage tasks through simple commands.

## Features

- **Add tasks**: Create new tasks with the `add` command
- **List tasks**: View all your tasks with their completion status
- **Mark tasks as done**: Complete tasks using the `done` command

## Commands

| Command | Description | Example |
|---------|-------------|---------|
| `add <task>` | Add a new task | `add Buy groceries` |
| `list` | List all tasks | `list` |
| `done <id>` | Mark task as done | `done 1` |

## Setup

### 1. Deploy to Encore

```bash
# Deploy the application
encore app create gtd-agent
encore run
```

### 2. Configure Google Chat Webhook

1. Go to [Google Chat API](https://developers.google.com/chat/api/guides/message-formats/basic)
2. Create a new webhook
3. Set the webhook URL to: `https://your-app.encore.app/chat`
4. Configure the webhook to send messages to your chat

### 3. Database Setup

The application automatically creates the required database table when deployed. The `tasks` table has the following structure:

```sql
CREATE TABLE tasks (
    id SERIAL PRIMARY KEY,
    content TEXT NOT NULL,
    done BOOLEAN DEFAULT FALSE
);
```

## API Endpoint

- **URL**: `/chat`
- **Method**: `POST`
- **Content-Type**: `application/json`

### Request Format

```json
{
  "message": {
    "text": "add Buy groceries"
  }
}
```

### Response Format

```json
{
  "text": "✅ Task added with ID: 1"
}
```

## Development

### Prerequisites

- Go 1.24.2 or later
- Encore CLI

### Running Locally

```bash
# Install dependencies
go mod tidy

# Run the application
encore run
```

The application will be available at `http://localhost:4000/chat` for local development.

## Architecture

- **Framework**: Encore (Go)
- **Database**: PostgreSQL (managed by Encore)
- **API**: RESTful webhook endpoint
- **Pattern**: Command-based message processing

The bot uses regular expressions to parse incoming messages and route them to the appropriate command handlers. Each command interacts with the PostgreSQL database to perform CRUD operations on tasks. 