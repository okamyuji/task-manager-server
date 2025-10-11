package handlers

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"task_manager_server/database/repository"
	"task_manager_server/models"
)

// mockTaskRepository
type mockTaskRepository struct {
	tasks map[string]*repository.Task
}

func newMockTaskRepository() *mockTaskRepository {
	return &mockTaskRepository{
		tasks: make(map[string]*repository.Task),
	}
}

func (m *mockTaskRepository) Create(task *repository.Task) error {
	if task.ID == "" {
		task.ID = "mock-task-id"
	}
	task.CreatedAt = time.Now()
	m.tasks[task.ID] = task
	return nil
}

func (m *mockTaskRepository) GetByID(id string) (*repository.Task, error) {
	if t, ok := m.tasks[id]; ok {
		return t, nil
	}
	return nil, nil
}

func (m *mockTaskRepository) GetByUserID(userID string) ([]*repository.Task, error) {
	var result []*repository.Task
	for _, t := range m.tasks {
		if t.UserID == userID {
			result = append(result, t)
		}
	}
	return result, nil
}

func (m *mockTaskRepository) Update(task *repository.Task) error {
	m.tasks[task.ID] = task
	return nil
}

func (m *mockTaskRepository) Delete(id string) error {
	delete(m.tasks, id)
	return nil
}

func (m *mockTaskRepository) Complete(id string) error {
	if t, ok := m.tasks[id]; ok {
		t.IsCompleted = true
		now := time.Now()
		t.CompletedAt = &now
		return nil
	}
	return nil
}

func (m *mockTaskRepository) Incomplete(id string) error {
	if t, ok := m.tasks[id]; ok {
		t.IsCompleted = false
		t.CompletedAt = nil
		return nil
	}
	return nil
}

// TestTaskHandler_HandleTasks_GET_正常系
func TestTaskHandler_HandleTasks_GET_正常系(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	taskRepo := newMockTaskRepository()
	handler := NewTaskHandler(taskRepo, logger)

	// テストデータ作成
	task := &repository.Task{
		ID:          "task1",
		UserID:      "user1",
		Title:       "Test Task",
		Description: "Description",
		Priority:    "high",
		Tags:        []string{"test"},
	}
	_ = taskRepo.Create(task)

	req := httptest.NewRequest(http.MethodGet, "/tasks", nil)
	req.Header.Set("X-User-ID", "user1")
	rec := httptest.NewRecorder()

	handler.HandleTasks(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
}

// TestTaskHandler_HandleTasks_POST_正常系
func TestTaskHandler_HandleTasks_POST_正常系(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	taskRepo := newMockTaskRepository()
	handler := NewTaskHandler(taskRepo, logger)

	reqBody := models.Task{
		Title:       "New Task",
		Description: "New Description",
		Priority:    "medium",
		Tags:        []string{"new"},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewReader(body))
	req.Header.Set("X-User-ID", "user1")
	rec := httptest.NewRecorder()

	handler.HandleTasks(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", rec.Code)
	}
}

// TestTaskHandler_HandleTaskByID_GET_正常系
func TestTaskHandler_HandleTaskByID_GET_正常系(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	taskRepo := newMockTaskRepository()
	handler := NewTaskHandler(taskRepo, logger)

	task := &repository.Task{
		ID:     "task1",
		UserID: "user1",
		Title:  "Test Task",
	}
	_ = taskRepo.Create(task)

	req := httptest.NewRequest(http.MethodGet, "/tasks/task1", nil)
	req.Header.Set("X-User-ID", "user1")
	rec := httptest.NewRecorder()

	handler.HandleTaskByID(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
}

// TestTaskHandler_HandleTaskByID_GET_異常系_権限なし
func TestTaskHandler_HandleTaskByID_GET_異常系_権限なし(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	taskRepo := newMockTaskRepository()
	handler := NewTaskHandler(taskRepo, logger)

	task := &repository.Task{
		ID:     "task1",
		UserID: "user1",
		Title:  "Test Task",
	}
	_ = taskRepo.Create(task)

	req := httptest.NewRequest(http.MethodGet, "/tasks/task1", nil)
	req.Header.Set("X-User-ID", "user2")
	rec := httptest.NewRecorder()

	handler.HandleTaskByID(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", rec.Code)
	}
}

// TestTaskHandler_HandleTaskByID_PUT_正常系
func TestTaskHandler_HandleTaskByID_PUT_正常系(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	taskRepo := newMockTaskRepository()
	handler := NewTaskHandler(taskRepo, logger)

	task := &repository.Task{
		ID:     "task1",
		UserID: "user1",
		Title:  "Old Title",
	}
	_ = taskRepo.Create(task)

	reqBody := models.Task{
		Title:    "Updated Title",
		Priority: "high",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/tasks/task1", bytes.NewReader(body))
	req.Header.Set("X-User-ID", "user1")
	rec := httptest.NewRecorder()

	handler.HandleTaskByID(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
}

// TestTaskHandler_HandleTaskByID_DELETE_正常系
func TestTaskHandler_HandleTaskByID_DELETE_正常系(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	taskRepo := newMockTaskRepository()
	handler := NewTaskHandler(taskRepo, logger)

	task := &repository.Task{
		ID:     "task1",
		UserID: "user1",
		Title:  "Delete Task",
	}
	_ = taskRepo.Create(task)

	req := httptest.NewRequest(http.MethodDelete, "/tasks/task1", nil)
	req.Header.Set("X-User-ID", "user1")
	rec := httptest.NewRecorder()

	handler.HandleTaskByID(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", rec.Code)
	}
}

// TestTaskHandler_CompleteTask_正常系
func TestTaskHandler_CompleteTask_正常系(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	taskRepo := newMockTaskRepository()
	handler := NewTaskHandler(taskRepo, logger)

	task := &repository.Task{
		ID:     "task1",
		UserID: "user1",
		Title:  "Complete Task",
	}
	_ = taskRepo.Create(task)

	req := httptest.NewRequest(http.MethodPatch, "/tasks/task1/complete", nil)
	req.Header.Set("X-User-ID", "user1")
	rec := httptest.NewRecorder()

	handler.HandleTaskByID(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
}

// TestTaskHandler_IncompleteTask_正常系
func TestTaskHandler_IncompleteTask_正常系(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	taskRepo := newMockTaskRepository()
	handler := NewTaskHandler(taskRepo, logger)

	task := &repository.Task{
		ID:          "task1",
		UserID:      "user1",
		Title:       "Incomplete Task",
		IsCompleted: true,
	}
	_ = taskRepo.Create(task)

	req := httptest.NewRequest(http.MethodPatch, "/tasks/task1/incomplete", nil)
	req.Header.Set("X-User-ID", "user1")
	rec := httptest.NewRecorder()

	handler.HandleTaskByID(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
}

// TestTaskHandler_HandleTasks_異常系_ユーザーIDなし
func TestTaskHandler_HandleTasks_異常系_ユーザーIDなし(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	taskRepo := newMockTaskRepository()
	handler := NewTaskHandler(taskRepo, logger)

	req := httptest.NewRequest(http.MethodGet, "/tasks", nil)
	rec := httptest.NewRecorder()

	handler.HandleTasks(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rec.Code)
	}
}
