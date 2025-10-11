package handlers

import (
	"bytes"
	"context"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// mockStorageService
type mockStorageService struct{}

func (m *mockStorageService) Upload(ctx context.Context, file multipart.File, filename string) (string, error) {
	return "https://example.com/uploads/" + filename, nil
}

func (m *mockStorageService) Delete(ctx context.Context, filename string) error {
	return nil
}

func (m *mockStorageService) GetURL(filename string) string {
	return "https://example.com/uploads/" + filename
}

// TestUploadHandler_HandleUpload_正常系
func TestUploadHandler_HandleUpload_正常系(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	storage := &mockStorageService{}
	handler := NewUploadHandler(storage, logger)

	// マルチパートフォームデータを作成
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("image", "test.jpg")
	_, _ = part.Write([]byte("fake image data"))
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()

	handler.HandleUpload(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
}

// TestUploadHandler_HandleUpload_異常系_GETメソッド
func TestUploadHandler_HandleUpload_異常系_GETメソッド(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	storage := &mockStorageService{}
	handler := NewUploadHandler(storage, logger)

	req := httptest.NewRequest(http.MethodGet, "/upload", nil)
	rec := httptest.NewRecorder()

	handler.HandleUpload(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", rec.Code)
	}
}

// TestUploadHandler_HandleUpload_異常系_ファイルなし
func TestUploadHandler_HandleUpload_異常系_ファイルなし(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	storage := &mockStorageService{}
	handler := NewUploadHandler(storage, logger)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()

	handler.HandleUpload(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}
}
