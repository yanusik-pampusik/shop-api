package transport_test

import (
	"context"
	"errors"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"authservice/internal/models"
	"authservice/internal/repository"
	"authservice/internal/service"
	"authservice/internal/token"
	"authservice/internal/transport"
	authpb "authservice/proto/auth"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

type memoryRepo struct {
	mu      sync.Mutex
	users   map[string]models.User
	failure error
}

func (r *memoryRepo) Create(_ context.Context, user *models.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	user.ID = uuid.New()
	r.users[user.Email] = *user
	return nil
}
func (r *memoryRepo) GetByEmail(_ context.Context, email string) (*models.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.failure != nil {
		return nil, r.failure
	}
	user, ok := r.users[email]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return &user, nil
}
func (r *memoryRepo) GetByID(_ context.Context, id string) (*models.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, user := range r.users {
		if user.ID.String() == id {
			return &user, nil
		}
	}
	return nil, repository.ErrNotFound
}
func (r *memoryRepo) UpdatePassword(_ context.Context, id, hash string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for email, user := range r.users {
		if user.ID.String() == id {
			user.PasswordHash = hash
			r.users[email] = user
			return nil
		}
	}
	return repository.ErrNotFound
}

func client(t *testing.T, repo *memoryRepo) authpb.AuthServiceClient {
	t.Helper()
	tokens, err := token.New(strings.Repeat("s", 32), "authservice", "shopapi", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	server := grpc.NewServer()
	authpb.RegisterAuthServiceServer(server, transport.New(service.NewAuthService(repo), tokens))
	listener := bufconn.Listen(1024 * 1024)
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(server.Stop)
	conn, err := grpc.NewClient("passthrough:///auth", grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return listener.DialContext(ctx) }))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return authpb.NewAuthServiceClient(conn)
}

func TestGRPCUserFlow(t *testing.T) {
	repo := &memoryRepo{users: make(map[string]models.User)}
	c := client(t, repo)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	registration := &authpb.RegisterRequest{Email: "user@example.com", FirstName: "Ivan", LastName: "Ivanov", Phone: "+79991234567", Password: "initial-password"}
	registered, err := c.Register(ctx, registration)
	if err != nil {
		t.Fatal(err)
	}
	if registered.Token == "" {
		t.Fatal("registration returned no JWT")
	}
	claims, err := c.ValidateToken(ctx, &authpb.ValidateTokenRequest{Token: registered.Token})
	if err != nil || !claims.GetIsValid() || claims.Email != registration.Email || claims.UserId == "" {
		t.Fatalf("token verification failed: %v %v", claims, err)
	}
	if _, err := c.Register(ctx, registration); status.Code(err) != codes.AlreadyExists {
		t.Fatalf("duplicate: %v", err)
	}
	if _, err := c.Login(ctx, &authpb.LoginRequest{Email: registration.Email, Password: "wrong"}); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("bad credentials: %v", err)
	}
	login, err := c.Login(ctx, &authpb.LoginRequest{Email: registration.Email, Password: registration.Password})
	if err != nil || login.GetToken() == "" {
		t.Fatalf("login failed: %v", err)
	}
	changed, err := c.ChangePassword(ctx, &authpb.ChangePasswordRequest{Token: registered.Token, OldPassword: registration.Password, NewPassword: "new-password"})
	if err != nil || !changed.GetSuccess() {
		t.Fatalf("change failed: %v", err)
	}
	if _, err := c.Login(ctx, &authpb.LoginRequest{Email: registration.Email, Password: registration.Password}); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("old password still works: %v", err)
	}
	if _, err := c.Login(ctx, &authpb.LoginRequest{Email: registration.Email, Password: "new-password"}); err != nil {
		t.Fatal(err)
	}
	if _, err := c.ChangePassword(ctx, &authpb.ChangePasswordRequest{Token: registered.Token, OldPassword: "wrong", NewPassword: "other"}); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("wrong old password: %v", err)
	}
	if _, err := c.ChangePassword(ctx, &authpb.ChangePasswordRequest{Token: "fake", OldPassword: "new-password", NewPassword: "other"}); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("fake token: %v", err)
	}
	invalid, err := c.ValidateToken(ctx, &authpb.ValidateTokenRequest{Token: "fake"})
	if err != nil || invalid.GetIsValid() {
		t.Fatalf("invalid token accepted: %v", err)
	}
	for _, email := range []string{registration.Email, "unknown@example.com"} {
		reset, err := c.ResetPassword(ctx, &authpb.ResetPasswordRequest{Email: email})
		if err != nil || !reset.GetSuccess() {
			t.Fatalf("reset failed: %v", err)
		}
	}
	if _, err := c.Login(ctx, &authpb.LoginRequest{Email: registration.Email, Password: "new-password"}); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("password before reset still works: %v", err)
	}
}

func TestGRPCValidationAndInternalErrors(t *testing.T) {
	repo := &memoryRepo{users: make(map[string]models.User)}
	c := client(t, repo)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for _, req := range []*authpb.RegisterRequest{
		{}, {Email: "bad", FirstName: "a", LastName: "b", Phone: "123", Password: "password"},
		{Email: "ok@example.com", FirstName: "a", LastName: "b", Phone: "123", Password: strings.Repeat("x", 73)},
	} {
		if _, err := c.Register(ctx, req); status.Code(err) != codes.InvalidArgument {
			t.Fatalf("invalid registration: %v", err)
		}
	}
	if _, err := c.Login(ctx, &authpb.LoginRequest{}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("empty login: %v", err)
	}
	if _, err := c.ResetPassword(ctx, &authpb.ResetPasswordRequest{}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("empty reset: %v", err)
	}
	repo.mu.Lock()
	repo.failure = errors.New("private database failure")
	repo.mu.Unlock()
	_, err := c.Login(ctx, &authpb.LoginRequest{Email: "user@example.com", Password: "password"})
	if status.Code(err) != codes.Internal || strings.Contains(err.Error(), "private") {
		t.Fatalf("internal error leaked: %v", err)
	}
}

func (r *memoryRepo) CompareAndSwapPassword(_ context.Context, id, oldHash, newHash string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for email, user := range r.users {
		if user.ID.String() == id && user.PasswordHash == oldHash {
			user.PasswordHash = newHash
			r.users[email] = user
			return nil
		}
	}
	return repository.ErrPasswordChanged
}

func TestGRPCRejectsNULInStoredText(t *testing.T) {
	c := client(t, &memoryRepo{users: make(map[string]models.User)})
	for _, field := range []string{"first_name", "last_name", "phone"} {
		t.Run(field, func(t *testing.T) {
			req := &authpb.RegisterRequest{Email: "valid@example.com", FirstName: "Ivan", LastName: "Ivanov", Phone: "123", Password: "password"}
			switch field {
			case "first_name":
				req.FirstName = "a\x00b"
			case "last_name":
				req.LastName = "a\x00b"
			case "phone":
				req.Phone = "1\x002"
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			if _, err := c.Register(ctx, req); status.Code(err) != codes.InvalidArgument {
				t.Fatalf("invalid stored text: %v", err)
			}
		})
	}
}
