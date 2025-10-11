package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"task_manager_server/database/repository"
	"task_manager_server/models"
	"task_manager_server/validation"
)

// TaskHandler タスク関連のハンドラー
type TaskHandler struct {
	taskRepo repository.TaskRepository
	logger   *slog.Logger
}

// NewTaskHandler タスクハンドラーを作成
func NewTaskHandler(taskRepo repository.TaskRepository, logger *slog.Logger) *TaskHandler {
	return &TaskHandler{
		taskRepo: taskRepo,
		logger:   logger,
	}
}

// HandleTasks タスク一覧の取得と作成
func (h *TaskHandler) HandleTasks(w http.ResponseWriter, r *http.Request) {
	// 認証情報を取得
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		h.logger.Warn("ユーザーIDが見つかりません")
		RespondWithError(w, http.StatusUnauthorized, "User ID not found")
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getTasks(w, userID)
	case http.MethodPost:
		h.createTask(w, r, userID)
	default:
		RespondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// HandleTaskByID タスクの取得、更新、削除
func (h *TaskHandler) HandleTaskByID(w http.ResponseWriter, r *http.Request) {
	// 認証情報を取得
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		h.logger.Warn("ユーザーIDが見つかりません")
		RespondWithError(w, http.StatusUnauthorized, "User ID not found")
		return
	}

	// URLからタスクIDを取得
	path := strings.TrimPrefix(r.URL.Path, "/tasks/")

	// パスが空の場合は不正なリクエスト
	if path == "" {
		h.logger.Warn("タスクIDが指定されていません", "path", r.URL.Path, "user_id", userID)
		RespondWithError(w, http.StatusBadRequest, "Task ID is required")
		return
	}

	// complete/incompleteのチェック
	if strings.HasSuffix(path, "/complete") {
		taskID := strings.TrimSuffix(path, "/complete")
		h.completeTask(w, r, userID, taskID)
		return
	}
	if strings.HasSuffix(path, "/incomplete") {
		taskID := strings.TrimSuffix(path, "/incomplete")
		h.incompleteTask(w, r, userID, taskID)
		return
	}

	taskID := path

	switch r.Method {
	case http.MethodGet:
		h.getTask(w, userID, taskID)
	case http.MethodPut:
		h.updateTask(w, r, userID, taskID)
	case http.MethodDelete:
		h.deleteTask(w, userID, taskID)
	default:
		RespondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// getTasks タスク一覧を取得（自分のタスクのみ）
func (h *TaskHandler) getTasks(w http.ResponseWriter, userID string) {
	tasks, err := h.taskRepo.GetByUserID(userID)
	if err != nil {
		h.logger.Error("タスク一覧取得エラー", "error", err, "user_id", userID)
		RespondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	// タスクをAPIモデルに変換
	apiTasks := make([]*models.Task, len(tasks))
	for i, task := range tasks {
		apiTasks[i] = repositoryTaskToAPITask(task)
	}

	h.logger.Info("タスク一覧取得", "user_id", userID, "count", len(apiTasks))
	RespondWithJSON(w, http.StatusOK, apiTasks)
}

// getTask タスク詳細を取得（自分のタスクのみ）
func (h *TaskHandler) getTask(w http.ResponseWriter, userID, taskID string) {
	task, err := h.taskRepo.GetByID(taskID)
	if err != nil {
		h.logger.Error("タスク取得エラー", "error", err, "task_id", taskID, "user_id", userID)
		RespondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	if task == nil {
		h.logger.Warn("タスクが見つかりません", "task_id", taskID, "user_id", userID)
		RespondWithError(w, http.StatusNotFound, "Task not found")
		return
	}

	// 所有者チェック
	if task.UserID != userID {
		h.logger.Warn("タスクへのアクセス権限がありません",
			"task_id", taskID,
			"task_owner", task.UserID,
			"requesting_user", userID,
		)
		RespondWithError(w, http.StatusForbidden, "Access denied")
		return
	}

	h.logger.Debug("タスク詳細取得", "task_id", taskID, "user_id", userID, "title", task.Title)
	RespondWithJSON(w, http.StatusOK, repositoryTaskToAPITask(task))
}

// createTask タスクを作成
func (h *TaskHandler) createTask(w http.ResponseWriter, r *http.Request, userID string) {
	var apiTask models.Task
	if err := json.NewDecoder(r.Body).Decode(&apiTask); err != nil {
		h.logger.Warn("タスク作成: リクエストボディのパースエラー", "error", err, "user_id", userID)
		RespondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// バリデーション
	if err := validation.ValidateTaskTitle(apiTask.Title); err != nil {
		h.logger.Warn("バリデーションエラー", "error", err)
		RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := validation.ValidateTaskDescription(apiTask.Description); err != nil {
		h.logger.Warn("バリデーションエラー", "error", err)
		RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := validation.ValidateTaskPriority(apiTask.Priority); err != nil {
		h.logger.Warn("バリデーションエラー", "error", err)
		RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	// リポジトリモデルに変換
	task := apiTaskToRepositoryTask(&apiTask)
	task.UserID = userID

	// 優先度のデフォルト値
	if task.Priority == "" {
		task.Priority = "medium"
	}

	// タグが未設定の場合は空配列
	if task.Tags == nil {
		task.Tags = []string{}
	}

	// タスク作成
	if err := h.taskRepo.Create(task); err != nil {
		h.logger.Error("タスク作成エラー", "error", err, "user_id", userID)
		RespondWithError(w, http.StatusInternalServerError, "Failed to create task")
		return
	}

	h.logger.Info("タスク作成完了",
		"task_id", task.ID,
		"user_id", userID,
		"title", task.Title,
		"priority", task.Priority,
	)

	RespondWithJSON(w, http.StatusCreated, repositoryTaskToAPITask(task))
}

// updateTask タスクを更新（自分のタスクのみ）
func (h *TaskHandler) updateTask(w http.ResponseWriter, r *http.Request, userID, taskID string) {
	// 既存タスクの確認
	existingTask, err := h.taskRepo.GetByID(taskID)
	if err != nil {
		h.logger.Error("タスク取得エラー", "error", err, "task_id", taskID, "user_id", userID)
		RespondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	if existingTask == nil {
		h.logger.Warn("タスク更新: タスクが見つかりません", "task_id", taskID, "user_id", userID)
		RespondWithError(w, http.StatusNotFound, "Task not found")
		return
	}

	// 所有者チェック
	if existingTask.UserID != userID {
		h.logger.Warn("タスク更新: アクセス権限がありません",
			"task_id", taskID,
			"task_owner", existingTask.UserID,
			"requesting_user", userID,
		)
		RespondWithError(w, http.StatusForbidden, "Access denied")
		return
	}

	var apiTask models.Task
	if err := json.NewDecoder(r.Body).Decode(&apiTask); err != nil {
		h.logger.Warn("タスク更新: リクエストボディのパースエラー", "error", err, "task_id", taskID, "user_id", userID)
		RespondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// バリデーション
	if err := validation.ValidateTaskTitle(apiTask.Title); err != nil {
		h.logger.Warn("バリデーションエラー", "error", err)
		RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := validation.ValidateTaskDescription(apiTask.Description); err != nil {
		h.logger.Warn("バリデーションエラー", "error", err)
		RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := validation.ValidateTaskPriority(apiTask.Priority); err != nil {
		h.logger.Warn("バリデーションエラー", "error", err)
		RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	// リポジトリモデルに変換
	task := apiTaskToRepositoryTask(&apiTask)
	task.ID = taskID
	task.UserID = userID
	task.CreatedAt = existingTask.CreatedAt // 作成日時は保持

	// タグが未設定の場合は空配列
	if task.Tags == nil {
		task.Tags = []string{}
	}

	// タスク更新
	if err := h.taskRepo.Update(task); err != nil {
		h.logger.Error("タスク更新エラー", "error", err, "task_id", taskID, "user_id", userID)
		RespondWithError(w, http.StatusInternalServerError, "Failed to update task")
		return
	}

	h.logger.Info("タスク更新完了", "task_id", taskID, "user_id", userID, "title", task.Title)
	RespondWithJSON(w, http.StatusOK, repositoryTaskToAPITask(task))
}

// deleteTask タスクを削除（自分のタスクのみ）
func (h *TaskHandler) deleteTask(w http.ResponseWriter, userID, taskID string) {
	// 既存タスクの確認
	task, err := h.taskRepo.GetByID(taskID)
	if err != nil {
		h.logger.Error("タスク取得エラー", "error", err, "task_id", taskID, "user_id", userID)
		RespondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	if task == nil {
		h.logger.Warn("タスク削除: タスクが見つかりません", "task_id", taskID, "user_id", userID)
		RespondWithError(w, http.StatusNotFound, "Task not found")
		return
	}

	// 所有者チェック
	if task.UserID != userID {
		h.logger.Warn("タスク削除: アクセス権限がありません",
			"task_id", taskID,
			"task_owner", task.UserID,
			"requesting_user", userID,
		)
		RespondWithError(w, http.StatusForbidden, "Access denied")
		return
	}

	// タスク削除
	if err := h.taskRepo.Delete(taskID); err != nil {
		h.logger.Error("タスク削除エラー", "error", err, "task_id", taskID, "user_id", userID)
		RespondWithError(w, http.StatusInternalServerError, "Failed to delete task")
		return
	}

	h.logger.Info("タスク削除完了", "task_id", taskID, "user_id", userID, "title", task.Title)
	w.WriteHeader(http.StatusNoContent)
}

// completeTask タスクを完了状態にする（自分のタスクのみ）
func (h *TaskHandler) completeTask(w http.ResponseWriter, r *http.Request, userID, taskID string) {
	if r.Method != http.MethodPatch {
		h.logger.Warn("不正なHTTPメソッド", "method", r.Method, "path", r.URL.Path)
		RespondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// 既存タスクの確認
	task, err := h.taskRepo.GetByID(taskID)
	if err != nil {
		h.logger.Error("タスク取得エラー", "error", err, "task_id", taskID, "user_id", userID)
		RespondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	if task == nil {
		h.logger.Warn("タスク完了: タスクが見つかりません", "task_id", taskID, "user_id", userID)
		RespondWithError(w, http.StatusNotFound, "Task not found")
		return
	}

	// 所有者チェック
	if task.UserID != userID {
		h.logger.Warn("タスク完了: アクセス権限がありません",
			"task_id", taskID,
			"task_owner", task.UserID,
			"requesting_user", userID,
		)
		RespondWithError(w, http.StatusForbidden, "Access denied")
		return
	}

	// タスク完了
	if err := h.taskRepo.Complete(taskID); err != nil {
		h.logger.Error("タスク完了エラー", "error", err, "task_id", taskID, "user_id", userID)
		RespondWithError(w, http.StatusInternalServerError, "Failed to complete task")
		return
	}

	// 更新後のタスクを取得
	updatedTask, _ := h.taskRepo.GetByID(taskID)

	h.logger.Info("タスク完了", "task_id", taskID, "user_id", userID, "title", task.Title)
	RespondWithJSON(w, http.StatusOK, repositoryTaskToAPITask(updatedTask))
}

// incompleteTask タスクを未完了状態にする（自分のタスクのみ）
func (h *TaskHandler) incompleteTask(w http.ResponseWriter, r *http.Request, userID, taskID string) {
	if r.Method != http.MethodPatch {
		h.logger.Warn("不正なHTTPメソッド", "method", r.Method, "path", r.URL.Path)
		RespondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// 既存タスクの確認
	task, err := h.taskRepo.GetByID(taskID)
	if err != nil {
		h.logger.Error("タスク取得エラー", "error", err, "task_id", taskID, "user_id", userID)
		RespondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	if task == nil {
		h.logger.Warn("タスク未完了化: タスクが見つかりません", "task_id", taskID, "user_id", userID)
		RespondWithError(w, http.StatusNotFound, "Task not found")
		return
	}

	// 所有者チェック
	if task.UserID != userID {
		h.logger.Warn("タスク未完了化: アクセス権限がありません",
			"task_id", taskID,
			"task_owner", task.UserID,
			"requesting_user", userID,
		)
		RespondWithError(w, http.StatusForbidden, "Access denied")
		return
	}

	// タスク未完了化
	if err := h.taskRepo.Incomplete(taskID); err != nil {
		h.logger.Error("タスク未完了化エラー", "error", err, "task_id", taskID, "user_id", userID)
		RespondWithError(w, http.StatusInternalServerError, "Failed to incomplete task")
		return
	}

	// 更新後のタスクを取得
	updatedTask, _ := h.taskRepo.GetByID(taskID)

	h.logger.Info("タスク未完了化", "task_id", taskID, "user_id", userID, "title", task.Title)
	RespondWithJSON(w, http.StatusOK, repositoryTaskToAPITask(updatedTask))
}

// repositoryTaskToAPITask リポジトリモデルをAPIモデルに変換
func repositoryTaskToAPITask(task *repository.Task) *models.Task {
	apiTask := &models.Task{
		ID:          task.ID,
		UserID:      task.UserID,
		Title:       task.Title,
		Description: task.Description,
		CreatedAt:   models.FlexibleTime{Time: task.CreatedAt},
		IsCompleted: task.IsCompleted,
		Tags:        task.Tags,
		Priority:    task.Priority,
		ImageUrl:    task.ImageUrl,
	}

	if task.DueDate != nil {
		apiTask.DueDate = &models.FlexibleTime{Time: *task.DueDate}
	}
	if task.CompletedAt != nil {
		apiTask.CompletedAt = &models.FlexibleTime{Time: *task.CompletedAt}
	}

	return apiTask
}

// apiTaskToRepositoryTask APIモデルをリポジトリモデルに変換
func apiTaskToRepositoryTask(apiTask *models.Task) *repository.Task {
	task := &repository.Task{
		ID:          apiTask.ID,
		UserID:      apiTask.UserID,
		Title:       apiTask.Title,
		Description: apiTask.Description,
		CreatedAt:   apiTask.CreatedAt.Time,
		IsCompleted: apiTask.IsCompleted,
		Tags:        apiTask.Tags,
		Priority:    apiTask.Priority,
		ImageUrl:    apiTask.ImageUrl,
	}

	if apiTask.DueDate != nil {
		dueDate := apiTask.DueDate.Time
		task.DueDate = &dueDate
	}
	if apiTask.CompletedAt != nil {
		completedAt := apiTask.CompletedAt.Time
		task.CompletedAt = &completedAt
	}

	return task
}
