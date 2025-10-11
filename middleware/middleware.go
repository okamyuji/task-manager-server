package middleware

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"task_manager_server/models"
	"task_manager_server/utils"
)

var logger *slog.Logger

// SetLogger ロガーを設定
func SetLogger(l *slog.Logger) {
	logger = l
}

// LoggingMiddleware リクエスト/レスポンスをログに記録するミドルウェア
func LoggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// レスポンスラッパーでステータスコードをキャプチャ
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// リクエスト開始ログ
		logger.Debug("リクエスト開始",
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
			"user_agent", r.UserAgent(),
		)

		next(wrapped, r)

		duration := time.Since(start)

		// リクエスト完了ログ
		logLevel := slog.LevelInfo
		if wrapped.statusCode >= 500 {
			logLevel = slog.LevelError
		} else if wrapped.statusCode >= 400 {
			logLevel = slog.LevelWarn
		}

		logger.Log(r.Context(), logLevel, "リクエスト完了",
			"method", r.Method,
			"path", r.URL.Path,
			"status", wrapped.statusCode,
			"duration_ms", duration.Milliseconds(),
			"remote_addr", r.RemoteAddr,
		)
	}
}

// responseWriter ステータスコードをキャプチャするためのラッパー
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// CORSMiddleware CORSヘッダーを追加するミドルウェア（本番対応版）
func CORSMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// 許可されたオリジンを環境変数から取得
		allowedOrigins := getAllowedOrigins()

		// オリジンチェック
		if isOriginAllowed(origin, allowedOrigins) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		} else if len(allowedOrigins) == 1 && allowedOrigins[0] == "*" {
			// 開発環境: 全てのオリジンを許可
			w.Header().Set("Access-Control-Allow-Origin", "*")
		} else {
			logger.Warn("許可されていないオリジン", "origin", origin)
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Max-Age", "7200")

		// プリフライトリクエストの処理
		if r.Method == http.MethodOptions {
			logger.Debug("CORSプリフライトリクエスト",
				"origin", origin,
				"method", r.Header.Get("Access-Control-Request-Method"),
			)
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next(w, r)
	}
}

// getAllowedOrigins 許可されたオリジンを取得
func getAllowedOrigins() []string {
	originsEnv := os.Getenv("ALLOWED_ORIGINS")
	if originsEnv == "" {
		// デフォルト: 開発環境用（全て許可）
		return []string{"*"}
	}

	// カンマ区切りで分割
	var origins []string
	for _, origin := range strings.Split(originsEnv, ",") {
		trimmed := strings.TrimSpace(origin)
		if trimmed != "" {
			origins = append(origins, trimmed)
		}
	}

	return origins
}

// isOriginAllowed オリジンが許可されているかチェック
func isOriginAllowed(origin string, allowedOrigins []string) bool {
	for _, allowed := range allowedOrigins {
		if allowed == "*" || allowed == origin {
			return true
		}
	}
	return false
}

// AuthMiddleware JWT認証を行うミドルウェア
func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Authorizationヘッダーからトークンを取得
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			logger.Warn("認証ヘッダーが見つかりません",
				"path", r.URL.Path,
				"remote_addr", r.RemoteAddr,
			)
			respondWithError(w, http.StatusUnauthorized, "Authorization header required")
			return
		}

		// "Bearer "プレフィックスを削除
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			logger.Warn("無効な認証形式",
				"path", r.URL.Path,
				"remote_addr", r.RemoteAddr,
			)
			respondWithError(w, http.StatusUnauthorized, "Invalid authorization format")
			return
		}

		// トークン検証
		claims, err := utils.VerifyToken(tokenString)
		if err != nil {
			logger.Warn("トークン検証失敗",
				"error", err,
				"path", r.URL.Path,
				"remote_addr", r.RemoteAddr,
			)
			respondWithError(w, http.StatusUnauthorized, "Invalid or expired token")
			return
		}

		logger.Debug("認証成功",
			"user_id", claims.UserID,
			"email", claims.Email,
			"path", r.URL.Path,
		)

		// クレームをコンテキストに追加（簡易実装のためヘッダーに設定）
		r.Header.Set("X-User-ID", claims.UserID)
		r.Header.Set("X-User-Email", claims.Email)

		next(w, r)
	}
}

// respondWithJSON JSON形式でレスポンスを返す
func respondWithJSON(w http.ResponseWriter, statusCode int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		logger.Error("JSONエンコードエラー", "error", err)
	}
}

// respondWithError エラーレスポンスを返す
func respondWithError(w http.ResponseWriter, statusCode int, message string) {
	respondWithJSON(w, statusCode, models.ErrorResponse{Message: message})
}
