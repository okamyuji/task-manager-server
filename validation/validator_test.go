package validation

import (
	"strings"
	"testing"
)

// TestValidateEmail_正常系
func TestValidateEmail_正常系(t *testing.T) {
	tests := []struct {
		name  string
		email string
	}{
		{"通常のメール", "test@example.com"},
		{"サブドメイン", "user@mail.example.com"},
		{"数字を含む", "user123@example.com"},
		{"ドットを含む", "user.name@example.com"},
		{"ハイフンを含む", "user-name@example-domain.com"},
		{"プラスを含む", "user+tag@example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEmail(tt.email)
			if err != nil {
				t.Errorf("ValidateEmail(%s) = %v, want nil", tt.email, err)
			}
		})
	}
}

// TestValidateEmail_異常系
func TestValidateEmail_異常系(t *testing.T) {
	tests := []struct {
		name  string
		email string
	}{
		{"空文字", ""},
		{"スペースのみ", "   "},
		{"@なし", "testexample.com"},
		{"ドメインなし", "test@"},
		{"ローカル部なし", "@example.com"},
		{"TLDなし", "test@example"},
		{"不正な文字", "test@exa mple.com"},
		{"255文字超", strings.Repeat("a", 244) + "@example.com"}, // 256文字
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEmail(tt.email)
			if err == nil {
				t.Errorf("ValidateEmail(%s) = nil, want error", tt.email)
			}
		})
	}
}

// TestValidatePassword_正常系
func TestValidatePassword_正常系(t *testing.T) {
	tests := []struct {
		name     string
		password string
	}{
		{"英数字8文字", "pass1234"},
		{"英数字と特殊文字", "P@ssw0rd!"},
		{"長いパスワード", "VeryLongPassword12345678"},
		{"最小要件", "abcd1234"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.password)
			if err != nil {
				t.Errorf("ValidatePassword(%s) = %v, want nil", tt.password, err)
			}
		})
	}
}

// TestValidatePassword_異常系
func TestValidatePassword_異常系(t *testing.T) {
	tests := []struct {
		name     string
		password string
		contains string // エラーメッセージに含まれるべき文字列
	}{
		{"空文字", "", "必須"},
		{"7文字", "pass123", "8文字以上"},
		{"数字なし", "password", "英字と数字"},
		{"英字なし", "12345678", "英字と数字"},
		{"128文字超", strings.Repeat("a", 129), "128文字以内"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.password)
			if err == nil {
				t.Errorf("ValidatePassword(%s) = nil, want error", tt.password)
				return
			}
			if !strings.Contains(err.Error(), tt.contains) {
				t.Errorf("ValidatePassword(%s) error = %v, want contains %s", tt.password, err, tt.contains)
			}
		})
	}
}

// TestValidateName_正常系
func TestValidateName_正常系(t *testing.T) {
	tests := []struct {
		name     string
		userName string
	}{
		{"日本語名", "山田太郎"},
		{"英語名", "John Doe"},
		{"短い名前", "A"},
		{"100文字", strings.Repeat("a", 100)}, // 英字で100文字
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateName(tt.userName)
			if err != nil {
				t.Errorf("ValidateName(%s) = %v, want nil", tt.userName, err)
			}
		})
	}
}

// TestValidateName_異常系
func TestValidateName_異常系(t *testing.T) {
	tests := []struct {
		name     string
		userName string
	}{
		{"空文字", ""},
		{"スペースのみ", "   "},
		{"100文字超", strings.Repeat("あ", 101)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateName(tt.userName)
			if err == nil {
				t.Errorf("ValidateName(%s) = nil, want error", tt.userName)
			}
		})
	}
}

// TestValidateTaskTitle_正常系
func TestValidateTaskTitle_正常系(t *testing.T) {
	tests := []struct {
		name  string
		title string
	}{
		{"通常のタイトル", "タスクを完了する"},
		{"英語タイトル", "Complete the task"},
		{"200文字", strings.Repeat("a", 200)}, // 英字で200文字
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTaskTitle(tt.title)
			if err != nil {
				t.Errorf("ValidateTaskTitle(%s) = %v, want nil", tt.title, err)
			}
		})
	}
}

// TestValidateTaskTitle_異常系
func TestValidateTaskTitle_異常系(t *testing.T) {
	tests := []struct {
		name  string
		title string
	}{
		{"空文字", ""},
		{"スペースのみ", "   "},
		{"200文字超", strings.Repeat("あ", 201)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTaskTitle(tt.title)
			if err == nil {
				t.Errorf("ValidateTaskTitle(%s) = nil, want error", tt.title)
			}
		})
	}
}

// TestValidateTaskDescription_正常系
func TestValidateTaskDescription_正常系(t *testing.T) {
	tests := []struct {
		name        string
		description string
	}{
		{"空文字（説明は任意）", ""},
		{"通常の説明", "これはタスクの説明です"},
		{"2000文字", strings.Repeat("a", 2000)}, // 英字で2000文字
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTaskDescription(tt.description)
			if err != nil {
				t.Errorf("ValidateTaskDescription() = %v, want nil", err)
			}
		})
	}
}

// TestValidateTaskDescription_異常系
func TestValidateTaskDescription_異常系(t *testing.T) {
	description := strings.Repeat("あ", 2001) // 2001文字
	err := ValidateTaskDescription(description)
	if err == nil {
		t.Error("ValidateTaskDescription() = nil, want error for 2001 chars")
	}
}

// TestValidateTaskPriority_正常系
func TestValidateTaskPriority_正常系(t *testing.T) {
	tests := []struct {
		name     string
		priority string
	}{
		{"low", "low"},
		{"medium", "medium"},
		{"high", "high"},
		{"urgent", "urgent"},
		{"空文字（優先度は任意）", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTaskPriority(tt.priority)
			if err != nil {
				t.Errorf("ValidateTaskPriority(%s) = %v, want nil", tt.priority, err)
			}
		})
	}
}

// TestValidateTaskPriority_異常系
func TestValidateTaskPriority_異常系(t *testing.T) {
	tests := []struct {
		name     string
		priority string
	}{
		{"不正な値", "invalid"},
		{"大文字", "HIGH"},
		{"数字", "1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTaskPriority(tt.priority)
			if err == nil {
				t.Errorf("ValidateTaskPriority(%s) = nil, want error", tt.priority)
			}
		})
	}
}

// TestValidateVerificationCode_正常系
func TestValidateVerificationCode_正常系(t *testing.T) {
	tests := []struct {
		name string
		code string
	}{
		{"6桁数字", "123456"},
		{"0から始まる", "012345"},
		{"全て9", "999999"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateVerificationCode(tt.code)
			if err != nil {
				t.Errorf("ValidateVerificationCode(%s) = %v, want nil", tt.code, err)
			}
		})
	}
}

// TestValidateVerificationCode_異常系
func TestValidateVerificationCode_異常系(t *testing.T) {
	tests := []struct {
		name string
		code string
	}{
		{"空文字", ""},
		{"5桁", "12345"},
		{"7桁", "1234567"},
		{"英字を含む", "12345a"},
		{"記号を含む", "12345-"},
		{"スペースを含む", "123 456"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateVerificationCode(tt.code)
			if err == nil {
				t.Errorf("ValidateVerificationCode(%s) = nil, want error", tt.code)
			}
		})
	}
}

// TestValidationError_Error ValidationErrorのError()メソッドのテスト
func TestValidationError_Error(t *testing.T) {
	err := &ValidationError{
		Field:   "email",
		Message: "無効なメールアドレス形式です",
	}

	expected := "email: 無効なメールアドレス形式です"
	if err.Error() != expected {
		t.Errorf("ValidationError.Error() = %s, want %s", err.Error(), expected)
	}
}
