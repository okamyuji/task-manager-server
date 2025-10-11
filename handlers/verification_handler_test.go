package handlers

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"task_manager_server/database/repository"
	"task_manager_server/models"
)

// TestVerificationHandler_HandleVerify_正常系
func TestVerificationHandler_HandleVerify_正常系(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	userRepo := newMockUserRepository()
	verifyRepo := newMockVerificationRepository()
	emailService := &mockEmailService{}

	handler := NewVerificationHandler(userRepo, verifyRepo, emailService, logger)

	// 未認証ユーザーを作成
	user := &repository.User{
		ID:         "user1",
		Email:      "verify@example.com",
		Name:       "Test User",
		IsVerified: false,
	}
	_ = userRepo.Create(user)

	// 認証コードを作成
	_, _ = verifyRepo.Create(user.ID, 15*time.Minute)

	reqBody := models.VerifyRequest{
		Email: "verify@example.com",
		Code:  "123456",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/auth/verify", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.HandleVerify(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
}

// TestVerificationHandler_HandleVerify_異常系_無効なコード
func TestVerificationHandler_HandleVerify_異常系_無効なコード(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	userRepo := newMockUserRepository()
	verifyRepo := newMockVerificationRepository()
	emailService := &mockEmailService{}

	handler := NewVerificationHandler(userRepo, verifyRepo, emailService, logger)

	user := &repository.User{
		ID:         "user1",
		Email:      "verify@example.com",
		Name:       "Test User",
		IsVerified: false,
	}
	_ = userRepo.Create(user)

	reqBody := models.VerifyRequest{
		Email: "verify@example.com",
		Code:  "999999",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/auth/verify", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.HandleVerify(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}
}

// TestVerificationHandler_HandleVerify_異常系_既に認証済み
func TestVerificationHandler_HandleVerify_異常系_既に認証済み(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	userRepo := newMockUserRepository()
	verifyRepo := newMockVerificationRepository()
	emailService := &mockEmailService{}

	handler := NewVerificationHandler(userRepo, verifyRepo, emailService, logger)

	user := &repository.User{
		ID:         "user1",
		Email:      "verified@example.com",
		Name:       "Test User",
		IsVerified: true,
	}
	_ = userRepo.Create(user)

	reqBody := models.VerifyRequest{
		Email: "verified@example.com",
		Code:  "123456",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/auth/verify", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.HandleVerify(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}
}

// TestVerificationHandler_HandleResendCode_正常系
func TestVerificationHandler_HandleResendCode_正常系(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	userRepo := newMockUserRepository()
	verifyRepo := newMockVerificationRepository()
	emailService := &mockEmailService{}

	handler := NewVerificationHandler(userRepo, verifyRepo, emailService, logger)

	user := &repository.User{
		ID:         "user1",
		Email:      "resend@example.com",
		Name:       "Test User",
		IsVerified: false,
	}
	_ = userRepo.Create(user)

	reqBody := models.ResendCodeRequest{
		Email: "resend@example.com",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/auth/resend", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.HandleResendCode(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
}

// TestVerificationHandler_HandleResendCode_異常系_既に認証済み
func TestVerificationHandler_HandleResendCode_異常系_既に認証済み(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	userRepo := newMockUserRepository()
	verifyRepo := newMockVerificationRepository()
	emailService := &mockEmailService{}

	handler := NewVerificationHandler(userRepo, verifyRepo, emailService, logger)

	user := &repository.User{
		ID:         "user1",
		Email:      "verified@example.com",
		Name:       "Test User",
		IsVerified: true,
	}
	_ = userRepo.Create(user)

	reqBody := models.ResendCodeRequest{
		Email: "verified@example.com",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/auth/resend", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.HandleResendCode(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}
}

// TestVerificationHandler_HandleResendCode_異常系_ユーザー不在
func TestVerificationHandler_HandleResendCode_異常系_ユーザー不在(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	userRepo := newMockUserRepository()
	verifyRepo := newMockVerificationRepository()
	emailService := &mockEmailService{}

	handler := NewVerificationHandler(userRepo, verifyRepo, emailService, logger)

	reqBody := models.ResendCodeRequest{
		Email: "nonexistent@example.com",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/auth/resend", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.HandleResendCode(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", rec.Code)
	}
}
