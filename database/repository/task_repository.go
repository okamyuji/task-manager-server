package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Task タスクモデル
type Task struct {
	ID          string
	UserID      string
	Title       string
	Description string
	CreatedAt   time.Time
	DueDate     *time.Time
	IsCompleted bool
	CompletedAt *time.Time
	Priority    string
	ImageUrl    *string // 添付画像URL
	Tags        []string
}

// TaskRepository タスクリポジトリインターフェース
type TaskRepository interface {
	Create(ctx context.Context, task *Task) error
	GetByID(ctx context.Context, id string) (*Task, error)
	GetByUserID(ctx context.Context, userID string) ([]*Task, error)
	Update(ctx context.Context, task *Task) error
	Delete(ctx context.Context, id string) error
	Complete(ctx context.Context, id string) error
	Incomplete(ctx context.Context, id string) error
}

type taskRepository struct {
	db *sql.DB
}

// NewTaskRepository タスクリポジトリを作成
func NewTaskRepository(db *sql.DB) TaskRepository {
	return &taskRepository{db: db}
}

// Create 新規タスクを作成
func (r *taskRepository) Create(ctx context.Context, task *Task) error {
	if task.ID == "" {
		task.ID = uuid.New().String()
	}
	task.CreatedAt = time.Now()

	// トランザクション開始
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// タスク挿入
	query := `
		INSERT INTO tasks (id, user_id, title, description, created_at, due_date, is_completed, completed_at, priority, image_url)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err = tx.ExecContext(ctx, query, task.ID, task.UserID, task.Title, task.Description, task.CreatedAt, task.DueDate, task.IsCompleted, task.CompletedAt, task.Priority, task.ImageUrl)
	if err != nil {
		return fmt.Errorf("タスク作成失敗: %w", err)
	}

	// タグ挿入
	if len(task.Tags) > 0 {
		tagQuery := `INSERT INTO task_tags (task_id, tag) VALUES ($1, $2)`
		for _, tag := range task.Tags {
			if _, err := tx.ExecContext(ctx, tagQuery, task.ID, tag); err != nil {
				return fmt.Errorf("タグ作成失敗: %w", err)
			}
		}
	}

	return tx.Commit()
}

// GetByID IDでタスクを取得
func (r *taskRepository) GetByID(ctx context.Context, id string) (*Task, error) {
	task := &Task{}
	query := `
		SELECT id, user_id, title, description, created_at, due_date, is_completed, completed_at, priority, image_url
		FROM tasks
		WHERE id = $1
	`
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&task.ID,
		&task.UserID,
		&task.Title,
		&task.Description,
		&task.CreatedAt,
		&task.DueDate,
		&task.IsCompleted,
		&task.CompletedAt,
		&task.Priority,
		&task.ImageUrl,
	)
	if err == sql.ErrNoRows {
		return nil, nil // タスクが見つからない場合はnilを返す
	}
	if err != nil {
		return nil, fmt.Errorf("タスク取得失敗: %w", err)
	}

	// タグ取得
	tags, err := r.getTaskTags(ctx, id)
	if err != nil {
		return nil, err
	}
	task.Tags = tags

	return task, nil
}

// GetByUserID ユーザーIDでタスク一覧を取得
func (r *taskRepository) GetByUserID(ctx context.Context, userID string) ([]*Task, error) {
	query := `
		SELECT id, user_id, title, description, created_at, due_date, is_completed, completed_at, priority, image_url
		FROM tasks
		WHERE user_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("タスク一覧取得失敗: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var tasks []*Task
	for rows.Next() {
		task := &Task{}
		if err := rows.Scan(
			&task.ID,
			&task.UserID,
			&task.Title,
			&task.Description,
			&task.CreatedAt,
			&task.DueDate,
			&task.IsCompleted,
			&task.CompletedAt,
			&task.Priority,
			&task.ImageUrl,
		); err != nil {
			return nil, err
		}

		// タグ取得
		tags, err := r.getTaskTags(ctx, task.ID)
		if err != nil {
			return nil, err
		}
		task.Tags = tags

		tasks = append(tasks, task)
	}

	return tasks, rows.Err()
}

// Update タスクを更新
func (r *taskRepository) Update(ctx context.Context, task *Task) error {
	// トランザクション開始
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// タスク更新
	query := `
		UPDATE tasks
		SET title = $1, description = $2, due_date = $3, is_completed = $4, completed_at = $5, priority = $6, image_url = $7
		WHERE id = $8
	`
	result, err := tx.ExecContext(ctx, query, task.Title, task.Description, task.DueDate, task.IsCompleted, task.CompletedAt, task.Priority, task.ImageUrl, task.ID)
	if err != nil {
		return fmt.Errorf("タスク更新失敗: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("タスクが見つかりません: %s", task.ID)
	}

	// 既存タグ削除
	if _, err := tx.ExecContext(ctx, "DELETE FROM task_tags WHERE task_id = $1", task.ID); err != nil {
		return fmt.Errorf("タグ削除失敗: %w", err)
	}

	// 新しいタグ挿入
	if len(task.Tags) > 0 {
		tagQuery := `INSERT INTO task_tags (task_id, tag) VALUES ($1, $2)`
		for _, tag := range task.Tags {
			if _, err := tx.ExecContext(ctx, tagQuery, task.ID, tag); err != nil {
				return fmt.Errorf("タグ作成失敗: %w", err)
			}
		}
	}

	return tx.Commit()
}

// Delete タスクを削除
func (r *taskRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM tasks WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// Complete タスクを完了にする
func (r *taskRepository) Complete(ctx context.Context, id string) error {
	query := `UPDATE tasks SET is_completed = TRUE, completed_at = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, time.Now(), id)
	return err
}

// Incomplete タスクを未完了にする
func (r *taskRepository) Incomplete(ctx context.Context, id string) error {
	query := `UPDATE tasks SET is_completed = FALSE, completed_at = NULL WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// getTaskTags タスクのタグを取得
func (r *taskRepository) getTaskTags(ctx context.Context, taskID string) ([]string, error) {
	query := `SELECT tag FROM task_tags WHERE task_id = $1 ORDER BY tag`
	rows, err := r.db.QueryContext(ctx, query, taskID)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	var tags []string
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}

	return tags, rows.Err()
}
