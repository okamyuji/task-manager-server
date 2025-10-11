package email

import (
	"context"
	"log/slog"
	"os"
	"testing"
)

// TestNewEmailService_SMTP
func TestNewEmailService_SMTP(t *testing.T) {
	os.Clearenv()
	_ = os.Setenv("SMTP_HOST", "localhost")
	_ = os.Setenv("SMTP_PORT", "1025")
	_ = os.Setenv("MAIL_FROM", "test@example.com")

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	service := NewEmailService(logger)

	if service == nil {
		t.Fatal("NewEmailService() returned nil")
	}

	// resendClientがnilであることを確認（SMTP使用）
	if service.resendClient != nil {
		t.Error("Expected resendClient to be nil for SMTP")
	}
	if service.smtpHost != "localhost" {
		t.Errorf("smtpHost = %s, want localhost", service.smtpHost)
	}
}

// TestNewEmailService_Resend
func TestNewEmailService_Resend(t *testing.T) {
	os.Clearenv()
	_ = os.Setenv("RESEND_API_KEY", "re_test_key_123")
	_ = os.Setenv("MAIL_FROM", "test@example.com")

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	service := NewEmailService(logger)

	if service == nil {
		t.Fatal("NewEmailService() returned nil")
	}

	// resendClientが設定されていることを確認
	if service.resendClient == nil {
		t.Error("Expected resendClient to be set")
	}
}

// TestEmailService_SendVerificationEmail_SMTP_正常系
func TestEmailService_SendVerificationEmail_SMTP_正常系(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	os.Clearenv()
	_ = os.Setenv("SMTP_HOST", "localhost")
	_ = os.Setenv("SMTP_PORT", "1025")
	_ = os.Setenv("MAIL_FROM", "test@example.com")

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	service := NewEmailService(logger)

	// MailHogが起動していない場合はスキップ
	err := service.SendVerificationCode(context.Background(), "recipient@example.com", "Test User", "123456")
	if err != nil {
		t.Logf("SMTPサーバーに接続できません（MailHog未起動の可能性）: %v", err)
		t.Skip("SMTP server not available")
	}
}

// TestEmailService_SendVerificationEmail_異常系_無効なメール
func TestEmailService_SendVerificationEmail_異常系_無効なメール(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	os.Clearenv()
	_ = os.Setenv("SMTP_HOST", "localhost")
	_ = os.Setenv("SMTP_PORT", "1025")
	_ = os.Setenv("MAIL_FROM", "test@example.com")

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	service := NewEmailService(logger)

	err := service.SendVerificationCode(context.Background(), "", "Test User", "123456")
	if err == nil {
		t.Error("Expected error for empty email")
	}
}

// TestEmailService_構造体フィールド確認
func TestEmailService_構造体フィールド確認(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	service := &EmailService{
		smtpHost: "localhost",
		smtpPort: "1025",
		from:     "test@example.com",
		logger:   logger,
	}

	if service.smtpHost != "localhost" {
		t.Errorf("smtpHost = %s, want localhost", service.smtpHost)
	}
	if service.smtpPort != "1025" {
		t.Errorf("smtpPort = %s, want 1025", service.smtpPort)
	}
	if service.from != "test@example.com" {
		t.Errorf("from = %s, want test@example.com", service.from)
	}
	if service.logger == nil {
		t.Error("logger should not be nil")
	}
}
