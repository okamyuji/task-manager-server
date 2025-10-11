package utils

import (
	"strings"
	"testing"
)

// TestHashPassword_正常系 パスワードハッシュ化のテスト
func TestHashPassword_正常系(t *testing.T) {
	tests := []struct {
		name     string
		password string
	}{
		{"通常のパスワード", "password123"},
		{"長いパスワード", "very-long-password-with-special-chars!@#$%^&*()"},
		{"短いパスワード", "pass"},
		{"特殊文字を含む", "P@ssw0rd!"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := HashPassword(tt.password)
			if err != nil {
				t.Errorf("HashPassword() error = %v", err)
				return
			}

			// ハッシュが生成されていることを確認
			if hash == "" {
				t.Error("HashPassword() ハッシュが空です")
			}

			// 元のパスワードと異なることを確認
			if hash == tt.password {
				t.Error("HashPassword() ハッシュが元のパスワードと同じです")
			}

			// bcryptハッシュのプレフィックスを確認
			if !strings.HasPrefix(hash, "$2a$") && !strings.HasPrefix(hash, "$2b$") {
				t.Errorf("HashPassword() 不正なbcryptハッシュ形式: %s", hash)
			}
		})
	}
}

// TestHashPassword_異常系 空のパスワード
func TestHashPassword_異常系_空のパスワード(t *testing.T) {
	// bcryptは空文字列も受け付けるので、エラーにならない
	// ただし、実際のアプリケーションではバリデーションで弾くべき
	hash, err := HashPassword("")
	if err != nil {
		t.Errorf("HashPassword(\"\") unexpected error = %v", err)
	}
	if hash == "" {
		t.Error("HashPassword(\"\") ハッシュが空です")
	}
}

// TestComparePassword_正常系 パスワード比較のテスト
func TestComparePassword_正常系(t *testing.T) {
	password := "test-password-123"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	err = ComparePassword(hash, password)
	if err != nil {
		t.Errorf("ComparePassword() error = %v, 正しいパスワードで比較失敗", err)
	}
}

// TestComparePassword_異常系_間違ったパスワード
func TestComparePassword_異常系_間違ったパスワード(t *testing.T) {
	password := "correct-password"
	wrongPassword := "wrong-password"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	err = ComparePassword(hash, wrongPassword)
	if err == nil {
		t.Error("ComparePassword() 間違ったパスワードでエラーが返されませんでした")
	}

	if err != nil && !strings.Contains(err.Error(), "パスワードが一致しません") {
		t.Errorf("ComparePassword() エラーメッセージが不正: %v", err)
	}
}

// TestComparePassword_異常系_不正なハッシュ
func TestComparePassword_異常系_不正なハッシュ(t *testing.T) {
	invalidHash := "invalid-hash-string"
	password := "test-password"

	err := ComparePassword(invalidHash, password)
	if err == nil {
		t.Error("ComparePassword() 不正なハッシュでエラーが返されませんでした")
	}
}

// TestComparePassword_異常系_空のハッシュ
func TestComparePassword_異常系_空のハッシュ(t *testing.T) {
	err := ComparePassword("", "password")
	if err == nil {
		t.Error("ComparePassword() 空のハッシュでエラーが返されませんでした")
	}
}

// TestHashPassword_一貫性 同じパスワードで異なるハッシュが生成されることを確認
func TestHashPassword_一貫性(t *testing.T) {
	password := "test-password"

	hash1, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	hash2, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	// bcryptは毎回異なるsaltを使うので、ハッシュは異なるはず
	if hash1 == hash2 {
		t.Error("HashPassword() 同じパスワードで同じハッシュが生成されました（saltが機能していない）")
	}

	// ただし、どちらも元のパスワードと一致するはず
	if err := ComparePassword(hash1, password); err != nil {
		t.Error("ComparePassword() hash1で比較失敗")
	}
	if err := ComparePassword(hash2, password); err != nil {
		t.Error("ComparePassword() hash2で比較失敗")
	}
}
