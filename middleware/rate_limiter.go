package middleware

import (
	"log/slog"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimiter レート制限ミドルウェア
type RateLimiter struct {
	visitors map[string]*rate.Limiter
	mu       sync.RWMutex
	r        rate.Limit
	b        int
	logger   *slog.Logger
}

// NewRateLimiter 新しいレート制限を作成
// r: 秒間リクエスト数、b: バーストサイズ
func NewRateLimiter(r rate.Limit, b int, logger *slog.Logger) *RateLimiter {
	limiter := &RateLimiter{
		visitors: make(map[string]*rate.Limiter),
		r:        r,
		b:        b,
		logger:   logger,
	}

	// 定期的に古いエントリをクリーンアップ（5分ごと）
	go limiter.cleanupVisitors()

	return limiter
}

// getVisitor IPアドレスに対応するリミッターを取得
func (rl *RateLimiter) getVisitor(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.visitors[ip]
	if !exists {
		limiter = rate.NewLimiter(rl.r, rl.b)
		rl.visitors[ip] = limiter
	}

	return limiter
}

// Middleware レート制限ミドルウェア
func (rl *RateLimiter) Middleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// IPアドレス取得（X-Forwarded-For対応）
		ip := getClientIP(r)

		limiter := rl.getVisitor(ip)
		if !limiter.Allow() {
			rl.logger.Warn("レート制限超過",
				"ip", ip,
				"path", r.URL.Path,
				"method", r.Method,
			)
			http.Error(w, "Too many requests", http.StatusTooManyRequests)
			return
		}

		next(w, r)
	}
}

// cleanupVisitors 古いエントリをクリーンアップ
func (rl *RateLimiter) cleanupVisitors() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		// 全てクリア（簡易実装）
		rl.visitors = make(map[string]*rate.Limiter)
		rl.mu.Unlock()

		rl.logger.Debug("レート制限キャッシュをクリアしました")
	}
}

// getClientIP クライアントのIPアドレスを取得
func getClientIP(r *http.Request) string {
	// X-Forwarded-Forヘッダーをチェック（リバースプロキシ対応）
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		// 最初のIPを使用
		return forwarded
	}

	// X-Real-IPヘッダーをチェック
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}

	// RemoteAddrを使用
	return r.RemoteAddr
}

// 事前定義されたレート制限
var (
	// 認証エンドポイント: 5 req/min per IP
	AuthRateLimiter *RateLimiter

	// タスクAPI: 60 req/min per IP（ユーザーベースにも対応可能）
	TaskRateLimiter *RateLimiter

	// アップロード: 10 req/min per IP
	UploadRateLimiter *RateLimiter
)

// InitRateLimiters レート制限を初期化
func InitRateLimiters(logger *slog.Logger) {
	// 認証: 5リクエスト/分、バースト10
	AuthRateLimiter = NewRateLimiter(rate.Limit(5.0/60.0), 10, logger)

	// タスク: 60リクエスト/分（= 1req/sec）、バースト100
	TaskRateLimiter = NewRateLimiter(rate.Limit(1.0), 100, logger)

	// アップロード: 10リクエスト/分、バースト15
	UploadRateLimiter = NewRateLimiter(rate.Limit(10.0/60.0), 15, logger)

	logger.Info("レート制限を初期化しました",
		"auth_rate", "5/min",
		"task_rate", "60/min",
		"upload_rate", "10/min",
	)
}
