package database

import (
	"database/sql"
	"embed"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	_ "github.com/lib/pq" // PostgreSQLドライバー
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// DB データベース接続ラッパー
type DB struct {
	*sql.DB
	logger *slog.Logger
}

// NewDB 新しいデータベース接続を作成
func NewDB(dataSourceName string, logger *slog.Logger) (*DB, error) {
	// PostgreSQL接続
	db, err := sql.Open("postgres", dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("データベース接続失敗: %w", err)
	}

	// 接続プール設定
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)

	// 接続確認
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("データベースping失敗: %w", err)
	}

	logger.Info("データベース接続成功（PostgreSQL）")

	return &DB{DB: db, logger: logger}, nil
}

// Migrate マイグレーションを実行
func (db *DB) Migrate() error {
	db.logger.Info("マイグレーション開始...")

	// マイグレーションテーブルの作成
	if err := db.createMigrationTable(); err != nil {
		return fmt.Errorf("マイグレーションテーブル作成失敗: %w", err)
	}

	// マイグレーションファイルを読み込み
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("マイグレーションディレクトリ読み込み失敗: %w", err)
	}

	// SQLファイルのみを抽出してソート
	var migrations []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			migrations = append(migrations, entry.Name())
		}
	}
	sort.Strings(migrations)

	db.logger.Info("マイグレーションファイル検出", "count", len(migrations))

	// 各マイグレーションを実行
	for _, filename := range migrations {
		if err := db.runMigration(filename); err != nil {
			return fmt.Errorf("マイグレーション %s 失敗: %w", filename, err)
		}
	}

	db.logger.Info("マイグレーション完了")
	return nil
}

// createMigrationTable マイグレーション履歴テーブルを作成
func (db *DB) createMigrationTable() error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`
	_, err := db.Exec(query)
	return err
}

// runMigration 個別のマイグレーションを実行
func (db *DB) runMigration(filename string) error {
	// 既に実行済みかチェック（PostgreSQLは$1形式）
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = $1", filename).Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		db.logger.Debug("マイグレーションスキップ（実行済み）", "file", filename)
		return nil
	}

	db.logger.Info("マイグレーション実行中", "file", filename)

	// SQLファイルを読み込み
	content, err := migrationsFS.ReadFile("migrations/" + filename)
	if err != nil {
		return err
	}

	// トランザクション開始
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// SQL実行（複数ステートメント対応）
	if _, err := tx.Exec(string(content)); err != nil {
		return fmt.Errorf("SQL実行エラー: %w", err)
	}

	// マイグレーション履歴に記録（PostgreSQLは$1形式）
	if _, err := tx.Exec("INSERT INTO schema_migrations (version) VALUES ($1)", filename); err != nil {
		return err
	}

	// コミット
	if err := tx.Commit(); err != nil {
		return err
	}

	db.logger.Info("マイグレーション成功", "file", filename)
	return nil
}

// Close データベース接続を閉じる
func (db *DB) Close() error {
	db.logger.Info("データベース接続をクローズします")
	return db.DB.Close()
}

// BeginTx トランザクションを開始（オプション指定可能）
func (db *DB) BeginTx() (*sql.Tx, error) {
	return db.Begin()
}
