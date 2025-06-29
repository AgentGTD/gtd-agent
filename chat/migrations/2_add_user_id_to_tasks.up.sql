ALTER TABLE tasks ADD COLUMN user_id VARCHAR(255) NOT NULL DEFAULT 'default';
CREATE INDEX idx_tasks_user_id ON tasks(user_id); 