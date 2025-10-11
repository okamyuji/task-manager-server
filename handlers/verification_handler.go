package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"task_manager_server/database/repository"
	"task_manager_server/email"
	"task_manager_server/models"
	"task_manager_server/validation"
)

// VerificationHandler 認証コード関連のハンドラー
type VerificationHandler struct {
	userRepo         repository.UserRepository
	verificationRepo repository.VerificationRepository
	emailService     email.EmailSender
	logger           *slog.Logger
}

// NewVerificationHandler 認証ハンドラーを作成
func NewVerificationHandler(
	userRepo repository.UserRepository,
	verificationRepo repository.VerificationRepository,
	emailService email.EmailSender,
	logger *slog.Logger,
) *VerificationHandler {
	return &VerificationHandler{
		userRepo:         userRepo,
		verificationRepo: verificationRepo,
		emailService:     emailService,
		logger:           logger,
	}
}

// HandleVerify 認証コードを検証
func (h *VerificationHandler) HandleVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.logger.Warn("不正なHTTPメソッド", "method", r.Method, "path", r.URL.Path)
		RespondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req models.VerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Warn("リクエストボディのパースエラー", "error", err)
		RespondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// バリデーション
	if err := validation.ValidateEmail(req.Email); err != nil {
		h.logger.Warn("バリデーションエラー", "error", err)
		RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := validation.ValidateVerificationCode(req.Code); err != nil {
		h.logger.Warn("バリデーションエラー", "error", err)
		RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.logger.Info("認証コード検証試行", "email", req.Email)

	// ユーザー取得
	user, err := h.userRepo.GetByEmail(req.Email)
	if err != nil {
		h.logger.Error("ユーザー取得エラー", "error", err)
		RespondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	if user == nil {
		h.logger.Warn("ユーザーが見つかりません", "email", req.Email)
		RespondWithError(w, http.StatusNotFound, "User not found")
		return
	}

	// 既に認証済みの場合
	if user.IsVerified {
		h.logger.Warn("既に認証済みのユーザー", "user_id", user.ID, "email", req.Email)
		RespondWithError(w, http.StatusBadRequest, "User already verified")
		return
	}

	// 認証コード検証
	verification, err := h.verificationRepo.GetByUserIDAndCode(user.ID, req.Code)
	if err != nil {
		h.logger.Error("認証コード取得エラー", "error", err)
		RespondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	if verification == nil {
		h.logger.Warn("無効な認証コード", "user_id", user.ID, "code", req.Code)
		RespondWithError(w, http.StatusBadRequest, "Invalid verification code")
		return
	}

	// 有効期限チェック
	if time.Now().After(verification.ExpiresAt) {
		h.logger.Warn("認証コード期限切れ", "user_id", user.ID, "expires_at", verification.ExpiresAt)
		RespondWithError(w, http.StatusBadRequest, "Verification code expired")
		return
	}

	// ユーザーを認証済みにする
	if err := h.userRepo.VerifyUser(user.ID); err != nil {
		h.logger.Error("ユーザー認証エラー", "error", err)
		RespondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	// 認証コードを使用済みにする
	if err := h.verificationRepo.MarkAsUsed(verification.ID); err != nil {
		h.logger.Error("認証コード使用済み更新エラー", "error", err)
		// 続行（エラーでも問題なし）
	}

	h.logger.Info("ユーザー認証成功", "user_id", user.ID, "email", req.Email)

	// ウェルカムメール送信（非同期、エラーは無視）
	go func() {
		if err := h.emailService.SendWelcomeEmail(r.Context(), user.Email, user.Name); err != nil {
			h.logger.Warn("ウェルカムメール送信失敗", "error", err, "email", user.Email)
		}
	}()

	RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Verification successful",
		"userId":  user.ID,
	})
}

// HandleResendCode 認証コードを再送信
func (h *VerificationHandler) HandleResendCode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.logger.Warn("不正なHTTPメソッド", "method", r.Method, "path", r.URL.Path)
		RespondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req models.ResendCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Warn("リクエストボディのパースエラー", "error", err)
		RespondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// バリデーション
	if err := validation.ValidateEmail(req.Email); err != nil {
		h.logger.Warn("バリデーションエラー", "error", err)
		RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.logger.Info("認証コード再送信試行", "email", req.Email)

	// ユーザー取得
	user, err := h.userRepo.GetByEmail(req.Email)
	if err != nil {
		h.logger.Error("ユーザー取得エラー", "error", err)
		RespondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	if user == nil {
		h.logger.Warn("ユーザーが見つかりません", "email", req.Email)
		RespondWithError(w, http.StatusNotFound, "User not found")
		return
	}

	// 既に認証済みの場合
	if user.IsVerified {
		h.logger.Warn("既に認証済みのユーザー", "user_id", user.ID, "email", req.Email)
		RespondWithError(w, http.StatusBadRequest, "User already verified")
		return
	}

	// 古い認証コードを削除
	if err := h.verificationRepo.DeleteByUserID(user.ID); err != nil {
		h.logger.Warn("古い認証コード削除エラー", "error", err)
		// 続行
	}

	// 新しい認証コード生成（有効期限15分）
	verification, err := h.verificationRepo.Create(user.ID, 15*time.Minute)
	if err != nil {
		h.logger.Error("認証コード生成エラー", "error", err)
		RespondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	// メール送信
	if err := h.emailService.SendVerificationCode(r.Context(), user.Email, user.Name, verification.Code); err != nil {
		h.logger.Error("メール送信エラー", "error", err)
		RespondWithError(w, http.StatusInternalServerError, "Failed to send verification email")
		return
	}

	h.logger.Info("認証コード再送信成功", "user_id", user.ID, "email", req.Email)

	RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Verification code sent",
	})
}
