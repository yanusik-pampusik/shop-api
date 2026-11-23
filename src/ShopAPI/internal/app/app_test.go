package app_test

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"shopapi/internal/app"
	"shopapi/internal/config"
	authpb "shopapi/proto/auth"
)

type authServer struct {
	authpb.UnimplementedAuthServiceServer
	validations atomic.Int32
}

func (s *authServer) ValidateToken(ctx context.Context, r *authpb.ValidateTokenRequest) (*authpb.ValidateTokenResponse, error) {
	s.validations.Add(1)
	switch r.Token {
	case "unavailable":
		return nil, status.Error(codes.Unavailable, "database password should never escape")
	case "slow":
		<-ctx.Done()
		return nil, status.FromContextError(ctx.Err()).Err()
	case "valid":
		return &authpb.ValidateTokenResponse{IsValid: true, UserId: "user-1", Email: "user@example.com"}, nil
	default:
		return &authpb.ValidateTokenResponse{IsValid: false}, nil
	}
}

func (s *authServer) Register(_ context.Context, r *authpb.RegisterRequest) (*authpb.AuthResponse, error) {
	if r.Email == "taken@example.com" {
		return nil, status.Error(codes.AlreadyExists, "taken")
	}
	if r.FirstName != "Ivan" || r.LastName != "Ivanov" || r.Phone != "+79990000000" || r.Password != "password" {
		return nil, status.Error(codes.InvalidArgument, "bad mapping")
	}
	return &authpb.AuthResponse{Token: "registered-token"}, nil
}
func (s *authServer) Login(ctx context.Context, r *authpb.LoginRequest) (*authpb.AuthResponse, error) {
	switch r.Password {
	case "wrong":
		return nil, status.Error(codes.Unauthenticated, "wrong")
	case "invalid":
		return nil, status.Error(codes.InvalidArgument, "bad")
	case "offline":
		return nil, status.Error(codes.Unavailable, "private database connection string")
	case "internal":
		return nil, status.Error(codes.Internal, "private database connection string")
	case "slow":
		<-ctx.Done()
		return nil, status.FromContextError(ctx.Err()).Err()
	}
	return &authpb.AuthResponse{Token: "login-token"}, nil
}
func (s *authServer) ResetPassword(_ context.Context, r *authpb.ResetPasswordRequest) (*authpb.ResetPasswordResponse, error) {
	if r.Email != "user@example.com" {
		return nil, status.Error(codes.InvalidArgument, "email")
	}
	return &authpb.ResetPasswordResponse{Success: true}, nil
}
func (s *authServer) ChangePassword(_ context.Context, r *authpb.ChangePasswordRequest) (*authpb.ChangePasswordResponse, error) {
	if r.Token != "valid" || r.OldPassword != "old" || r.NewPassword != "new" {
		return nil, status.Error(codes.Unauthenticated, "bad mapping")
	}
	return &authpb.ChangePasswordResponse{Success: true}, nil
}

// This resolves the real IoC graph and communicates over actual gRPC/HTTP2.
func setup(t *testing.T) (*gin.Engine, *authServer) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	backend := &authServer{}
	authpb.RegisterAuthServiceServer(server, backend)
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { server.Stop(); _ = listener.Close() })
	conn, err := grpc.NewClient("passthrough:///auth", grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	c, err := app.Container(&config.Config{AuthTimeout: 100 * time.Millisecond}, nil, authpb.NewAuthServiceClient(conn))
	if err != nil {
		t.Fatal(err)
	}
	var router *gin.Engine
	if err := c.Invoke(func(r *gin.Engine) { router = r }); err != nil {
		t.Fatal(err)
	}
	return router, backend
}
func request(router http.Handler, method, path, body, authorization string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func TestEveryStoreRouteRequiresAuthorization(t *testing.T) {
	router, backend := setup(t)
	protected := 0
	for _, route := range router.Routes() {
		if !strings.HasPrefix(route.Path, "/api/v1/") && route.Path != "/change-password" {
			continue
		}
		protected++
		path := strings.ReplaceAll(strings.ReplaceAll(route.Path, ":product_id", "invalid-id"), ":id", "invalid-id")
		t.Run(route.Method+" "+path, func(t *testing.T) {
			for _, header := range []string{"", "Basic abc", "Bearer", "Bearer a b", "Bearer invalid"} {
				w := request(router, route.Method, path, `{}`, header)
				if w.Code != http.StatusUnauthorized {
					t.Fatalf("%q: %d %s", header, w.Code, w.Body)
				}
				if w.Header().Get("WWW-Authenticate") != "Bearer" {
					t.Fatal("missing challenge")
				}
			}
		})
	}
	if protected != 22 {
		t.Fatalf("expected 22 protected routes, got %d", protected)
	}
	if backend.validations.Load() != int32(protected) {
		t.Fatalf("malformed headers must not call gRPC: %d", backend.validations.Load())
	}
}

func TestVerificationOnEveryRequestAndFailClosed(t *testing.T) {
	router, backend := setup(t)
	for i := 0; i < 2; i++ {
		w := request(router, "GET", "/api/v1/products/invalid-id", "", "bearer valid")
		if w.Code != 400 {
			t.Fatalf("verified request did not reach controller: %d %s", w.Code, w.Body)
		}
	}
	if backend.validations.Load() != 2 {
		t.Fatal("token verification was cached")
	}
	for _, token := range []string{"unavailable", "slow", "invalid"} {
		w := request(router, "GET", "/api/v1/products/invalid-id", "", "Bearer "+token)
		if w.Code != 401 {
			t.Fatalf("%s bypassed protection: %d", token, w.Code)
		}
	}
	req := httptest.NewRequest("GET", "/api/v1/products", nil)
	req.Header.Add("Authorization", "Bearer valid")
	req.Header.Add("Authorization", "Bearer invalid")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != 401 {
		t.Fatal("duplicate authorization headers accepted")
	}
}

func TestPublicAuthEndpointsAndPasswordChange(t *testing.T) {
	router, backend := setup(t)
	cases := []struct {
		path, body, token string
		code              int
		response          string
	}{
		{"/register", `{"email":"user@example.com","first_name":"Ivan","last_name":"Ivanov","phone":"+79990000000","password":"password"}`, "", 201, `{"token":"registered-token"}`},
		{"/register", `{"email":"taken@example.com","first_name":"Ivan","last_name":"Ivanov","phone":"+79990000000","password":"password"}`, "", 409, ""},
		{"/register", `{}`, "", 400, ""},
		{"/auth", `{"email":"user@example.com","password":"password"}`, "", 200, `{"token":"login-token"}`},
		{"/auth", `{"email":"user@example.com","password":"wrong"}`, "", 401, ""},
		{"/auth", `{"email":"user@example.com","password":"invalid"}`, "", 400, ""},
		{"/auth", `{"email":"user@example.com","password":"offline"}`, "", 503, ""},
		{"/auth", `{"email":"user@example.com","password":"internal"}`, "", 500, ""},
		{"/auth", `{"email":"user@example.com","password":"slow"}`, "", 504, ""},
		{"/auth", `{`, "", 400, ""},
		{"/reset", `{"email":"user@example.com"}`, "", 200, `{"success":true}`},
		{"/reset", `{}`, "", 400, ""},
		{"/change-password", `{"old_password":"old","new_password":"new"}`, "Bearer valid", 200, `{"success":true}`},
		{"/change-password", `{"old_password":"old","new_password":"new"}`, "", 401, ""},
	}
	for _, tc := range cases {
		w := request(router, "POST", tc.path, tc.body, tc.token)
		if w.Code != tc.code || (tc.response != "" && w.Body.String() != tc.response) {
			t.Errorf("%s: got %d %s", tc.path, w.Code, w.Body)
		}
		if strings.Contains(w.Body.String(), "private") {
			t.Fatal("RPC error details leaked")
		}
		if strings.Contains(tc.response, "token") && w.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("token response may be cached")
		}
	}
	if backend.validations.Load() != 1 {
		t.Fatal("public endpoints required verification")
	}
	w := request(router, "POST", "/auth", `{"email":"user@example.com","password":"`+strings.Repeat("p", 17000)+`"}`, "")
	if w.Code != 400 {
		t.Fatal("oversized request accepted")
	}
	w = request(router, "GET", "/swagger/doc.json", "", "")
	if w.Code != 200 {
		t.Fatalf("swagger unavailable: %d", w.Code)
	}
	_, _ = io.Copy(io.Discard, w.Body)
}
