package utils

import (
	"log/slog"
	"os"
	"testing"
	"time"
)

// TestGenerateAccessToken_正常系
func TestGenerateAccessToken_正常系(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	SetLogger(logger)

	token, err := GenerateAccessToken("user123", "test@example.com")
	if err != nil {
		t.Fatalf("GenerateAccessToken() error = %v", err)
	}

	if token == "" {
		t.Error("GenerateAccessToken() returned empty token")
	}
}

// TestGenerateRefreshToken_正常系
func TestGenerateRefreshToken_正常系(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	SetLogger(logger)

	token, err := GenerateRefreshToken("user123", "test@example.com")
	if err != nil {
		t.Fatalf("GenerateRefreshToken() error = %v", err)
	}

	if token == "" {
		t.Error("GenerateRefreshToken() returned empty token")
	}
}

// TestVerifyToken_正常系
func TestVerifyToken_正常系(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	SetLogger(logger)

	token, err := GenerateAccessToken("user456", "verify@example.com")
	if err != nil {
		t.Fatalf("GenerateAccessToken() error = %v", err)
	}

	claims, err := VerifyToken(token)
	if err != nil {
		t.Fatalf("VerifyToken() error = %v", err)
	}

	if claims.UserID != "user456" {
		t.Errorf("UserID = %s, want user456", claims.UserID)
	}
	if claims.Email != "verify@example.com" {
		t.Errorf("Email = %s, want verify@example.com", claims.Email)
	}
}

// TestVerifyToken_異常系_無効なトークン
func TestVerifyToken_異常系_無効なトークン(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	SetLogger(logger)

	_, err := VerifyToken("invalid.token.here")
	if err == nil {
		t.Error("VerifyToken() expected error for invalid token")
	}
}

// TestVerifyToken_異常系_期限切れ
func TestVerifyToken_異常系_期限切れ(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	SetLogger(logger)

	// 負の期限でトークン生成（即座に期限切れ）
	service := &JWTTokenService{logger: logger}
	token, err := service.generateToken("user789", "expired@example.com", -1*time.Hour)
	if err != nil {
		t.Fatalf("generateToken() error = %v", err)
	}

	_, err = VerifyToken(token)
	if err == nil {
		t.Error("VerifyToken() expected error for expired token")
	}
}

// TestVerifyToken_異常系_不正な署名
func TestVerifyToken_異常系_不正な署名(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	SetLogger(logger)

	token, err := GenerateAccessToken("user999", "tamper@example.com")
	if err != nil {
		t.Fatalf("GenerateAccessToken() error = %v", err)
	}

	// トークンを改ざん
	tamperedToken := token + "tampered"

	_, err = VerifyToken(tamperedToken)
	if err == nil {
		t.Error("VerifyToken() expected error for tampered token")
	}
}

// TestVerifyToken_異常系_不正なフォーマット
func TestVerifyToken_異常系_不正なフォーマット(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	SetLogger(logger)

	tests := []struct {
		name  string
		token string
	}{
		{"空文字", ""},
		{"1パート", "onlyonepart"},
		{"2パート", "two.parts"},
		{"4パート", "four.parts.are.invalid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := VerifyToken(tt.token)
			if err == nil {
				t.Errorf("VerifyToken(%s) expected error", tt.name)
			}
		})
	}
}

// TestJWTClaims_有効期限チェック
func TestJWTClaims_有効期限チェック(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	SetLogger(logger)

	token, err := GenerateAccessToken("user_exp", "exp@example.com")
	if err != nil {
		t.Fatalf("GenerateAccessToken() error = %v", err)
	}

	claims, err := VerifyToken(token)
	if err != nil {
		t.Fatalf("VerifyToken() error = %v", err)
	}

	// 有効期限が未来であることを確認
	if claims.Exp <= time.Now().Unix() {
		t.Error("Token expiration should be in the future")
	}

	// 発行時刻が過去または現在であることを確認
	if claims.Iat > time.Now().Unix() {
		t.Error("Token issued at should be in the past or present")
	}
}
