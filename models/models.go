package models

import (
	"encoding/json"
	"strings"
	"sync"
	"time"
)

// FlexibleTime 柔軟な時刻パース用のカスタム型
type FlexibleTime struct {
	time.Time
}

// UnmarshalJSON 複数の日時フォーマットに対応
func (ft *FlexibleTime) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), "\"")
	if s == "null" || s == "" {
		ft.Time = time.Time{}
		return nil
	}

	// 試行する日時フォーマット
	formats := []string{
		time.RFC3339,                 // 2006-01-02T15:04:05Z07:00
		time.RFC3339Nano,             // 2006-01-02T15:04:05.999999999Z07:00
		"2006-01-02T15:04:05.999999", // マイクロ秒（タイムゾーンなし）
		"2006-01-02T15:04:05",        // 秒（タイムゾーンなし）
		"2006-01-02T15:04:05Z",       // UTC
		"2006-01-02",                 // 日付のみ
	}

	var err error
	for _, format := range formats {
		ft.Time, err = time.Parse(format, s)
		if err == nil {
			return nil
		}
	}

	return err
}

// MarshalJSON RFC3339形式で出力
func (ft FlexibleTime) MarshalJSON() ([]byte, error) {
	if ft.IsZero() {
		return []byte("null"), nil
	}
	return json.Marshal(ft.Format(time.RFC3339))
}

// Task タスクモデル
type Task struct {
	ID          string        `json:"id"`
	UserID      string        `json:"userId"` // タスクの所有者
	Title       string        `json:"title"`
	Description string        `json:"description"`
	CreatedAt   FlexibleTime  `json:"createdAt"`
	DueDate     *FlexibleTime `json:"dueDate,omitempty"`
	IsCompleted bool          `json:"isCompleted"`
	CompletedAt *FlexibleTime `json:"completedAt,omitempty"`
	Tags        []string      `json:"tags"`
	Priority    string        `json:"priority"`           // low, medium, high, urgent
	ImageUrl    *string       `json:"imageUrl,omitempty"` // 添付画像URL
}

// TaskStore タスクのインメモリストア
type TaskStore struct {
	Tasks map[string]*Task
	Mutex *sync.RWMutex
}

// User ユーザーモデル
type User struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Password string `json:"-"` // JSONには含めない
	Name     string `json:"name"`
}

// LoginRequest ログインリクエスト
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// RegisterRequest 登録リクエスト
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

// AuthResponse 認証レスポンス
type AuthResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	UserID       string `json:"userId"`
}

// RefreshRequest トークンリフレッシュリクエスト
type RefreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

// RefreshResponse トークンリフレッシュレスポンス
type RefreshResponse struct {
	AccessToken string `json:"accessToken"`
}

// ErrorResponse エラーレスポンス
type ErrorResponse struct {
	Message string `json:"message"`
}

// UploadResponse 画像アップロードレスポンス
type UploadResponse struct {
	URL string `json:"url"`
}

// VerifyRequest 認証リクエスト
type VerifyRequest struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

// ResendCodeRequest 認証コード再送信リクエスト
type ResendCodeRequest struct {
	Email string `json:"email"`
}
