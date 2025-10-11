package validation

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	// メールアドレスの正規表現
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
)

// ValidationError バリデーションエラー
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidateEmail メールアドレスのバリデーション
func ValidateEmail(email string) error {
	email = strings.TrimSpace(email)
	if email == "" {
		return &ValidationError{Field: "email", Message: "メールアドレスは必須です"}
	}
	if len(email) > 255 {
		return &ValidationError{Field: "email", Message: "メールアドレスは255文字以内である必要があります"}
	}
	if !emailRegex.MatchString(email) {
		return &ValidationError{Field: "email", Message: "無効なメールアドレス形式です"}
	}
	return nil
}

// ValidatePassword パスワードのバリデーション
func ValidatePassword(password string) error {
	if password == "" {
		return &ValidationError{Field: "password", Message: "パスワードは必須です"}
	}
	if len(password) < 8 {
		return &ValidationError{Field: "password", Message: "パスワードは8文字以上である必要があります"}
	}
	if len(password) > 128 {
		return &ValidationError{Field: "password", Message: "パスワードは128文字以内である必要があります"}
	}

	// 英字と数字を含むかチェック
	hasLetter := regexp.MustCompile(`[a-zA-Z]`).MatchString(password)
	hasDigit := regexp.MustCompile(`[0-9]`).MatchString(password)

	if !hasLetter || !hasDigit {
		return &ValidationError{Field: "password", Message: "パスワードは英字と数字を含む必要があります"}
	}

	return nil
}

// ValidateName 名前のバリデーション
func ValidateName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return &ValidationError{Field: "name", Message: "名前は必須です"}
	}
	if len(name) > 100 {
		return &ValidationError{Field: "name", Message: "名前は100文字以内である必要があります"}
	}
	return nil
}

// ValidateTaskTitle タスクタイトルのバリデーション
func ValidateTaskTitle(title string) error {
	title = strings.TrimSpace(title)
	if title == "" {
		return &ValidationError{Field: "title", Message: "タイトルは必須です"}
	}
	if len(title) > 200 {
		return &ValidationError{Field: "title", Message: "タイトルは200文字以内である必要があります"}
	}
	return nil
}

// ValidateTaskDescription タスク説明のバリデーション
func ValidateTaskDescription(description string) error {
	if len(description) > 2000 {
		return &ValidationError{Field: "description", Message: "説明は2000文字以内である必要があります"}
	}
	return nil
}

// ValidateTaskPriority タスク優先度のバリデーション
func ValidateTaskPriority(priority string) error {
	validPriorities := map[string]bool{
		"low":    true,
		"medium": true,
		"high":   true,
		"urgent": true,
	}

	if priority != "" && !validPriorities[priority] {
		return &ValidationError{Field: "priority", Message: "優先度はlow, medium, high, urgentのいずれかである必要があります"}
	}
	return nil
}

// ValidateVerificationCode 認証コードのバリデーション
func ValidateVerificationCode(code string) error {
	code = strings.TrimSpace(code)
	if code == "" {
		return &ValidationError{Field: "code", Message: "認証コードは必須です"}
	}
	if len(code) != 6 {
		return &ValidationError{Field: "code", Message: "認証コードは6桁である必要があります"}
	}
	if !regexp.MustCompile(`^\d{6}$`).MatchString(code) {
		return &ValidationError{Field: "code", Message: "認証コードは数字のみである必要があります"}
	}
	return nil
}
