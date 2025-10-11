package handlers

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"task_manager_server/database/repository"
	"task_manager_server/models"
	"task_manager_server/utils"
)

// TestAuthHandler_HandleLogin_正常系
func TestAuthHandler_HandleLogin_正常系(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	utils.SetLogger(logger)

	userRepo := newMockUserRepository()
	verifyRepo := newMockVerificationRepository()
	emailService := &mockEmailService{}
	tokenService := &mockTokenService{}
	passwordHasher := &mockPasswordHasher{}

	handler := NewAuthHandler(userRepo, verifyRepo, emailService, tokenService, passwordHasher, logger)

	// 認証済みユーザーを作成
	passwordHash, _ := passwordHasher.HashPassword("password123")
	user := &repository.User{
		ID:           "user1",
		Email:        "login@example.com",
		PasswordHash: passwordHash,
		Name:         "Test User",
		IsVerified:   true,
	}
	_ = userRepo.Create(user)

	reqBody := models.LoginRequest{
		Email:    "login@example.com",
		Password: "password123",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.HandleLogin(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
}

// TestAuthHandler_HandleLogin_異常系_未認証
func TestAuthHandler_HandleLogin_異常系_未認証(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	utils.SetLogger(logger)

	userRepo := newMockUserRepository()
	verifyRepo := newMockVerificationRepository()
	emailService := &mockEmailService{}

	handler := NewAuthHandler(userRepo, verifyRepo, emailService, &mockTokenService{}, &mockPasswordHasher{}, logger)

	// 未認証ユーザーを作成
	passwordHasher := &mockPasswordHasher{}
	passwordHash, _ := passwordHasher.HashPassword("password123")
	user := &repository.User{
		Email:        "unverified@example.com",
		PasswordHash: passwordHash,
		Name:         "Unverified User",
		IsVerified:   false,
	}
	_ = userRepo.Create(user)

	reqBody := models.LoginRequest{
		Email:    "unverified@example.com",
		Password: "password123",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.HandleLogin(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", rec.Code)
	}
}

// TestAuthHandler_HandleRegister_正常系
func TestAuthHandler_HandleRegister_正常系(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	utils.SetLogger(logger)

	userRepo := newMockUserRepository()
	verifyRepo := newMockVerificationRepository()
	emailService := &mockEmailService{}

	handler := NewAuthHandler(userRepo, verifyRepo, emailService, &mockTokenService{}, &mockPasswordHasher{}, logger)

	reqBody := models.RegisterRequest{
		Email:    "newuser@example.com",
		Password: "Password123!",
		Name:     "New User",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.HandleRegister(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", rec.Code)
	}
}

// TestAuthHandler_HandleRegister_異常系_重複メール
func TestAuthHandler_HandleRegister_異常系_重複メール(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	utils.SetLogger(logger)

	userRepo := newMockUserRepository()
	verifyRepo := newMockVerificationRepository()
	emailService := &mockEmailService{}

	handler := NewAuthHandler(userRepo, verifyRepo, emailService, &mockTokenService{}, &mockPasswordHasher{}, logger)

	// 既存ユーザー作成
	user := &repository.User{
		Email:        "existing@example.com",
		PasswordHash: "hash",
		Name:         "Existing User",
	}
	_ = userRepo.Create(user)

	reqBody := models.RegisterRequest{
		Email:    "existing@example.com",
		Password: "Password123!",
		Name:     "Another User",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.HandleRegister(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("Expected status 409, got %d", rec.Code)
	}
}

// TestAuthHandler_HandleRefresh_正常系
func TestAuthHandler_HandleRefresh_正常系(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	utils.SetLogger(logger)

	userRepo := newMockUserRepository()
	verifyRepo := newMockVerificationRepository()
	emailService := &mockEmailService{}

	handler := NewAuthHandler(userRepo, verifyRepo, emailService, &mockTokenService{}, &mockPasswordHasher{}, logger)

	// リフレッシュトークン生成
	refreshToken, _ := utils.GenerateRefreshToken("user123", "refresh@example.com")

	reqBody := models.RefreshRequest{
		RefreshToken: refreshToken,
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.HandleRefresh(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
}
