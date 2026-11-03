package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"task_manager_server/database"
	"task_manager_server/database/repository"
	"task_manager_server/middleware"
	"task_manager_server/models"
	"task_manager_server/utils"
)

// setupTestDB PostgreSQLのコンテナを起動し、本番と同じマイグレーションを流す
func setupTestDB(t *testing.T) (*database.DB, func()) {
	t.Helper()
	ctx := context.Background()

	// ユーザー名とパスワードは指定せず、モジュールの既定値で起動する
	pgContainer, err := postgres.Run(ctx,
		"postgres:16.8-alpine",
		postgres.WithDatabase("testdb"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2),
		),
	)
	if err != nil {
		t.Fatalf("コンテナ起動失敗: %v", err)
	}

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("接続文字列の取得失敗: %v", err)
	}

	db, err := database.NewDB(connStr, slog.New(slog.NewTextHandler(os.Stdout, nil)))
	if err != nil {
		t.Fatalf("データベース接続失敗: %v", err)
	}
	if err := db.Migrate(); err != nil {
		t.Fatalf("マイグレーション失敗: %v", err)
	}

	cleanup := func() {
		_ = db.Close()
		_ = pgContainer.Terminate(ctx)
	}
	return db, cleanup
}

func TestTaskAPIIntegration(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// tasks.user_id は users を参照するので、先にユーザーを作る
	hash, err := utils.HashPassword(t.Name())
	if err != nil {
		t.Fatalf("パスワードのハッシュ化失敗: %v", err)
	}
	user := &repository.User{Email: "integration@example.com", PasswordHash: hash, Name: "Integration User"}
	if err := repository.NewUserRepository(db.DB).Create(user); err != nil {
		t.Fatalf("ユーザー作成失敗: %v", err)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	handler := NewTaskHandler(repository.NewTaskRepository(db.DB), logger)

	t.Run("タスク作成と取得", func(t *testing.T) {
		createReq := httptest.NewRequest(http.MethodPost, "/tasks",
			strings.NewReader(`{"title":"統合テスト","tags":["go"]}`))
		createReq.Header.Set("Content-Type", "application/json")
		ctx := middleware.WithUserID(createReq.Context(), user.ID)
		createReq = createReq.WithContext(ctx)

		createW := httptest.NewRecorder()
		handler.HandleTasks(createW, createReq)

		if createW.Code != http.StatusCreated {
			t.Fatalf("タスク作成失敗: status=%d body=%s", createW.Code, createW.Body.String())
		}

		listReq := httptest.NewRequest(http.MethodGet, "/tasks", nil).WithContext(ctx)
		listW := httptest.NewRecorder()
		handler.HandleTasks(listW, listReq)

		if listW.Code != http.StatusOK {
			t.Fatalf("タスク取得失敗: status=%d", listW.Code)
		}

		var tasks []*models.Task
		if err := json.NewDecoder(listW.Body).Decode(&tasks); err != nil {
			t.Fatalf("レスポンスのパース失敗: %v", err)
		}
		if len(tasks) != 1 {
			t.Fatalf("expected 1 task, got %d", len(tasks))
		}
		if tasks[0].Title != "統合テスト" {
			t.Errorf("expected title '統合テスト', got '%s'", tasks[0].Title)
		}
		if len(tasks[0].Tags) != 1 || tasks[0].Tags[0] != "go" {
			t.Errorf("expected tags [go], got %v", tasks[0].Tags)
		}
	})
}
