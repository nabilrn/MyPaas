package user

import (
	"net/http"
	"net/mail"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"mypaas/internal/auth"
	"mypaas/internal/db"
	"mypaas/internal/errs"
	"mypaas/internal/httpx"
)

type Handler struct {
	queries *db.Queries
}

func NewHandler(queries *db.Queries) *Handler {
	return &Handler{queries: queries}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	users, err := h.queries.ListUsers(r.Context())
	if err != nil {
		httpx.DomainError(w, err)
		return
	}
	out := make([]auth.UserResponse, 0, len(users))
	for _, item := range users {
		out = append(out, auth.UserResponseFromDB(item))
	}
	httpx.JSON(w, http.StatusOK, out)
}

func (h *Handler) Add(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
		Role  string `json:"role"`
	}
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_JSON", "Request body must be valid JSON.", nil)
		return
	}
	if _, err := mail.ParseAddress(req.Email); err != nil {
		httpx.DomainError(w, errs.ErrValidation)
		return
	}
	if req.Role == "" {
		req.Role = "collaborator"
	}
	if req.Role != "owner" && req.Role != "collaborator" {
		httpx.DomainError(w, errs.ErrValidation)
		return
	}

	created, err := h.queries.CreateUser(r.Context(), db.CreateUserParams{
		Email: req.Email,
		Role:  req.Role,
	})
	if err != nil {
		httpx.DomainError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, auth.UserResponseFromDB(created))
}

func (h *Handler) Remove(w http.ResponseWriter, r *http.Request) {
	current, err := auth.CurrentUser(r)
	if err != nil {
		httpx.DomainError(w, err)
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.DomainError(w, errs.ErrNotFound)
		return
	}
	if id == current.ID {
		httpx.Error(w, http.StatusBadRequest, "CANNOT_REMOVE_SELF", "You cannot remove your own account.", nil)
		return
	}
	if err := h.queries.DeleteUser(r.Context(), id); err != nil {
		httpx.DomainError(w, err)
		return
	}
	httpx.NoContent(w)
}
