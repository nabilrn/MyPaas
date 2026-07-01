package webhook

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"mypaas/internal/db"
	"mypaas/internal/errs"
	"mypaas/internal/httpx"
)

const maxPayloadBytes = 1 << 20

type Deployer interface {
	TriggerWebhook(ctx context.Context, projectID uuid.UUID) (db.Deployment, error)
}

type Handler struct {
	queries *db.Queries
	deploys Deployer
	limiter *rateLimiter
}

type pushPayload struct {
	Ref string `json:"ref"`
}

func NewHandler(queries *db.Queries, deploys Deployer) *Handler {
	return &Handler{
		queries: queries,
		deploys: deploys,
		limiter: newRateLimiter(10, time.Minute, time.Now),
	}
}

func (h *Handler) GitHub(w http.ResponseWriter, r *http.Request) {
	projectID, err := uuid.Parse(chi.URLParam(r, "projectId"))
	if err != nil {
		httpx.DomainError(w, errs.ErrNotFound)
		return
	}

	project, err := h.queries.GetProjectByID(r.Context(), projectID)
	if err == pgx.ErrNoRows {
		httpx.DomainError(w, errs.ErrNotFound)
		return
	}
	if err != nil {
		httpx.DomainError(w, err)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, maxPayloadBytes+1))
	if err != nil {
		httpx.DomainError(w, fmt.Errorf("read webhook payload: %w", err))
		return
	}
	if len(body) > maxPayloadBytes {
		httpx.Error(w, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "Webhook payload is too large.", nil)
		return
	}

	eventType := r.Header.Get("X-GitHub-Event")
	deliveryID := r.Header.Get("X-GitHub-Delivery")
	signatureValid := verifySignature(project.WebhookSecret, body, r.Header.Get("X-Hub-Signature-256"))
	if !signatureValid {
		h.logDelivery(r.Context(), project.ID, deliveryID, eventType, nil, false, false, uuid.UUID{})
		httpx.Error(w, http.StatusUnauthorized, "INVALID_WEBHOOK_SIGNATURE", "Webhook signature is invalid.", nil)
		return
	}

	if !h.limiter.allow(project.ID) {
		h.logDelivery(r.Context(), project.ID, deliveryID, eventType, nil, true, false, uuid.UUID{})
		httpx.Error(w, http.StatusTooManyRequests, "WEBHOOK_RATE_LIMITED", "Too many webhook deliveries for this project.", nil)
		return
	}

	if eventType != "push" {
		h.logDelivery(r.Context(), project.ID, deliveryID, eventType, nil, true, false, uuid.UUID{})
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ignored", "reason": "unsupported_event"})
		return
	}

	var payload pushPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		h.logDelivery(r.Context(), project.ID, deliveryID, eventType, nil, true, false, uuid.UUID{})
		httpx.Error(w, http.StatusBadRequest, "INVALID_WEBHOOK_PAYLOAD", "Webhook payload must be valid GitHub push JSON.", nil)
		return
	}

	branch := branchFromRef(payload.Ref)
	if branch == "" || branch != project.Branch {
		h.logDelivery(r.Context(), project.ID, deliveryID, eventType, stringPtr(branch), true, false, uuid.UUID{})
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ignored", "reason": "branch_mismatch"})
		return
	}

	deployment, err := h.deploys.TriggerWebhook(r.Context(), project.ID)
	if err != nil {
		h.logDelivery(r.Context(), project.ID, deliveryID, eventType, &branch, true, false, uuid.UUID{})
		httpx.DomainError(w, err)
		return
	}

	h.logDelivery(r.Context(), project.ID, deliveryID, eventType, &branch, true, true, deployment.ID)
	httpx.JSON(w, http.StatusAccepted, map[string]any{
		"status":       "queued",
		"deploymentId": deployment.ID.String(),
	})
}

func (h *Handler) logDelivery(ctx context.Context, projectID uuid.UUID, deliveryID, eventType string, branch *string, signatureValid, processed bool, deploymentID uuid.UUID) {
	var deployment pgtype.UUID
	if deploymentID != uuid.Nil {
		deployment = pgtype.UUID{Bytes: deploymentID, Valid: true}
	}
	if _, err := h.queries.CreateWebhookDelivery(ctx, db.CreateWebhookDeliveryParams{
		ProjectID:        projectID,
		GithubDeliveryID: stringPtr(deliveryID),
		SignatureValid:   signatureValid,
		EventType:        stringPtr(eventType),
		Branch:           branch,
		Processed:        processed,
		DeploymentID:     deployment,
	}); err != nil {
		slog.Warn("log webhook delivery", "projectId", projectID, "error", err)
	}
}

func verifySignature(secret string, body []byte, header string) bool {
	signature, ok := strings.CutPrefix(strings.TrimSpace(header), "sha256=")
	if !ok || signature == "" {
		return false
	}
	expected, err := hex.DecodeString(signature)
	if err != nil {
		return false
	}

	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	actual := mac.Sum(nil)
	return hmac.Equal(actual, expected)
}

func branchFromRef(ref string) string {
	return strings.TrimPrefix(ref, "refs/heads/")
}

func stringPtr(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

type rateLimiter struct {
	limit  int
	window time.Duration
	now    func() time.Time

	mu      sync.Mutex
	buckets map[uuid.UUID]rateBucket
}

type rateBucket struct {
	resetAt time.Time
	count   int
}

func newRateLimiter(limit int, window time.Duration, now func() time.Time) *rateLimiter {
	return &rateLimiter{
		limit:   limit,
		window:  window,
		now:     now,
		buckets: make(map[uuid.UUID]rateBucket),
	}
}

func (l *rateLimiter) allow(projectID uuid.UUID) bool {
	if l.limit <= 0 {
		return false
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	bucket := l.buckets[projectID]
	if bucket.resetAt.IsZero() || !now.Before(bucket.resetAt) {
		bucket = rateBucket{resetAt: now.Add(l.window)}
	}
	if bucket.count >= l.limit {
		l.buckets[projectID] = bucket
		return false
	}
	bucket.count++
	l.buckets[projectID] = bucket
	return true
}
