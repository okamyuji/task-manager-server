package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// StorageService ファイルストレージサービス
type StorageService interface {
	Upload(ctx context.Context, file multipart.File, filename string) (string, error)
	Delete(ctx context.Context, filename string) error
	GetURL(filename string) string
}

// LocalStorage ローカルファイルシステムストレージ
type LocalStorage struct {
	uploadDir string
	baseURL   string
	logger    *slog.Logger
}

// NewLocalStorage ローカルストレージを作成
func NewLocalStorage(uploadDir, baseURL string, logger *slog.Logger) *LocalStorage {
	// アップロードディレクトリを作成（読み取り専用FSの場合は警告のみ）
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		logger.Warn("アップロードディレクトリの作成をスキップ", "error", err, "dir", uploadDir)
	} else {
		logger.Info("アップロードディレクトリを確認", "dir", uploadDir)
	}

	return &LocalStorage{
		uploadDir: uploadDir,
		baseURL:   baseURL,
		logger:    logger,
	}
}

// Upload ファイルをローカルに保存
func (s *LocalStorage) Upload(ctx context.Context, file multipart.File, filename string) (string, error) {
	filePath := filepath.Join(s.uploadDir, filename)

	// ファイルを作成
	dst, err := os.Create(filePath)
	if err != nil {
		s.logger.Error("ファイル作成エラー", "error", err, "path", filePath)
		return "", fmt.Errorf("ファイル作成エラー: %w", err)
	}
	defer func() {
		_ = dst.Close()
	}()

	// ファイルをコピー
	if _, err := io.Copy(dst, file); err != nil {
		s.logger.Error("ファイル書き込みエラー", "error", err)
		return "", fmt.Errorf("ファイル書き込みエラー: %w", err)
	}

	url := s.GetURL(filename)
	s.logger.Info("ファイルアップロード成功（ローカル）", "filename", filename, "url", url)
	return url, nil
}

// Delete ローカルファイルを削除
func (s *LocalStorage) Delete(ctx context.Context, filename string) error {
	filePath := filepath.Join(s.uploadDir, filename)

	if err := os.Remove(filePath); err != nil {
		if os.IsNotExist(err) {
			return nil // ファイルが存在しない場合はエラーにしない
		}
		s.logger.Error("ファイル削除エラー", "error", err, "path", filePath)
		return fmt.Errorf("ファイル削除エラー: %w", err)
	}

	s.logger.Info("ファイル削除成功（ローカル）", "filename", filename)
	return nil
}

// GetURL ファイルのURLを取得
func (s *LocalStorage) GetURL(filename string) string {
	return fmt.Sprintf("%s/uploads/%s", s.baseURL, filename)
}

// S3Storage S3互換オブジェクトストレージ
type S3Storage struct {
	client *s3.Client
	bucket string
	cdnURL string
	logger *slog.Logger
}

// NewS3Storage S3ストレージを作成
func NewS3Storage(endpoint, region, bucket, accessKey, secretKey, cdnURL string, logger *slog.Logger) *S3Storage {
	cfg := aws.Config{
		Region:      region,
		Credentials: credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
	}

	if endpoint != "" {
		cfg.BaseEndpoint = aws.String(endpoint)
	}

	client := s3.NewFromConfig(cfg)

	logger.Info("S3ストレージを初期化",
		"endpoint", endpoint,
		"region", region,
		"bucket", bucket,
		"cdn_url", cdnURL,
	)

	return &S3Storage{
		client: client,
		bucket: bucket,
		cdnURL: cdnURL,
		logger: logger,
	}
}

// Upload ファイルをS3にアップロード
func (s *S3Storage) Upload(ctx context.Context, file multipart.File, filename string) (string, error) {
	// ファイル内容を読み込む
	data, err := io.ReadAll(file)
	if err != nil {
		s.logger.Error("ファイル読み込みエラー", "error", err)
		return "", fmt.Errorf("ファイル読み込みエラー: %w", err)
	}

	// Content-Typeを推測
	contentType := "application/octet-stream"
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".jpg", ".jpeg":
		contentType = "image/jpeg"
	case ".png":
		contentType = "image/png"
	case ".gif":
		contentType = "image/gif"
	case ".webp":
		contentType = "image/webp"
	}

	// S3にアップロード
	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(filename),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		s.logger.Error("S3アップロードエラー", "error", err, "filename", filename)
		return "", fmt.Errorf("S3アップロードエラー: %w", err)
	}

	url := s.GetURL(filename)
	s.logger.Info("ファイルアップロード成功（S3）", "filename", filename, "url", url)
	return url, nil
}

// Delete S3からファイルを削除
func (s *S3Storage) Delete(ctx context.Context, filename string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(filename),
	})
	if err != nil {
		s.logger.Error("S3削除エラー", "error", err, "filename", filename)
		return fmt.Errorf("S3削除エラー: %w", err)
	}

	s.logger.Info("ファイル削除成功（S3）", "filename", filename)
	return nil
}

// GetURL ファイルのURLを取得
func (s *S3Storage) GetURL(filename string) string {
	// CDN URLが設定されている場合はそれを使用
	if s.cdnURL != "" {
		return fmt.Sprintf("%s/%s", strings.TrimSuffix(s.cdnURL, "/"), filename)
	}
	// フォールバック: S3直接URL（実際には使わない）
	return fmt.Sprintf("https://%s.s3.amazonaws.com/%s", s.bucket, filename)
}

// NewStorageService 環境変数に応じて適切なストレージサービスを作成
func NewStorageService(logger *slog.Logger) StorageService {
	storageType := os.Getenv("STORAGE_TYPE")
	if storageType == "" {
		storageType = "local" // デフォルトはローカル
	}

	if storageType == "s3" {
		endpoint := os.Getenv("S3_ENDPOINT")
		region := os.Getenv("S3_REGION")
		bucket := os.Getenv("S3_BUCKET")
		accessKey := os.Getenv("S3_ACCESS_KEY")
		secretKey := os.Getenv("S3_SECRET_KEY")
		cdnURL := os.Getenv("S3_CDN_URL")

		if endpoint == "" || region == "" || bucket == "" || accessKey == "" || secretKey == "" {
			logger.Error("S3設定が不完全です。ローカルストレージにフォールバックします。")
			// フォールバック: ローカルストレージを使用
		} else {
			logger.Info("ストレージ: S3を使用します", "bucket", bucket)
			return NewS3Storage(endpoint, region, bucket, accessKey, secretKey, cdnURL, logger)
		}
	}

	// ローカルストレージ
	uploadDir := os.Getenv("UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = "./uploads"
	}

	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		port := os.Getenv("PORT")
		if port == "" {
			port = "8080"
		}
		baseURL = fmt.Sprintf("http://localhost:%s", port)
	}

	logger.Info("ストレージ: ローカルファイルシステムを使用します", "upload_dir", uploadDir)
	return NewLocalStorage(uploadDir, baseURL, logger)
}
