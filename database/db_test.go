package database

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// setupTestPostgres テスト用PostgreSQLコンテナをセットアップ
func setupTestPostgres(t *testing.T) (*DB, func()) {
	t.Helper()

	ctx := context.Background()

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

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("Failed to get connection string: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	db, err := NewDB(connStr, logger)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	cleanup := func() {
		_ = db.Close()
		_ = pgContainer.Terminate(ctx)
	}

	return db, cleanup
}

// TestNewDB_正常系
func TestNewDB_正常系(t *testing.T) {
	db, cleanup := setupTestPostgres(t)
	defer cleanup()

	if db == nil {
		t.Fatal("NewDB() returned nil")
	}

	// Ping確認
	if err := db.Ping(); err != nil {
		t.Errorf("Ping() error = %v", err)
	}
}

// TestNewDB_異常系_無効な接続文字列
func TestNewDB_異常系_無効な接続文字列(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	_, err := NewDB("invalid-connection-string", logger)
	if err == nil {
		t.Error("NewDB() expected error for invalid connection string")
	}
}

// TestMigrate_正常系
func TestMigrate_正常系(t *testing.T) {
	db, cleanup := setupTestPostgres(t)
	defer cleanup()

	err := db.Migrate()
	if err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	// schema_migrationsテーブルが作成されていることを確認
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count)
	if err != nil {
		t.Errorf("Failed to query schema_migrations: %v", err)
	}

	if count == 0 {
		t.Error("Migrate() did not apply any migrations")
	}
}

// TestMigrate_冪等性
func TestMigrate_冪等性(t *testing.T) {
	db, cleanup := setupTestPostgres(t)
	defer cleanup()

	// 1回目のマイグレーション
	err := db.Migrate()
	if err != nil {
		t.Fatalf("Migrate() first run error = %v", err)
	}

	// 2回目のマイグレーション（重複実行）
	err = db.Migrate()
	if err != nil {
		t.Errorf("Migrate() second run error = %v", err)
	}
}

// TestBeginTx_正常系
func TestBeginTx_正常系(t *testing.T) {
	db, cleanup := setupTestPostgres(t)
	defer cleanup()

	tx, err := db.BeginTx()
	if err != nil {
		t.Fatalf("BeginTx() error = %v", err)
	}

	if tx == nil {
		t.Fatal("BeginTx() returned nil transaction")
	}

	// ロールバック
	err = tx.Rollback()
	if err != nil {
		t.Errorf("Rollback() error = %v", err)
	}
}

// TestClose_正常系
func TestClose_正常系(t *testing.T) {
	db, cleanup := setupTestPostgres(t)
	defer cleanup()

	err := db.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// 閉じた後のPingは失敗するはず
	err = db.Ping()
	if err == nil {
		t.Error("Ping() should fail after Close()")
	}
}
