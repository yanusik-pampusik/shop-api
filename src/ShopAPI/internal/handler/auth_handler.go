package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"shopapi/internal/config"
	"shopapi/internal/dto"
	"shopapi/internal/middleware"
	authpb "shopapi/proto/auth"
)

type AuthHandler struct {
	auth    authpb.AuthServiceClient
	timeout time.Duration
}

func NewAuthHandler(auth authpb.AuthServiceClient, cfg *config.Config) *AuthHandler {
	return &AuthHandler{auth: auth, timeout: cfg.AuthTimeout}
}

// Register godoc
// @Summary Register a user and issue a signed JWT
// @Tags auth
// @Accept json
// @Produce json
// @Param user body dto.RegisterRequest true "User details"
// @Success 201 {object} dto.AuthResponse
// @Failure 400,409,500,503,504 {object} map[string]string
// @Router /register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var input dto.RegisterRequest
	if !bindAuth(c, &input) {
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()
	result, err := h.auth.Register(ctx, &authpb.RegisterRequest{
		Email: input.Email, FirstName: input.FirstName, LastName: input.LastName,
		Phone: input.Phone, Password: input.Password,
	})
	if err != nil {
		authError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusCreated, dto.AuthResponse{Token: result.GetToken()})
}

// Login godoc
// @Summary Authenticate by email and password
// @Tags auth
// @Accept json
// @Produce json
// @Param credentials body dto.LoginRequest true "Credentials"
// @Success 200 {object} dto.AuthResponse
// @Failure 400,401,500,503,504 {object} map[string]string
// @Router /auth [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var input dto.LoginRequest
	if !bindAuth(c, &input) {
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()
	result, err := h.auth.Login(ctx, &authpb.LoginRequest{Email: input.Email, Password: input.Password})
	if err != nil {
		authError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, dto.AuthResponse{Token: result.GetToken()})
}

// Reset godoc
// @Summary Reset a password (email is printed in AuthService console)
// @Tags auth
// @Accept json
// @Produce json
// @Param user body dto.ResetRequest true "Email"
// @Success 200 {object} dto.SuccessResponse
// @Failure 400,500,503,504 {object} map[string]string
// @Router /reset [post]
func (h *AuthHandler) Reset(c *gin.Context) {
	var input dto.ResetRequest
	if !bindAuth(c, &input) {
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()
	result, err := h.auth.ResetPassword(ctx, &authpb.ResetPasswordRequest{Email: input.Email})
	if err != nil {
		authError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.SuccessResponse{Success: result.GetSuccess()})
}

// ChangePassword godoc
// @Summary Change the authenticated user's password
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param passwords body dto.ChangePasswordRequest true "Old and new password"
// @Success 200 {object} dto.SuccessResponse
// @Failure 400,401,500,503,504 {object} map[string]string
// @Router /change-password [post]
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var input dto.ChangePasswordRequest
	if !bindAuth(c, &input) {
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()
	result, err := h.auth.ChangePassword(ctx, &authpb.ChangePasswordRequest{
		Token: c.GetString(middleware.TokenKey), OldPassword: input.OldPassword, NewPassword: input.NewPassword,
	})
	if err != nil {
		authError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.SuccessResponse{Success: result.GetSuccess()})
}

func bindAuth(c *gin.Context, value any) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 16*1024)
	if err := c.ShouldBindJSON(value); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON or missing required fields"})
		return false
	}
	return true
}

func authError(c *gin.Context, err error) {
	code, message := http.StatusInternalServerError, "authentication service error"
	switch status.Code(err) {
	case codes.InvalidArgument:
		code, message = http.StatusBadRequest, "invalid user data"
	case codes.AlreadyExists:
		code, message = http.StatusConflict, "email is already registered"
	case codes.Unauthenticated:
		code, message = http.StatusUnauthorized, "invalid credentials"
	case codes.Unavailable:
		code, message = http.StatusServiceUnavailable, "authentication service unavailable"
	case codes.DeadlineExceeded, codes.Canceled:
		code, message = http.StatusGatewayTimeout, "authentication request timed out"
	}
	c.JSON(code, gin.H{"error": message})
}
