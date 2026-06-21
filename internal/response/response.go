package response

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	apperrors "github.com/pavelc4/astra/internal/errors"
)

type ErrorDetail struct {
	Code   apperrors.ErrorCode `json:"code"`
	Detail string              `json:"detail"`
}

type APIResponse struct {
	Status  int          `json:"status"`
	Success bool         `json:"success"`
	Message string       `json:"message"`
	Data    any          `json:"data,omitempty"`
	Error   *ErrorDetail `json:"error,omitempty"`
}

func OK(w http.ResponseWriter, data any, msg string) {
	writeJSON(w, http.StatusOK, APIResponse{
		Status:  http.StatusOK,
		Success: true,
		Message: msg,
		Data:    data,
	})
}

func Fail(w http.ResponseWriter, status int, msg string, code apperrors.ErrorCode, detail string) {
	writeJSON(w, status, APIResponse{
		Status:  status,
		Success: false,
		Message: msg,
		Error:   &ErrorDetail{Code: code, Detail: detail},
	})
}

func HandleError(w http.ResponseWriter, err error) {
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		Fail(w, appErr.Status, appErr.Message, appErr.Code, appErr.Message)
		return
	}
	Fail(w, http.StatusInternalServerError, "Internal server error", apperrors.CodeInternalError, err.Error())
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("Failed to encode JSON response", "error", err)
	}
}
