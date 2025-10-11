package handlers

import (
	"encoding/json"
	"net/http"

	"task_manager_server/models"
)

// RespondWithJSON JSON形式でレスポンスを返す
func RespondWithJSON(w http.ResponseWriter, statusCode int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}

// RespondWithError エラーレスポンスを返す
func RespondWithError(w http.ResponseWriter, statusCode int, message string) {
	RespondWithJSON(w, statusCode, models.ErrorResponse{Message: message})
}
