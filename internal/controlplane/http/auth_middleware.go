package http

import (
	"context"
	nethttp "net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const (
	ctxKeyUserID contextKey = "user_id"
	ctxKeyRole   contextKey = "role"
)

// UserIDFromContext 从请求 context 获取当前用户 ID
func UserIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(ctxKeyUserID).(string)
	return v
}

// RoleFromContext 从请求 context 获取当前用户角色
func RoleFromContext(ctx context.Context) string {
	v, _ := ctx.Value(ctxKeyRole).(string)
	return v
}

// AuthMiddleware 通用 JWT 认证中间件，从 token 提取 user_id 和 role 注入 context
func AuthMiddleware(secret []byte) func(nethttp.Handler) nethttp.Handler {
	return func(next nethttp.Handler) nethttp.Handler {
		return nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
			tokenStr := extractToken(r)
			if tokenStr == "" {
				writeJSON(w, nethttp.StatusUnauthorized, map[string]string{"error": "missing or invalid authorization header"})
				return
			}

			claims := &AuthClaims{}
			token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return secret, nil
			}, jwt.WithValidMethods([]string{"HS256"}))

			if err != nil || !token.Valid {
				writeJSON(w, nethttp.StatusUnauthorized, map[string]string{"error": "invalid token"})
				return
			}

			// 防御性检查：旧 token 可能没有 role claim
			if claims.Role == "" {
				writeJSON(w, nethttp.StatusUnauthorized, map[string]string{"error": "token missing role claim, please re-login"})
				return
			}

			ctx := context.WithValue(r.Context(), ctxKeyUserID, claims.UserID)
			ctx = context.WithValue(ctx, ctxKeyRole, claims.Role)

			// 当通过 query param 认证时，设置 cookie（兼容 VNC 代理等场景）
			if r.URL.Query().Get("token") != "" {
				nethttp.SetCookie(w, &nethttp.Cookie{
					Name:     "admin_token",
					Value:    tokenStr,
					Path:     "/v1/",
					HttpOnly: true,
					SameSite: nethttp.SameSiteLaxMode,
					MaxAge:   86400,
				})
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole 角色限制中间件，只允许指定角色通过
func RequireRole(roles ...string) func(nethttp.Handler) nethttp.Handler {
	return func(next nethttp.Handler) nethttp.Handler {
		return nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
			role := RoleFromContext(r.Context())
			for _, allowed := range roles {
				if role == allowed {
					next.ServeHTTP(w, r)
					return
				}
			}
			writeJSON(w, nethttp.StatusForbidden, map[string]string{"error": "insufficient permissions"})
		})
	}
}

// extractToken 从 Authorization header / query param / cookie 中提取 token
func extractToken(r *nethttp.Request) string {
	if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	if t := r.URL.Query().Get("token"); t != "" {
		return t
	}
	if c, err := r.Cookie("admin_token"); err == nil && c.Value != "" {
		return c.Value
	}
	return ""
}
