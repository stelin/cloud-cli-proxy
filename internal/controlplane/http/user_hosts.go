package http

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	nethttp "net/http"

	"github.com/jackc/pgx/v5"

	"github.com/zanel1u/cloud-cli-proxy/internal/agentapi"
	"github.com/zanel1u/cloud-cli-proxy/internal/store/repository"
)

// UserHostStore defines the data access interface for user host endpoints.
type UserHostStore interface {
	ListHostsWithEgressByUserID(context.Context, string) ([]repository.UserHostSummary, error)
	GetHost(context.Context, string) (repository.Host, error)
	GetHostDetail(context.Context, string) (repository.HostDetail, error)
	GetUser(context.Context, string) (repository.User, error)
}

// UserHostsHandler handles user self-service host endpoints.
type UserHostsHandler struct {
	logger *slog.Logger
	store  UserHostStore
	queue  HostActionQueuer
	events EventRecorder
}

// NewUserHostsHandler creates a new UserHostsHandler.
func NewUserHostsHandler(logger *slog.Logger, store UserHostStore, queue HostActionQueuer, events EventRecorder) *UserHostsHandler {
	return &UserHostsHandler{logger: logger, store: store, queue: queue, events: events}
}

// List returns a handler for GET /v1/user/hosts.
func (h *UserHostsHandler) List() nethttp.Handler {
	return nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		userID := UserIDFromContext(r.Context())
		if userID == "" {
			writeJSON(w, nethttp.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}

		hosts, err := h.store.ListHostsWithEgressByUserID(r.Context(), userID)
		if err != nil {
			h.logger.Error("list user hosts failed", "user_id", userID, "error", err)
			writeJSON(w, nethttp.StatusInternalServerError, map[string]string{"error": "list hosts failed"})
			return
		}

		writeJSON(w, nethttp.StatusOK, map[string]any{"hosts": hosts})
	})
}

// Get returns a handler for GET /v1/user/hosts/{hostID}.
func (h *UserHostsHandler) Get() nethttp.Handler {
	return nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		userID := UserIDFromContext(r.Context())
		role := RoleFromContext(r.Context())
		hostID := r.PathValue("hostID")

		if userID == "" {
			writeJSON(w, nethttp.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}

		// Ownership check: fetch host first
		host, err := h.store.GetHost(r.Context(), hostID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				writeJSON(w, nethttp.StatusNotFound, map[string]string{"error": "host not found"})
				return
			}
			h.logger.Error("get host failed", "host_id", hostID, "error", err)
			writeJSON(w, nethttp.StatusInternalServerError, map[string]string{"error": "get host failed"})
			return
		}

		// Admin skips ownership check
		if role != "admin" && host.UserID != userID {
			writeJSON(w, nethttp.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}

		// Get full detail
		detail, err := h.store.GetHostDetail(r.Context(), hostID)
		if err != nil {
			h.logger.Error("get host detail failed", "host_id", hostID, "error", err)
			writeJSON(w, nethttp.StatusInternalServerError, map[string]string{"error": "get host detail failed"})
			return
		}

		// Build response with sensitive fields filtered
		bindings := make([]repository.UserEgressBinding, 0, len(detail.Bindings))
		for _, b := range detail.Bindings {
			bindings = append(bindings, repository.UserEgressBinding{
				IPAddress:  b.EgressIP.IPAddress,
				TunnelType: b.EgressIP.TunnelType,
			})
		}

		resp := repository.UserHostDetail{
			ID:             detail.Host.ID,
			Hostname:       detail.Host.Hostname,
			Status:         detail.Host.Status,
			Timezone:       detail.Host.Timezone,
			CreatedAt:      detail.Host.CreatedAt,
			UpdatedAt:      detail.Host.UpdatedAt,
			EgressBindings: bindings,
		}

		if user, err := h.store.GetUser(r.Context(), userID); err == nil && user.ShortID != "" {
			scheme := "https"
			if r.TLS == nil {
				scheme = "http"
			}
			baseURL := fmt.Sprintf("%s://%s", scheme, r.Host)
			resp.ConnectionInfo = &repository.ConnectionInfo{
				CurlCommand: fmt.Sprintf("curl -sSL %s/entry/%s | bash", baseURL, user.ShortID),
				SSHCommand:  fmt.Sprintf("ssh %s@%s -p 2222", user.ShortID, r.Host),
				SSHPort:     2222,
			}
		}

		writeJSON(w, nethttp.StatusOK, resp)
	})
}

// Rebuild returns a handler for POST /v1/user/hosts/{hostID}/rebuild.
func (h *UserHostsHandler) Rebuild() nethttp.Handler {
	return nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		userID := UserIDFromContext(r.Context())
		role := RoleFromContext(r.Context())
		hostID := r.PathValue("hostID")

		if userID == "" {
			writeJSON(w, nethttp.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}

		// Ownership check
		host, err := h.store.GetHost(r.Context(), hostID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				writeJSON(w, nethttp.StatusNotFound, map[string]string{"error": "host not found"})
				return
			}
			h.logger.Error("get host failed", "host_id", hostID, "error", err)
			writeJSON(w, nethttp.StatusInternalServerError, map[string]string{"error": "get host failed"})
			return
		}

		// Admin skips ownership check
		if role != "admin" && host.UserID != userID {
			writeJSON(w, nethttp.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}

		task, err := h.queue.QueueHostAction(r.Context(), hostID, agentapi.ActionRebuildHost, userID)
		if err != nil {
			h.logger.Error("queue rebuild failed", "host_id", hostID, "error", err)
			writeJSON(w, nethttp.StatusInternalServerError, map[string]string{"error": "queue rebuild failed"})
			return
		}

		// Record event
		if h.events != nil {
			if _, err := h.events.RecordEvent(r.Context(), repository.RecordEventParams{
				HostID:  &hostID,
				UserID:  &userID,
				Level:   "info",
				Type:    "user.host.rebuild",
				Message: "用户发起主机重建",
				Metadata: map[string]any{
					"user_id": userID,
				},
			}); err != nil {
				h.logger.Error("record event failed", "type", "user.host.rebuild", "error", err)
			}
		}

		writeJSON(w, nethttp.StatusAccepted, map[string]any{
			"task_id": task.ID,
			"status":  "202 Accepted",
		})
	})
}
