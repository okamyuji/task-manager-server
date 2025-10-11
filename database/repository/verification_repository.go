package repository

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"math/big"
	"time"
)

// VerificationCode 認証コードモデル
type VerificationCode struct {
	ID        int64
	UserID    string
	Code      string
	ExpiresAt time.Time
	CreatedAt time.Time
	Used      bool
}

// VerificationRepository 認証コードリポジトリインターフェース
type VerificationRepository interface {
	Create(userID string, expiresIn time.Duration) (*VerificationCode, error)
	GetByUserIDAndCode(userID, code string) (*VerificationCode, error)
	MarkAsUsed(id int64) error
	DeleteExpired() error
	DeleteByUserID(userID string) error
}

type verificationRepository struct {
	db *sql.DB
}

// NewVerificationRepository 認証コードリポジトリを作成
func NewVerificationRepository(db *sql.DB) VerificationRepository {
	return &verificationRepository{db: db}
}

// Create 新しい認証コードを生成
func (r *verificationRepository) Create(userID string, expiresIn time.Duration) (*VerificationCode, error) {
	// 6桁のランダムコード生成
	code, err := generateVerificationCode()
	if err != nil {
		return nil, fmt.Errorf("認証コード生成失敗: %w", err)
	}

	verification := &VerificationCode{
		UserID:    userID,
		Code:      code,
		ExpiresAt: time.Now().Add(expiresIn),
		CreatedAt: time.Now(),
		Used:      false,
	}

	query := `
		INSERT INTO verification_codes (user_id, code, expires_at, created_at, used)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`
	err = r.db.QueryRow(query, verification.UserID, verification.Code, verification.ExpiresAt, verification.CreatedAt, verification.Used).Scan(&verification.ID)
	if err != nil {
		return nil, fmt.Errorf("認証コード保存失敗: %w", err)
	}

	return verification, nil
}

// GetByUserIDAndCode ユーザーIDとコードで認証コードを取得
func (r *verificationRepository) GetByUserIDAndCode(userID, code string) (*VerificationCode, error) {
	verification := &VerificationCode{}
	query := `
		SELECT id, user_id, code, expires_at, created_at, used
		FROM verification_codes
		WHERE user_id = $1 AND code = $2 AND used = FALSE
		ORDER BY created_at DESC
		LIMIT 1
	`
	err := r.db.QueryRow(query, userID, code).Scan(
		&verification.ID,
		&verification.UserID,
		&verification.Code,
		&verification.ExpiresAt,
		&verification.CreatedAt,
		&verification.Used,
	)
	if err == sql.ErrNoRows {
		return nil, nil // 認証コードが見つからない場合はnilを返す
	}
	if err != nil {
		return nil, fmt.Errorf("認証コード取得失敗: %w", err)
	}
	return verification, nil
}

// MarkAsUsed 認証コードを使用済みにする
func (r *verificationRepository) MarkAsUsed(id int64) error {
	query := `UPDATE verification_codes SET used = TRUE WHERE id = $1`
	_, err := r.db.Exec(query, id)
	return err
}

// DeleteExpired 期限切れの認証コードを削除
func (r *verificationRepository) DeleteExpired() error {
	query := `DELETE FROM verification_codes WHERE expires_at < $1`
	_, err := r.db.Exec(query, time.Now())
	return err
}

// DeleteByUserID ユーザーの認証コードを全て削除
func (r *verificationRepository) DeleteByUserID(userID string) error {
	query := `DELETE FROM verification_codes WHERE user_id = $1`
	_, err := r.db.Exec(query, userID)
	return err
}

// generateVerificationCode 6桁のランダムな認証コードを生成
func generateVerificationCode() (string, error) {
	const digits = "0123456789"
	code := make([]byte, 6)
	for i := range code {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			return "", err
		}
		code[i] = digits[n.Int64()]
	}
	return string(code), nil
}
