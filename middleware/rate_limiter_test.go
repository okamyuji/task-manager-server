package middleware

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

// TestNewRateLimiter_正常系
func TestNewRateLimiter_正常系(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	limiter := NewRateLimiter(100, 200, logger)

	if limiter == nil {
		t.Fatal("NewRateLimiter() returned nil")
	}
}

// TestRateLimiter_制限内リクエスト
func TestRateLimiter_制限内リクエスト(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	limiter := NewRateLimiter(100, 200, logger)

	// ダミーハンドラー
	called := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	// レートリミッターを適用
	limitedHandler := limiter.Middleware(handler)

	// リクエスト作成
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.1:1234"
	rec := httptest.NewRecorder()

	// リクエスト実行
	limitedHandler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
	if !called {
		t.Error("Handler was not called")
	}
}

// TestRateLimiter_制限超過
func TestRateLimiter_制限超過(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	limiter := NewRateLimiter(10, 10, logger) // 低い制限

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	limitedHandler := limiter.Middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.2:1234"

	// 制限を超えるまでリクエスト送信（10回/秒 + burst 10）
	successCount := 0
	limitExceeded := false

	for i := 0; i < 25; i++ {
		rec := httptest.NewRecorder()
		limitedHandler.ServeHTTP(rec, req)

		if rec.Code == http.StatusOK {
			successCount++
		} else if rec.Code == http.StatusTooManyRequests {
			limitExceeded = true
			break
		}
	}

	if !limitExceeded {
		t.Errorf("Expected rate limit to be exceeded, but got %d successful requests", successCount)
	}
}

// TestRateLimiter_異なるIP
func TestRateLimiter_異なるIP(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	limiter := NewRateLimiter(100, 200, logger)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	limitedHandler := limiter.Middleware(handler)

	// IP1からのリクエスト
	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req1.RemoteAddr = "192.168.1.10:1234"
	rec1 := httptest.NewRecorder()
	limitedHandler.ServeHTTP(rec1, req1)

	// IP2からのリクエスト
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req2.RemoteAddr = "192.168.1.20:1234"
	rec2 := httptest.NewRecorder()
	limitedHandler.ServeHTTP(rec2, req2)

	// 両方とも成功することを確認
	if rec1.Code != http.StatusOK || rec2.Code != http.StatusOK {
		t.Error("Both requests from different IPs should succeed")
	}
}

// TestRateLimiter_低い制限値
func TestRateLimiter_低い制限値(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	limiter := NewRateLimiter(1, 1, logger) // 非常に低い制限

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	limitedHandler := limiter.Middleware(handler)

	req := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
	req.RemoteAddr = "192.168.1.3:1234"
	rec := httptest.NewRecorder()

	limitedHandler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
}

// TestRateLimiter_クリーンアップ
func TestRateLimiter_クリーンアップ(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping cleanup test in short mode")
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	limiter := NewRateLimiter(100, 200, logger)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	limitedHandler := limiter.Middleware(handler)

	// リクエスト送信
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.4:1234"
	rec := httptest.NewRecorder()
	limitedHandler.ServeHTTP(rec, req)

	// 初期リミッター数を確認（1つ以上存在）
	initialCount := len(limiter.visitors)
	if initialCount == 0 {
		t.Error("Expected at least one limiter to exist")
	}

	// クリーンアップが実行されるまで待機（cleanupIntervalは5分だが、テストでは確認のみ）
	time.Sleep(100 * time.Millisecond)

	// リミッターが存在することを確認（クリーンアップは5分後なのでまだ存在）
	currentCount := len(limiter.visitors)
	if currentCount == 0 {
		t.Error("Limiters should still exist shortly after creation")
	}
}
