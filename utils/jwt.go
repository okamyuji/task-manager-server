package utils

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	jwtSecret     []byte
	jwtSecretOnce sync.Once
	logger        *slog.Logger
)

// TokenService JWT トークンサービスインターフェース
type TokenService interface {
	GenerateAccessToken(userID, email string) (string, error)
	GenerateRefreshToken(userID, email string) (string, error)
	VerifyToken(tokenString string) (*JWTClaims, error)
}

// JWTTokenService JWT トークンサービス実装
type JWTTokenService struct {
	logger *slog.Logger
}

// NewTokenService トークンサービスを作成
func NewTokenService(l *slog.Logger) TokenService {
	return &JWTTokenService{logger: l}
}

// SetLogger ロガーを設定
func SetLogger(l *slog.Logger) {
	logger = l
}

// getJWTSecret JWT秘密鍵を環境変数から取得（遅延初期化）
func getJWTSecret() []byte {
	jwtSecretOnce.Do(func() {
		secret := os.Getenv("JWT_SECRET")
		if secret == "" {
			// JWT_SECRETが未設定の場合はエラー
			// 注意: logger初期化前に呼ばれる可能性があるため、標準出力に出力
			fmt.Println("エラー: JWT_SECRETが設定されていません。")
			fmt.Println("以下のコマンドで秘密鍵を生成してください:")
			fmt.Println("  openssl rand -base64 32")
			fmt.Println("その後、環境変数を設定してください:")
			fmt.Println("  export JWT_SECRET=\"生成された秘密鍵\"")
			os.Exit(1)
		}
		jwtSecret = []byte(secret)
	})
	return jwtSecret
}

// JWTClaims JWTのペイロード
type JWTClaims struct {
	UserID string `json:"userId"`
	Email  string `json:"email"`
	Exp    int64  `json:"exp"` // 有効期限（Unix timestamp）
	Iat    int64  `json:"iat"` // 発行時刻（Unix timestamp）
}

// GenerateAccessToken アクセストークンを生成（15分有効）
func (s *JWTTokenService) GenerateAccessToken(userID, email string) (string, error) {
	if s.logger != nil {
		s.logger.Debug("アクセストークン生成開始", "user_id", userID)
	}
	return s.generateToken(userID, email, 15*time.Minute)
}

// GenerateRefreshToken リフレッシュトークンを生成（7日有効）
func (s *JWTTokenService) GenerateRefreshToken(userID, email string) (string, error) {
	if s.logger != nil {
		s.logger.Debug("リフレッシュトークン生成開始", "user_id", userID)
	}
	return s.generateToken(userID, email, 7*24*time.Hour)
}

// generateToken JWTトークンを生成
func (s *JWTTokenService) generateToken(userID, email string, expiration time.Duration) (string, error) {
	now := time.Now()
	claims := JWTClaims{
		UserID: userID,
		Email:  email,
		Exp:    now.Add(expiration).Unix(),
		Iat:    now.Unix(),
	}

	// ヘッダー
	header := map[string]string{
		"alg": "HS256",
		"typ": "JWT",
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		if s.logger != nil {
			s.logger.Error("JWTヘッダーのマーシャルエラー", "error", err)
		}
		return "", err
	}

	// ペイロード
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		if s.logger != nil {
			s.logger.Error("JWTクレームのマーシャルエラー", "error", err)
		}
		return "", err
	}

	// Base64エンコード
	headerEncoded := base64.RawURLEncoding.EncodeToString(headerJSON)
	claimsEncoded := base64.RawURLEncoding.EncodeToString(claimsJSON)

	// 署名
	message := headerEncoded + "." + claimsEncoded
	signature := createSignature(message, getJWTSecret())

	// トークン生成
	token := message + "." + signature

	if s.logger != nil {
		s.logger.Debug("JWTトークン生成成功",
			"user_id", userID,
			"expiration_minutes", expiration.Minutes(),
		)
	}

	return token, nil
}

// VerifyToken トークンを検証してクレームを返す
func (s *JWTTokenService) VerifyToken(tokenString string) (*JWTClaims, error) {
	// トークンを分割
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		if s.logger != nil {
			s.logger.Debug("トークンフォーマットエラー", "parts_count", len(parts))
		}
		return nil, errors.New("invalid token format")
	}

	headerEncoded := parts[0]
	claimsEncoded := parts[1]
	signatureEncoded := parts[2]

	// 署名検証
	message := headerEncoded + "." + claimsEncoded
	expectedSignature := createSignature(message, getJWTSecret())

	if signatureEncoded != expectedSignature {
		if s.logger != nil {
			s.logger.Warn("JWT署名検証失敗")
		}
		return nil, errors.New("invalid signature")
	}

	// クレームをデコード
	claimsJSON, err := base64.RawURLEncoding.DecodeString(claimsEncoded)
	if err != nil {
		if s.logger != nil {
			s.logger.Warn("クレームデコードエラー", "error", err)
		}
		return nil, fmt.Errorf("failed to decode claims: %w", err)
	}

	var claims JWTClaims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		if s.logger != nil {
			s.logger.Warn("クレームアンマーシャルエラー", "error", err)
		}
		return nil, fmt.Errorf("failed to unmarshal claims: %w", err)
	}

	// 有効期限チェック
	now := time.Now().Unix()
	if now > claims.Exp {
		if s.logger != nil {
			s.logger.Debug("トークン期限切れ",
				"exp", claims.Exp,
				"now", now,
				"user_id", claims.UserID,
			)
		}
		return nil, errors.New("token expired")
	}

	if s.logger != nil {
		s.logger.Debug("トークン検証成功", "user_id", claims.UserID, "email", claims.Email)
	}
	return &claims, nil
}

// 後方互換性のためのグローバル関数
var defaultTokenService = NewTokenService(logger)

func GenerateAccessToken(userID, email string) (string, error) {
	return defaultTokenService.GenerateAccessToken(userID, email)
}

func GenerateRefreshToken(userID, email string) (string, error) {
	return defaultTokenService.GenerateRefreshToken(userID, email)
}

func VerifyToken(tokenString string) (*JWTClaims, error) {
	return defaultTokenService.VerifyToken(tokenString)
}

// createSignature 署名を作成
func createSignature(message string, secret []byte) string {
	h := hmac.New(sha256.New, secret)
	h.Write([]byte(message))
	signature := h.Sum(nil)
	return base64.RawURLEncoding.EncodeToString(signature)
}
