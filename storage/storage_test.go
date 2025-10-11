package storage

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLocalStorage_Upload_正常系
func TestLocalStorage_Upload_正常系(t *testing.T) {
	// テスト用一時ディレクトリ
	tmpDir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	storage := NewLocalStorage(tmpDir, "http://localhost:8080", logger)

	// テスト用ファイルを作成
	content := []byte("test file content")
	filename := "test.txt"

	// multipart.Fileをモック
	mockFile := &mockMultipartFile{
		Reader: bytes.NewReader(content),
	}

	url, err := storage.Upload(context.Background(), mockFile, filename)
	if err != nil {
		t.Fatalf("Upload() error = %v", err)
	}

	// URLが正しいことを確認
	expectedURL := "http://localhost:8080/uploads/test.txt"
	if url != expectedURL {
		t.Errorf("Upload() url = %s, want %s", url, expectedURL)
	}

	// ファイルが実際に作成されていることを確認
	savedPath := filepath.Join(tmpDir, filename)
	savedContent, err := os.ReadFile(savedPath)
	if err != nil {
		t.Fatalf("ファイル読み込みエラー: %v", err)
	}

	if !bytes.Equal(savedContent, content) {
		t.Errorf("保存されたファイルの内容が一致しません: got %s, want %s", savedContent, content)
	}
}

// TestLocalStorage_Delete_正常系
func TestLocalStorage_Delete_正常系(t *testing.T) {
	tmpDir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	storage := NewLocalStorage(tmpDir, "http://localhost:8080", logger)

	// テスト用ファイルを作成
	filename := "test.txt"
	filePath := filepath.Join(tmpDir, filename)
	err := os.WriteFile(filePath, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("テストファイル作成エラー: %v", err)
	}

	// 削除実行
	err = storage.Delete(context.Background(), filename)
	if err != nil {
		t.Errorf("Delete() error = %v", err)
	}

	// ファイルが削除されていることを確認
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Error("Delete() ファイルが削除されていません")
	}
}

// TestLocalStorage_Delete_存在しないファイル
func TestLocalStorage_Delete_存在しないファイル(t *testing.T) {
	tmpDir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	storage := NewLocalStorage(tmpDir, "http://localhost:8080", logger)

	// 存在しないファイルの削除（エラーにならないことを確認）
	err := storage.Delete(context.Background(), "nonexistent.txt")
	if err != nil {
		t.Errorf("Delete() 存在しないファイルでエラー: %v", err)
	}
}

// TestLocalStorage_GetURL
func TestLocalStorage_GetURL(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	storage := NewLocalStorage("./uploads", "http://example.com", logger)

	url := storage.GetURL("test.jpg")
	expected := "http://example.com/uploads/test.jpg"

	if url != expected {
		t.Errorf("GetURL() = %s, want %s", url, expected)
	}
}

// TestNewStorageService_ローカルストレージ
func TestNewStorageService_ローカルストレージ(t *testing.T) {
	// 環境変数をクリア
	_ = os.Unsetenv("STORAGE_TYPE")
	_ = os.Unsetenv("S3_ENDPOINT")

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	service := NewStorageService(logger)

	// LocalStorageが返されることを確認
	_, ok := service.(*LocalStorage)
	if !ok {
		t.Error("NewStorageService() デフォルトでLocalStorageが返されませんでした")
	}
}

// TestNewStorageService_S3設定不完全でローカルにフォールバック
func TestNewStorageService_S3設定不完全でローカルにフォールバック(t *testing.T) {
	// 不完全なS3設定
	_ = os.Setenv("STORAGE_TYPE", "s3")
	_ = os.Setenv("S3_ENDPOINT", "https://example.com")
	// 他の設定なし
	defer os.Clearenv()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	service := NewStorageService(logger)

	// LocalStorageにフォールバックすることを確認
	_, ok := service.(*LocalStorage)
	if !ok {
		t.Error("NewStorageService() 不完全なS3設定でLocalStorageにフォールバックしませんでした")
	}
}

// TestS3Storage_GetURL_CDN使用
func TestS3Storage_GetURL_CDN使用(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	storage := NewS3Storage(
		"https://s3.example.com",
		"us-east-1",
		"test-bucket",
		"access-key",
		"secret-key",
		"https://cdn.example.com",
		logger,
	)

	url := storage.GetURL("test.jpg")
	expected := "https://cdn.example.com/test.jpg"

	if url != expected {
		t.Errorf("GetURL() = %s, want %s", url, expected)
	}
}

// TestS3Storage_GetURL_CDNなし
func TestS3Storage_GetURL_CDNなし(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	storage := NewS3Storage(
		"https://s3.example.com",
		"us-east-1",
		"test-bucket",
		"access-key",
		"secret-key",
		"", // CDN URL なし
		logger,
	)

	url := storage.GetURL("test.jpg")

	// S3直接URLが生成されることを確認
	if !strings.Contains(url, "test-bucket") || !strings.Contains(url, "test.jpg") {
		t.Errorf("GetURL() = %s, バケット名とファイル名が含まれていません", url)
	}
}

// mockMultipartFile multipart.Fileのモック
type mockMultipartFile struct {
	*bytes.Reader
}

func (m *mockMultipartFile) Close() error {
	return nil
}

func (m *mockMultipartFile) ReadAt(p []byte, off int64) (n int, err error) {
	return 0, io.EOF
}

func (m *mockMultipartFile) Seek(offset int64, whence int) (int64, error) {
	return m.Reader.Seek(offset, whence)
}
