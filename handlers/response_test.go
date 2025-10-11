package handlers

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"task_manager_server/models"
)

// TestRespondWithJSON_正常系
func TestRespondWithJSON_正常系(t *testing.T) {
	rec := httptest.NewRecorder()
	payload := map[string]string{"message": "success"}

	RespondWithJSON(rec, 200, payload)

	if rec.Code != 200 {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	var result map[string]string
	_ = json.NewDecoder(rec.Body).Decode(&result)
	if result["message"] != "success" {
		t.Errorf("Expected message 'success', got '%s'", result["message"])
	}
}

// TestRespondWithError_正常系
func TestRespondWithError_正常系(t *testing.T) {
	rec := httptest.NewRecorder()

	RespondWithError(rec, 400, "Bad request")

	if rec.Code != 400 {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}

	var result models.ErrorResponse
	_ = json.NewDecoder(rec.Body).Decode(&result)
	if result.Message != "Bad request" {
		t.Errorf("Expected message 'Bad request', got '%s'", result.Message)
	}
}

// TestRespondWithJSON_複雑な構造体
func TestRespondWithJSON_複雑な構造体(t *testing.T) {
	rec := httptest.NewRecorder()
	payload := models.AuthResponse{
		AccessToken:  "access_token",
		RefreshToken: "refresh_token",
		UserID:       "user123",
	}

	RespondWithJSON(rec, 200, payload)

	if rec.Code != 200 {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var result models.AuthResponse
	_ = json.NewDecoder(rec.Body).Decode(&result)
	if result.UserID != "user123" {
		t.Errorf("Expected UserID 'user123', got '%s'", result.UserID)
	}
}
