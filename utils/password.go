package utils

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

const (
	// bcryptコスト係数（12 = 2^12回のハッシュ処理）
	bcryptCost = 12
)

// PasswordHasher パスワードハッシュ化インターフェース
type PasswordHasher interface {
	HashPassword(password string) (string, error)
	ComparePassword(hashedPassword, password string) error
}

// BcryptPasswordHasher bcryptを使用したパスワードハッシャー
type BcryptPasswordHasher struct{}

// NewPasswordHasher パスワードハッシャーを作成
func NewPasswordHasher() PasswordHasher {
	return &BcryptPasswordHasher{}
}

// HashPassword パスワードをbcryptでハッシュ化
func (h *BcryptPasswordHasher) HashPassword(password string) (string, error) {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", fmt.Errorf("パスワードハッシュ化失敗: %w", err)
	}
	return string(hashedBytes), nil
}

// ComparePassword パスワードとハッシュを比較
func (h *BcryptPasswordHasher) ComparePassword(hashedPassword, password string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		return fmt.Errorf("パスワードが一致しません")
	}
	return nil
}

// 後方互換性のためのグローバル関数（テストで使用）
var defaultHasher = NewPasswordHasher()

func HashPassword(password string) (string, error) {
	return defaultHasher.HashPassword(password)
}

func ComparePassword(hashedPassword, password string) error {
	return defaultHasher.ComparePassword(hashedPassword, password)
}
