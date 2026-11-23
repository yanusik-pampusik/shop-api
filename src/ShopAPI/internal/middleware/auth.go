package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	authpb "shopapi/proto/auth"
)

const TokenKey = "auth.token"
const UserIDKey = "auth.user_id"

// RequireAuth is the Go equivalent of an authorization attribute: attach it to
// a route or group. All verification runs here, before its controller executes.
// Every request is verified remotely; the shop never receives the signing key.
func RequireAuth(auth authpb.AuthServiceClient, timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		headers := c.Request.Header.Values("Authorization")
		if len(headers) != 1 {
			unauthorized(c)
			return
		}
		parts := strings.Fields(headers[0])
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			unauthorized(c)
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()
		result, err := auth.ValidateToken(ctx, &authpb.ValidateTokenRequest{Token: parts[1]})
		if err != nil || !result.GetIsValid() || result.GetUserId() == "" {
			unauthorized(c)
			return
		}
		c.Set(TokenKey, parts[1])
		c.Set(UserIDKey, result.UserId)
		c.Next()
	}
}

func unauthorized(c *gin.Context) {
	c.Header("WWW-Authenticate", "Bearer")
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "valid Bearer token is required"})
}
