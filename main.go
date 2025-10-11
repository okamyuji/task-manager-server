package main

import (
	"log/slog"
	"net/http"
	"os"

	"task_manager_server/database"
	"task_manager_server/database/repository"
	"task_manager_server/email"
	"task_manager_server/handlers"
	"task_manager_server/middleware"
	"task_manager_server/storage"
	"task_manager_server/utils"
)

func main() {
	// 構造化ロガーの初期化
	logLevel := slog.LevelInfo
	if os.Getenv("DEBUG") == "true" {
		logLevel = slog.LevelDebug
	}

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     logLevel,
		AddSource: true,
	})
	logger := slog.New(handler)
	slog.SetDefault(logger)

	// ロガーを各パッケージに設定
	utils.SetLogger(logger)
	middleware.SetLogger(logger)

	logger.Info("サーバーを初期化中...")

	// データベース初期化（PostgreSQL）
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		logger.Error("DATABASE_URLが設定されていません。環境変数を設定してください。")
		logger.Info("例: export DATABASE_URL=\"postgresql://user:password@localhost:5432/dbname?sslmode=disable\"")
		os.Exit(1)
	}

	db, err := database.NewDB(dbURL, logger)
	if err != nil {
		logger.Error("データベース接続失敗", "error", err)
		os.Exit(1)
	}
	defer func() {
		_ = db.Close()
	}()

	// マイグレーション実行
	if err := db.Migrate(); err != nil {
		logger.Error("マイグレーション失敗", "error", err)
		os.Exit(1)
	}

	// リポジトリ初期化
	userRepo := repository.NewUserRepository(db.DB)
	taskRepo := repository.NewTaskRepository(db.DB)
	verificationRepo := repository.NewVerificationRepository(db.DB)

	// メールサービス初期化
	emailService := email.NewEmailService(logger)

	// ストレージサービス初期化
	storageService := storage.NewStorageService(logger)

	// トークンサービス初期化
	tokenService := utils.NewTokenService(logger)

	// パスワードハッシャー初期化
	passwordHasher := utils.NewPasswordHasher()

	// レート制限初期化
	middleware.InitRateLimiters(logger)

	// ハンドラー初期化
	authHandler := handlers.NewAuthHandler(userRepo, verificationRepo, emailService, tokenService, passwordHasher, logger)
	taskHandler := handlers.NewTaskHandler(taskRepo, logger)
	verificationHandler := handlers.NewVerificationHandler(userRepo, verificationRepo, emailService, logger)
	uploadHandler := handlers.NewUploadHandler(storageService, logger)

	// ルーティング設定
	mux := http.NewServeMux()

	// 認証関連エンドポイント（レート制限あり）
	mux.HandleFunc("/auth/login", middleware.LoggingMiddleware(middleware.AuthRateLimiter.Middleware(middleware.CORSMiddleware(authHandler.HandleLogin))))
	mux.HandleFunc("/auth/register", middleware.LoggingMiddleware(middleware.AuthRateLimiter.Middleware(middleware.CORSMiddleware(authHandler.HandleRegister))))
	mux.HandleFunc("/auth/refresh", middleware.LoggingMiddleware(middleware.AuthRateLimiter.Middleware(middleware.CORSMiddleware(authHandler.HandleRefresh))))
	mux.HandleFunc("/auth/verify", middleware.LoggingMiddleware(middleware.AuthRateLimiter.Middleware(middleware.CORSMiddleware(verificationHandler.HandleVerify))))
	mux.HandleFunc("/auth/resend-code", middleware.LoggingMiddleware(middleware.AuthRateLimiter.Middleware(middleware.CORSMiddleware(verificationHandler.HandleResendCode))))

	// タスク関連エンドポイント（認証 + レート制限）
	mux.HandleFunc("/tasks", middleware.LoggingMiddleware(middleware.TaskRateLimiter.Middleware(middleware.AuthMiddleware(middleware.CORSMiddleware(taskHandler.HandleTasks)))))
	mux.HandleFunc("/tasks/", middleware.LoggingMiddleware(middleware.TaskRateLimiter.Middleware(middleware.AuthMiddleware(middleware.CORSMiddleware(taskHandler.HandleTaskByID)))))

	// 画像アップロードエンドポイント（認証 + レート制限）
	mux.HandleFunc("/upload", middleware.LoggingMiddleware(middleware.UploadRateLimiter.Middleware(middleware.AuthMiddleware(middleware.CORSMiddleware(uploadHandler.HandleUpload)))))

	// 静的ファイル配信（ローカルストレージ使用時のみ）
	storageType := os.Getenv("STORAGE_TYPE")
	if storageType == "" || storageType == "local" {
		// ローカルストレージの場合のみアップロードディレクトリを作成・配信
		uploadDir := os.Getenv("UPLOAD_DIR")
		if uploadDir == "" {
			uploadDir = "./uploads"
		}

		// ディレクトリ作成（ローカル環境のみ）
		if err := os.MkdirAll(uploadDir, 0755); err != nil {
			logger.Warn("アップロードディレクトリの作成をスキップ（読み取り専用FS）", "error", err)
		} else {
			logger.Info("アップロードディレクトリを作成しました", "path", uploadDir)
		}

		// 静的ファイル配信
		mux.Handle("/uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir(uploadDir))))
	} else {
		logger.Info("S3ストレージを使用するため、ローカルアップロードディレクトリは不要です")
	}

	// サーバー起動
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// 環境変数の確認
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		logger.Warn("JWT_SECRETが設定されていません。本番環境では必ず設定してください。")
	}

	allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
	if allowedOrigins == "" {
		logger.Warn("ALLOWED_ORIGINSが設定されていません。全てのオリジンを許可します（開発環境用）。")
	}

	logger.Info("サーバーを起動しました",
		"port", port,
		"database", "PostgreSQL",
		"jwt_secret_configured", jwtSecret != "",
		"cors_configured", allowedOrigins != "",
		"endpoints", []string{
			"POST   /auth/login",
			"POST   /auth/register",
			"POST   /auth/refresh",
			"POST   /auth/verify",
			"POST   /auth/resend-code",
			"GET    /tasks",
			"POST   /tasks",
			"GET    /tasks/{id}",
			"PUT    /tasks/{id}",
			"DELETE /tasks/{id}",
			"PATCH  /tasks/{id}/complete",
			"PATCH  /tasks/{id}/incomplete",
			"POST   /upload",
		},
	)

	if err := http.ListenAndServe(":"+port, mux); err != nil {
		logger.Error("サーバーの起動に失敗しました", "error", err)
		os.Exit(1)
	}
}
