package httpx

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"mypaas/internal/errs"
)

type errorResponse struct {
	Error apiError `json:"error"`
}

type apiError struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

func JSON(w http.ResponseWriter, status int, data any) {
	writeJSON(w, status, map[string]any{"data": data})
}

func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

func Error(w http.ResponseWriter, status int, code, message string, details map[string]any) {
	writeJSON(w, status, errorResponse{
		Error: apiError{
			Code:    code,
			Message: message,
			Details: details,
		},
	})
}

func DomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errs.ErrUnauthorized):
		Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication is required.", nil)
	case errors.Is(err, errs.ErrForbidden):
		Error(w, http.StatusForbidden, "FORBIDDEN", "You do not have access to this resource.", nil)
	case errors.Is(err, errs.ErrEmailNotWhitelisted):
		Error(w, http.StatusForbidden, "EMAIL_NOT_WHITELISTED", "This GitHub email is not whitelisted.", nil)
	case errors.Is(err, errs.ErrNotFound):
		Error(w, http.StatusNotFound, "NOT_FOUND", "Resource not found.", nil)
	case errors.Is(err, errs.ErrProjectNameTaken):
		Error(w, http.StatusConflict, "PROJECT_NAME_TAKEN", "Project name is already taken.", nil)
	case errors.Is(err, errs.ErrUserAlreadyExists):
		Error(w, http.StatusConflict, "USER_ALREADY_EXISTS", "User is already whitelisted.", nil)
	case errors.Is(err, errs.ErrPortPoolExhausted):
		Error(w, http.StatusConflict, "PORT_POOL_EXHAUSTED", "No available internal port remains.", nil)
	case errors.Is(err, errs.ErrQuotaExceeded):
		Error(w, http.StatusConflict, "QUOTA_EXCEEDED", err.Error(), nil)
	case errors.Is(err, errs.ErrComposeFileNotFound):
		Error(w, http.StatusBadRequest, "COMPOSE_FILE_NOT_FOUND", "Compose file was not found in the repository.", nil)
	case errors.Is(err, errs.ErrComposeUnsupported):
		Error(w, http.StatusBadRequest, "COMPOSE_UNSUPPORTED", "This action is not supported for Compose projects yet.", nil)
	case errors.Is(err, errs.ErrDockerfileNotFound):
		Error(w, http.StatusBadRequest, "DOCKERFILE_NOT_FOUND", "Dockerfile was not found in the repository root.", nil)
	case errors.Is(err, errs.ErrNoDeployConfig):
		message := strings.TrimPrefix(err.Error(), errs.ErrNoDeployConfig.Error()+": ")
		if message == err.Error() {
			message = "No Dockerfile, Compose file, or static site was found in the selected branch repository root."
		}
		Error(w, http.StatusBadRequest, "NO_DEPLOY_CONFIG", message, nil)
	case errors.Is(err, errs.ErrValidation), errors.Is(err, errs.ErrBadRequest):
		Error(w, http.StatusBadRequest, "VALIDATION_FAILED", err.Error(), nil)
	default:
		slog.Error("unhandled request error", "error", err)
		Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Unexpected server error.", nil)
	}
}

func DecodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.Warn("write json response", "error", err)
	}
}
