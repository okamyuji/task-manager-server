package middleware

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"task_manager_server/utils"
	"testing"
)

// TestLoggingMiddleware_正常系
func TestLoggingMiddleware_正常系(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	SetLogger(logger)

	called := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	middleware := LoggingMiddleware(handler)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	middleware.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
	if !called {
		t.Error("Handler was not called")
	}
}

// TestCORSMiddleware_許可されたオリジン
func TestCORSMiddleware_許可されたオリジン(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	SetLogger(logger)

	_ = os.Setenv("ALLOWED_ORIGINS", "https://example.com,https://test.com")
	defer os.Clearenv()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := CORSMiddleware(handler)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()

	middleware.ServeHTTP(rec, req)

	origin := rec.Header().Get("Access-Control-Allow-Origin")
	if origin != "https://example.com" {
		t.Errorf("Expected origin https://example.com, got %s", origin)
	}
}

// TestCORSMiddleware_不許可オリジン
func TestCORSMiddleware_不許可オリジン(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	SetLogger(logger)

	_ = os.Setenv("ALLOWED_ORIGINS", "https://example.com")
	defer os.Clearenv()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := CORSMiddleware(handler)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://evil.com")
	rec := httptest.NewRecorder()

	middleware.ServeHTTP(rec, req)

	origin := rec.Header().Get("Access-Control-Allow-Origin")
	if origin == "https://evil.com" {
		t.Error("Evil origin should not be allowed")
	}
}

// TestCORSMiddleware_プリフライトリクエスト
func TestCORSMiddleware_プリフライトリクエスト(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	SetLogger(logger)

	called := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	middleware := CORSMiddleware(handler)
	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()

	middleware.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", rec.Code)
	}
	if called {
		t.Error("Handler should not be called for OPTIONS request")
	}
}

// TestAuthMiddleware_正常系
func TestAuthMiddleware_正常系(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	SetLogger(logger)

	// 有効なトークンを生成
	token, err := generateTestToken("user123", "test@example.com")
	if err != nil {
		t.Fatalf("Failed to generate test token: %v", err)
	}

	called := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		userID := r.Header.Get("X-User-ID")
		if userID != "user123" {
			t.Errorf("Expected user ID user123, got %s", userID)
		}
		w.WriteHeader(http.StatusOK)
	})

	middleware := AuthMiddleware(handler)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	middleware.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
	if !called {
		t.Error("Handler was not called")
	}
}

// TestAuthMiddleware_異常系_トークンなし
func TestAuthMiddleware_異常系_トークンなし(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	SetLogger(logger)

	called := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	middleware := AuthMiddleware(handler)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	middleware.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rec.Code)
	}
	if called {
		t.Error("Handler should not be called without token")
	}
}

// TestAuthMiddleware_異常系_無効なトークン
func TestAuthMiddleware_異常系_無効なトークン(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	SetLogger(logger)

	called := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	middleware := AuthMiddleware(handler)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.here")
	rec := httptest.NewRecorder()

	middleware.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rec.Code)
	}
	if called {
		t.Error("Handler should not be called with invalid token")
	}
}

// TestGetAllowedOrigins
func TestGetAllowedOrigins(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected []string
	}{
		{
			name:     "デフォルト（環境変数なし）",
			envValue: "",
			expected: []string{"*"},
		},
		{
			name:     "単一オリジン",
			envValue: "https://example.com",
			expected: []string{"https://example.com"},
		},
		{
			name:     "複数オリジン",
			envValue: "https://example.com,https://test.com",
			expected: []string{"https://example.com", "https://test.com"},
		},
		{
			name:     "スペース付き",
			envValue: "https://example.com , https://test.com ",
			expected: []string{"https://example.com", "https://test.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Clearenv()
			if tt.envValue != "" {
				_ = os.Setenv("ALLOWED_ORIGINS", tt.envValue)
			}

			result := getAllowedOrigins()

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d origins, got %d", len(tt.expected), len(result))
				return
			}

			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("Expected origin[%d] = %s, got %s", i, expected, result[i])
				}
			}
		})
	}
}

// TestIsOriginAllowed
func TestIsOriginAllowed(t *testing.T) {
	tests := []struct {
		name           string
		origin         string
		allowedOrigins []string
		expected       bool
	}{
		{
			name:           "ワイルドカード",
			origin:         "https://anywhere.com",
			allowedOrigins: []string{"*"},
			expected:       true,
		},
		{
			name:           "完全一致",
			origin:         "https://example.com",
			allowedOrigins: []string{"https://example.com"},
			expected:       true,
		},
		{
			name:           "不一致",
			origin:         "https://evil.com",
			allowedOrigins: []string{"https://example.com"},
			expected:       false,
		},
		{
			name:           "複数のうちの1つ",
			origin:         "https://test.com",
			allowedOrigins: []string{"https://example.com", "https://test.com"},
			expected:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isOriginAllowed(tt.origin, tt.allowedOrigins)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// generateTestToken テスト用トークン生成ヘルパー
func generateTestToken(userID, email string) (string, error) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	utils.SetLogger(logger)
	return utils.GenerateAccessToken(userID, email)
}
