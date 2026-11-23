package app_test

import (
	"context"
	"errors"
	"net"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"authservice/internal/app"
	"authservice/internal/config"
	"authservice/internal/models"
	"authservice/internal/repository"
	authpb "authservice/proto/auth"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestPostgresApplication(t *testing.T) {
	dsn := os.Getenv("AUTH_TEST_DB_DSN")
	if dsn == "" {
		t.Skip("set AUTH_TEST_DB_DSN to a separate PostgreSQL test database")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{TranslateError: true, Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	cfg := config.Config{
		DatabaseDSN: dsn, JWTSecret: strings.Repeat("s", 32), JWTIssuer: "authservice",
		JWTAudience: "shopapi", TokenTTL: time.Hour,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	c, stop := start(t, cfg)
	email := uuid.NewString() + "@example.com"
	t.Cleanup(func() { db.Where("email = ?", email).Delete(&models.User{}) })
	registered, err := c.Register(ctx, &authpb.RegisterRequest{
		Email: email, FirstName: "Ivan", LastName: "Ivanov", Phone: "+79991234567", Password: "initial-password",
	}, grpc.WaitForReady(true))
	if err != nil {
		t.Fatal(err)
	}
	var saved models.User
	if err := db.Where("email = ?", email).First(&saved).Error; err != nil {
		t.Fatal(err)
	}
	if saved.ID == uuid.Nil || saved.PasswordHash == "initial-password" {
		t.Fatal("user or password was not stored correctly")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(saved.PasswordHash), []byte("initial-password")); err != nil {
		t.Fatal(err)
	}
	duplicate := saved
	duplicate.ID = uuid.New()
	if err := repository.NewUserRepository(db).Create(ctx, &duplicate); err != repository.ErrDuplicateEmail {
		t.Fatalf("duplicate SQL email not translated: %v", err)
	}
	changed, err := c.ChangePassword(ctx, &authpb.ChangePasswordRequest{
		Token: registered.Token, OldPassword: "initial-password", NewPassword: "changed-password",
	})
	if err != nil || !changed.GetSuccess() {
		t.Fatalf("change failed: %v", err)
	}
	if _, err := c.Login(ctx, &authpb.LoginRequest{Email: email, Password: "initial-password"}); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("old password accepted: %v", err)
	}
	stop()
	c, _ = start(t, cfg)
	login, err := c.Login(ctx, &authpb.LoginRequest{Email: email, Password: "changed-password"}, grpc.WaitForReady(true))
	if err != nil || login.GetToken() == "" {
		t.Fatalf("password did not survive restart: %v", err)
	}
	verified, err := c.ValidateToken(ctx, &authpb.ValidateTokenRequest{Token: registered.Token})
	if err != nil || !verified.GetIsValid() || verified.UserId != saved.ID.String() || verified.Email != email {
		t.Fatalf("JWT did not survive restart: %v %v", verified, err)
	}
	reset, err := c.ResetPassword(ctx, &authpb.ResetPasswordRequest{Email: email})
	if err != nil || !reset.GetSuccess() {
		t.Fatalf("reset failed: %v", err)
	}
	if _, err := c.Login(ctx, &authpb.LoginRequest{Email: email, Password: "changed-password"}); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("reset not persisted: %v", err)
	}
	// A password validated before a reset must never overwrite the reset result.
	repo := repository.NewUserRepository(db)
	if err := repo.CompareAndSwapPassword(ctx, saved.ID.String(), saved.PasswordHash, "stale-update"); !errors.Is(err, repository.ErrPasswordChanged) {
		t.Fatalf("stale password change after reset: %v", err)
	}
}

func start(t *testing.T, cfg config.Config) (authpb.AuthServiceClient, func()) {
	t.Helper()
	// Select a free local TCP port before passing it to the production entry point.
	reservation, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	cfg.GRPCAddress = reservation.Addr().String()
	if err := reservation.Close(); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() { result <- app.Run(ctx, cfg) }()
	conn, err := grpc.NewClient(cfg.GRPCAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	var once sync.Once
	stop := func() {
		once.Do(func() {
			_ = conn.Close()
			cancel()
			select {
			case err := <-result:
				if err != nil {
					t.Errorf("application stopped with error: %v", err)
				}
			case <-time.After(10 * time.Second):
				t.Error("application did not shut down")
			}
		})
	}
	t.Cleanup(stop)
	return authpb.NewAuthServiceClient(conn), stop
}
