package http

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	nethttp "net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/zanel1u/cloud-cli-proxy/internal/store/repository"
)

// AuthUserStore 统一登录所需的 store 接口
type AuthUserStore interface {
	GetUserByShortIDForAuth(context.Context, string) (repository.User, error)
}

type UnifiedLoginHandler struct {
	store     AuthUserStore
	jwtSecret []byte
	logger    *slog.Logger
}

func NewUnifiedLoginHandler(logger *slog.Logger, store AuthUserStore, jwtSecret []byte) *UnifiedLoginHandler {
	return &UnifiedLoginHandler{store: store, jwtSecret: jwtSecret, logger: logger}
}

type unifiedLoginRequest struct {
	ShortID  string `json:"short_id"`
	Password string `json:"password"`
	// 兼容旧前端发送 username 字段
	Username string `json:"username"`
}

func (h *UnifiedLoginHandler) ServeHTTP(w nethttp.ResponseWriter, r *nethttp.Request) {
	if r.Method != nethttp.MethodPost {
		writeJSON(w, nethttp.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req unifiedLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, nethttp.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	// 兼容：如果 short_id 为空但 username 有值，用 username 作为 short_id
	shortID := req.ShortID
	if shortID == "" {
		shortID = req.Username
	}
	if shortID == "" || req.Password == "" {
		writeJSON(w, nethttp.StatusBadRequest, map[string]string{"error": "short_id and password are required"})
		return
	}

	user, err := h.store.GetUserByShortIDForAuth(r.Context(), shortID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, nethttp.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
			return
		}
		h.logger.Error("unified login: lookup user failed", "short_id", shortID, "error", err)
		writeJSON(w, nethttp.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	if user.Status != "active" {
		writeJSON(w, nethttp.StatusForbidden, map[string]string{"error": "account is not active"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		writeJSON(w, nethttp.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
		return
	}

	tokenStr, err := GenerateAuthToken(h.jwtSecret, user.ID, user.Role, 24*time.Hour)
	if err != nil {
		h.logger.Error("unified login: sign jwt failed", "error", err)
		writeJSON(w, nethttp.StatusInternalServerError, map[string]string{"error": "token generation failed"})
		return
	}

	writeJSON(w, nethttp.StatusOK, map[string]any{
		"token":      tokenStr,
		"role":       user.Role,
		"expires_in": 86400,
	})
}
