package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"task_manager_server/models"
	"task_manager_server/storage"

	"github.com/google/uuid"
)

// UploadHandler 画像アップロード関連のハンドラー
type UploadHandler struct {
	storage storage.StorageService
	logger  *slog.Logger
}

// NewUploadHandler アップロードハンドラーを作成
func NewUploadHandler(storage storage.StorageService, logger *slog.Logger) *UploadHandler {
	return &UploadHandler{
		storage: storage,
		logger:  logger,
	}
}

// HandleUpload 画像アップロード処理
func (h *UploadHandler) HandleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.logger.Warn("不正なHTTPメソッド", "method", r.Method, "path", r.URL.Path)
		RespondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	h.logger.Info("画像アップロード開始", "remote_addr", r.RemoteAddr)

	// マルチパートフォームをパース（最大10MB）
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		h.logger.Warn("マルチパートフォームのパースエラー", "error", err)
		RespondWithError(w, http.StatusBadRequest, "Failed to parse multipart form")
		return
	}

	// 画像ファイルを取得
	file, header, err := r.FormFile("image")
	if err != nil {
		h.logger.Warn("画像ファイル取得エラー", "error", err)
		RespondWithError(w, http.StatusBadRequest, "Failed to get image file")
		return
	}
	defer func() {
		if err := file.Close(); err != nil {
			h.logger.Error("ファイルクローズエラー", "error", err)
		}
	}()

	h.logger.Debug("画像ファイル受信",
		"filename", header.Filename,
		"size_bytes", header.Size,
		"content_type", header.Header.Get("Content-Type"),
	)

	// 許可されたファイルタイプのバリデーション
	allowedExtensions := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".gif":  true,
		".webp": true,
	}

	extLower := strings.ToLower(filepath.Ext(header.Filename))
	if extLower == "" {
		h.logger.Warn("ファイル拡張子がありません", "filename", header.Filename)
		RespondWithError(w, http.StatusBadRequest, "File must have a valid image extension (.jpg, .jpeg, .png, .gif, .webp)")
		return
	}

	if !allowedExtensions[extLower] {
		h.logger.Warn("許可されていないファイルタイプ", "filename", header.Filename, "ext", extLower)
		RespondWithError(w, http.StatusBadRequest, "Only image files are allowed (.jpg, .jpeg, .png, .gif, .webp)")
		return
	}

	// ユニークなファイル名を生成
	filename := fmt.Sprintf("%d_%s%s", time.Now().Unix(), uuid.New().String(), extLower)

	// ストレージサービスを使用してアップロード
	url, err := h.storage.Upload(r.Context(), file, filename)
	if err != nil {
		h.logger.Error("ファイルアップロードエラー", "error", err, "filename", filename)
		RespondWithError(w, http.StatusInternalServerError, "Failed to upload file")
		return
	}

	h.logger.Info("画像アップロード完了", "url", url, "filename", filename)

	// レスポンス
	response := models.UploadResponse{
		URL: url,
	}

	RespondWithJSON(w, http.StatusOK, response)
}
