package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// User ユーザーモデル
type User struct {
	ID           string
	Email        string
	PasswordHash string
	Name         string
	IsVerified   bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// UserRepository ユーザーリポジトリインターフェース
type UserRepository interface {
	Create(user *User) error
	GetByID(id string) (*User, error)
	GetByEmail(email string) (*User, error)
	Update(user *User) error
	Delete(id string) error
	VerifyUser(id string) error
	List() ([]*User, error)
}

type userRepository struct {
	db *sql.DB
}

// NewUserRepository ユーザーリポジトリを作成
func NewUserRepository(db *sql.DB) UserRepository {
	return &userRepository{db: db}
}

// Create 新規ユーザーを作成
func (r *userRepository) Create(user *User) error {
	if user.ID == "" {
		user.ID = uuid.New().String()
	}
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	query := `
		INSERT INTO users (id, email, password_hash, name, is_verified, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.Exec(query, user.ID, user.Email, user.PasswordHash, user.Name, user.IsVerified, user.CreatedAt, user.UpdatedAt)
	if err != nil {
		return fmt.Errorf("ユーザー作成失敗: %w", err)
	}
	return nil
}

// GetByID IDでユーザーを取得
func (r *userRepository) GetByID(id string) (*User, error) {
	user := &User{}
	query := `
		SELECT id, email, password_hash, name, is_verified, created_at, updated_at
		FROM users
		WHERE id = $1
	`
	err := r.db.QueryRow(query, id).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Name,
		&user.IsVerified,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil // ユーザーが存在しない場合はnilを返す
	}
	if err != nil {
		return nil, fmt.Errorf("ユーザー取得失敗: %w", err)
	}
	return user, nil
}

// GetByEmail メールアドレスでユーザーを取得
func (r *userRepository) GetByEmail(email string) (*User, error) {
	user := &User{}
	query := `
		SELECT id, email, password_hash, name, is_verified, created_at, updated_at
		FROM users
		WHERE email = $1
	`
	err := r.db.QueryRow(query, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Name,
		&user.IsVerified,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil // ユーザーが存在しない場合はnilを返す
	}
	if err != nil {
		return nil, fmt.Errorf("ユーザー取得失敗: %w", err)
	}
	return user, nil
}

// Update ユーザー情報を更新
func (r *userRepository) Update(user *User) error {
	user.UpdatedAt = time.Now()
	query := `
		UPDATE users
		SET email = $1, password_hash = $2, name = $3, is_verified = $4, updated_at = $5
		WHERE id = $6
	`
	result, err := r.db.Exec(query, user.Email, user.PasswordHash, user.Name, user.IsVerified, user.UpdatedAt, user.ID)
	if err != nil {
		return fmt.Errorf("ユーザー更新失敗: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("ユーザーが見つかりません: %s", user.ID)
	}

	return nil
}

// Delete ユーザーを削除
func (r *userRepository) Delete(id string) error {
	query := `DELETE FROM users WHERE id = $1`
	_, err := r.db.Exec(query, id)
	return err
}

// VerifyUser ユーザーを認証済みにする
func (r *userRepository) VerifyUser(id string) error {
	query := `UPDATE users SET is_verified = TRUE, updated_at = $1 WHERE id = $2`
	_, err := r.db.Exec(query, time.Now(), id)
	return err
}

// List 全ユーザーを取得
func (r *userRepository) List() ([]*User, error) {
	query := `
		SELECT id, email, password_hash, name, is_verified, created_at, updated_at
		FROM users
		ORDER BY created_at DESC
	`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("ユーザー一覧取得失敗: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var users []*User
	for rows.Next() {
		user := &User{}
		if err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.PasswordHash,
			&user.Name,
			&user.IsVerified,
			&user.CreatedAt,
			&user.UpdatedAt,
		); err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, rows.Err()
}
