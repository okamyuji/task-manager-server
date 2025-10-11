package repository

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	_ "github.com/lib/pq"
)

// setupTestDB PostgreSQL TestContainerをセットアップ
func setupTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()

	ctx := context.Background()

	// PostgreSQL 16.8コンテナを起動
	pgContainer, err := postgres.Run(ctx,
		"postgres:16.8-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2),
		),
	)
	if err != nil {
		t.Fatalf("Failed to start PostgreSQL container: %v", err)
	}

	// 接続文字列を取得
	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("Failed to get connection string: %v", err)
	}

	// データベース接続
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// 接続確認
	if err := db.Ping(); err != nil {
		t.Fatalf("Failed to ping database: %v", err)
	}

	// スキーマ作成
	if err := createSchema(db); err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	// クリーンアップ関数
	cleanup := func() {
		_ = db.Close()
		_ = pgContainer.Terminate(ctx)
	}

	return db, cleanup
}

// createSchema テスト用スキーマを作成
func createSchema(db *sql.DB) error {
	schema := `
		-- ユーザーテーブル
		CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			email TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			name TEXT NOT NULL,
			is_verified BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
		CREATE INDEX IF NOT EXISTS idx_users_verified ON users(is_verified);

		-- タスクテーブル
		CREATE TABLE IF NOT EXISTS tasks (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			title TEXT NOT NULL,
			description TEXT,
			created_at TIMESTAMP NOT NULL,
			due_date TIMESTAMP,
			is_completed BOOLEAN DEFAULT FALSE,
			completed_at TIMESTAMP,
			priority TEXT DEFAULT 'medium',
			image_url TEXT DEFAULT NULL,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		);

		CREATE INDEX IF NOT EXISTS idx_tasks_user_id ON tasks(user_id);
		CREATE INDEX IF NOT EXISTS idx_tasks_completed ON tasks(is_completed);
		CREATE INDEX IF NOT EXISTS idx_tasks_priority ON tasks(priority);

		-- タスクタグテーブル
		CREATE TABLE IF NOT EXISTS task_tags (
			task_id TEXT NOT NULL,
			tag TEXT NOT NULL,
			PRIMARY KEY (task_id, tag),
			FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
		);

		-- 認証コードテーブル
		CREATE TABLE IF NOT EXISTS verification_codes (
			id SERIAL PRIMARY KEY,
			user_id TEXT NOT NULL,
			code TEXT NOT NULL,
			expires_at TIMESTAMP NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			used BOOLEAN DEFAULT FALSE,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		);

		CREATE INDEX IF NOT EXISTS idx_verification_user_id ON verification_codes(user_id);
		CREATE INDEX IF NOT EXISTS idx_verification_code ON verification_codes(code);
		CREATE INDEX IF NOT EXISTS idx_verification_expires ON verification_codes(expires_at);
	`

	_, err := db.Exec(schema)
	return err
}

// cleanupDB テストデータをクリーンアップ
func cleanupDB(t *testing.T, db *sql.DB) {
	t.Helper()

	queries := []string{
		"DELETE FROM task_tags",
		"DELETE FROM tasks",
		"DELETE FROM verification_codes",
		"DELETE FROM users",
	}

	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			t.Logf("Failed to cleanup: %v", err)
		}
	}
}

// assertNoError エラーがないことをアサート
func assertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// assertError エラーがあることをアサート
func assertError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// assertEqual 値が等しいことをアサート
func assertEqual(t *testing.T, got, want interface{}) {
	t.Helper()
	if fmt.Sprintf("%v", got) != fmt.Sprintf("%v", want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
