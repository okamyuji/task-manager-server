package repository

import (
	"context"
	"errors"
	"testing"
	"time"
)

// TestTaskRepository_Create_正常系
func TestTaskRepository_Create_正常系(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	defer cleanupDB(t, db)

	userRepo := NewUserRepository(db)
	taskRepo := NewTaskRepository(db)

	// ユーザー作成
	user := &User{
		Email:        "task@example.com",
		PasswordHash: "password",
		Name:         "Task User",
	}
	assertNoError(t, userRepo.Create(user))

	// タスク作成
	dueDate := time.Now().Add(24 * time.Hour)
	task := &Task{
		UserID:      user.ID,
		Title:       "Test Task",
		Description: "Test Description",
		DueDate:     &dueDate,
		Priority:    "high",
		Tags:        []string{"test", "important"},
	}

	err := taskRepo.Create(context.Background(), task)
	assertNoError(t, err)

	if task.ID == "" {
		t.Error("Create() task.ID が空です")
	}
	if task.CreatedAt.IsZero() {
		t.Error("Create() task.CreatedAt が設定されていません")
	}
}

// TestTaskRepository_Create_異常系_存在しないユーザー
func TestTaskRepository_Create_異常系_存在しないユーザー(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	defer cleanupDB(t, db)

	taskRepo := NewTaskRepository(db)

	task := &Task{
		UserID: "nonexistent-user",
		Title:  "Test Task",
	}

	err := taskRepo.Create(context.Background(), task)
	assertError(t, err)
}

// TestTaskRepository_Create_異常系_タグ挿入失敗でタスクも残らない
func TestTaskRepository_Create_異常系_タグ挿入失敗でタスクも残らない(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	defer cleanupDB(t, db)

	userRepo := NewUserRepository(db)
	taskRepo := NewTaskRepository(db)
	user := &User{Email: "rollback@example.com", PasswordHash: "password", Name: "Rollback User"}
	assertNoError(t, userRepo.Create(user))

	// 同じタグを2回入れて主キー違反を起こす
	task := &Task{UserID: user.ID, Title: "Rollback Task", Tags: []string{"dup", "dup"}}
	assertError(t, taskRepo.Create(context.Background(), task))

	var count int
	assertNoError(t, db.QueryRow("SELECT COUNT(*) FROM tasks WHERE id = $1", task.ID).Scan(&count))
	assertEqual(t, count, 0)
}

// TestTaskRepository_Create_異常系_キャンセル済みcontext
func TestTaskRepository_Create_異常系_キャンセル済みcontext(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	defer cleanupDB(t, db)

	userRepo := NewUserRepository(db)
	taskRepo := NewTaskRepository(db)
	user := &User{Email: "canceled@example.com", PasswordHash: "password", Name: "Canceled User"}
	assertNoError(t, userRepo.Create(user))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := taskRepo.Create(ctx, &Task{UserID: user.ID, Title: "Canceled Task"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Create() error = %v, want context.Canceled", err)
	}

	tasks, err := taskRepo.GetByUserID(context.Background(), user.ID)
	assertNoError(t, err)
	assertEqual(t, len(tasks), 0)
}

// TestTaskRepository_GetByID_異常系_キャンセル済みcontext
func TestTaskRepository_GetByID_異常系_キャンセル済みcontext(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	defer cleanupDB(t, db)

	taskRepo := NewTaskRepository(db)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := taskRepo.GetByID(ctx, "any-id")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("GetByID() error = %v, want context.Canceled", err)
	}
}

// TestTaskRepository_GetByID_正常系
func TestTaskRepository_GetByID_正常系(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	defer cleanupDB(t, db)

	userRepo := NewUserRepository(db)
	taskRepo := NewTaskRepository(db)

	user := &User{
		Email:        "getbyid@example.com",
		PasswordHash: "password",
		Name:         "User",
	}
	assertNoError(t, userRepo.Create(user))

	task := &Task{
		UserID: user.ID,
		Title:  "Get By ID Task",
		Tags:   []string{"test"},
	}
	assertNoError(t, taskRepo.Create(context.Background(), task))

	retrieved, err := taskRepo.GetByID(context.Background(), task.ID)
	assertNoError(t, err)

	assertEqual(t, retrieved.ID, task.ID)
	assertEqual(t, retrieved.Title, task.Title)
	if len(retrieved.Tags) != 1 || retrieved.Tags[0] != "test" {
		t.Errorf("Tags が一致しません: got %v, want [test]", retrieved.Tags)
	}
}

// TestTaskRepository_GetByID_異常系_存在しないID
func TestTaskRepository_GetByID_異常系_存在しないID(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	defer cleanupDB(t, db)

	taskRepo := NewTaskRepository(db)

	task, err := taskRepo.GetByID(context.Background(), "nonexistent-id")
	assertNoError(t, err)
	if task != nil {
		t.Errorf("expected nil task, got %v", task)
	}
}

// TestTaskRepository_GetByUserID_正常系
func TestTaskRepository_GetByUserID_正常系(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	defer cleanupDB(t, db)

	userRepo := NewUserRepository(db)
	taskRepo := NewTaskRepository(db)

	user := &User{
		Email:        "getuserlist@example.com",
		PasswordHash: "password",
		Name:         "User",
	}
	assertNoError(t, userRepo.Create(user))

	for i := 0; i < 3; i++ {
		task := &Task{
			UserID: user.ID,
			Title:  "Task " + string(rune('A'+i)),
		}
		assertNoError(t, taskRepo.Create(context.Background(), task))
		time.Sleep(time.Millisecond)
	}

	tasks, err := taskRepo.GetByUserID(context.Background(), user.ID)
	assertNoError(t, err)

	if len(tasks) != 3 {
		t.Errorf("GetByUserID() タスク数が不正: got %d, want 3", len(tasks))
	}
}

// TestTaskRepository_Update_正常系
func TestTaskRepository_Update_正常系(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	defer cleanupDB(t, db)

	userRepo := NewUserRepository(db)
	taskRepo := NewTaskRepository(db)

	user := &User{
		Email:        "update@example.com",
		PasswordHash: "password",
		Name:         "User",
	}
	assertNoError(t, userRepo.Create(user))

	task := &Task{
		UserID: user.ID,
		Title:  "Old Title",
		Tags:   []string{"old"},
	}
	assertNoError(t, taskRepo.Create(context.Background(), task))

	task.Title = "New Title"
	task.Description = "Updated Description"
	task.Tags = []string{"new", "updated"}

	err := taskRepo.Update(context.Background(), task)
	assertNoError(t, err)

	retrieved, err := taskRepo.GetByID(context.Background(), task.ID)
	assertNoError(t, err)
	assertEqual(t, retrieved.Title, "New Title")
	assertEqual(t, retrieved.Description, "Updated Description")
	if len(retrieved.Tags) != 2 {
		t.Errorf("Tags の数が一致しません: got %d, want 2", len(retrieved.Tags))
	}
}

// TestTaskRepository_Delete_正常系
func TestTaskRepository_Delete_正常系(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	defer cleanupDB(t, db)

	userRepo := NewUserRepository(db)
	taskRepo := NewTaskRepository(db)

	user := &User{
		Email:        "delete@example.com",
		PasswordHash: "password",
		Name:         "User",
	}
	assertNoError(t, userRepo.Create(user))

	task := &Task{
		UserID: user.ID,
		Title:  "Delete Task",
	}
	assertNoError(t, taskRepo.Create(context.Background(), task))

	err := taskRepo.Delete(context.Background(), task.ID)
	assertNoError(t, err)

	deletedTask, err := taskRepo.GetByID(context.Background(), task.ID)
	assertNoError(t, err)
	if deletedTask != nil {
		t.Errorf("expected nil (task should be deleted), got %v", deletedTask)
	}
}

// TestTaskRepository_Complete_正常系
func TestTaskRepository_Complete_正常系(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	defer cleanupDB(t, db)

	userRepo := NewUserRepository(db)
	taskRepo := NewTaskRepository(db)

	user := &User{
		Email:        "complete@example.com",
		PasswordHash: "password",
		Name:         "User",
	}
	assertNoError(t, userRepo.Create(user))

	task := &Task{
		UserID: user.ID,
		Title:  "Complete Task",
	}
	assertNoError(t, taskRepo.Create(context.Background(), task))

	err := taskRepo.Complete(context.Background(), task.ID)
	assertNoError(t, err)

	retrieved, err := taskRepo.GetByID(context.Background(), task.ID)
	assertNoError(t, err)
	if !retrieved.IsCompleted {
		t.Error("Complete() タスクが完了していません")
	}
	if retrieved.CompletedAt == nil {
		t.Error("Complete() CompletedAt が設定されていません")
	}
}

// TestTaskRepository_incomplete_正常系
func TestTaskRepository_incomplete_正常系(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	defer cleanupDB(t, db)

	userRepo := NewUserRepository(db)
	taskRepo := NewTaskRepository(db)

	user := &User{
		Email:        "uncomplete@example.com",
		PasswordHash: "password",
		Name:         "User",
	}
	assertNoError(t, userRepo.Create(user))

	task := &Task{
		UserID: user.ID,
		Title:  "incomplete Task",
	}
	assertNoError(t, taskRepo.Create(context.Background(), task))
	assertNoError(t, taskRepo.Complete(context.Background(), task.ID))

	err := taskRepo.Incomplete(context.Background(), task.ID)
	assertNoError(t, err)

	retrieved, err := taskRepo.GetByID(context.Background(), task.ID)
	assertNoError(t, err)
	if retrieved.IsCompleted {
		t.Error("incomplete() タスクが未完了になっていません")
	}
	if retrieved.CompletedAt != nil {
		t.Error("incomplete() CompletedAt がクリアされていません")
	}
}

// TestTaskRepository_GetByUserID_複数タスク
func TestTaskRepository_GetByUserID_複数タスク(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	defer cleanupDB(t, db)

	userRepo := NewUserRepository(db)
	taskRepo := NewTaskRepository(db)

	user := &User{
		Email:        "filter@example.com",
		PasswordHash: "password",
		Name:         "User",
	}
	assertNoError(t, userRepo.Create(user))

	for i := 0; i < 3; i++ {
		task := &Task{
			UserID: user.ID,
			Title:  "Task " + string(rune('A'+i)),
		}
		assertNoError(t, taskRepo.Create(context.Background(), task))
		if i%2 == 0 {
			assertNoError(t, taskRepo.Complete(context.Background(), task.ID))
		}
		time.Sleep(time.Millisecond)
	}

	allTasks, err := taskRepo.GetByUserID(context.Background(), user.ID)
	assertNoError(t, err)

	if len(allTasks) != 3 {
		t.Errorf("タスク数が不正: got %d, want 3", len(allTasks))
	}
}
