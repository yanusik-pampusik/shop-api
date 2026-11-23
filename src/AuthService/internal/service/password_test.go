package service_test

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"authservice/internal/models"
	"authservice/internal/service"

	"golang.org/x/crypto/bcrypt"
)

func TestPasswordChangeAndRandomReset(t *testing.T) {
	repo := &mockRepo{users: make(map[string]*models.User)}
	svc := service.NewAuthService(repo)
	ctx := context.Background()
	user, err := svc.Register(ctx, "user@example.com", "Ivan", "Ivanov", "+79991234567", "old-password")
	if err != nil {
		t.Fatal(err)
	}
	original := user.PasswordHash
	if err := svc.ChangePassword(ctx, user.ID.String(), "wrong", "new-password"); err != service.ErrInvalidCredentials {
		t.Fatalf("wrong password accepted: %v", err)
	}
	if user.PasswordHash != original {
		t.Fatal("failed change modified the password")
	}
	if err := svc.ChangePassword(ctx, user.ID.String(), "old-password", "new-password"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Login(ctx, user.Email, "new-password"); err != nil {
		t.Fatal(err)
	}
	first := captureReset(t, func() error { return svc.ResetPassword(ctx, user.Email) })
	second := captureReset(t, func() error { return svc.ResetPassword(ctx, user.Email) })
	if first == second || first == "new_temp_pass_123" || len(first) != 32 || len(second) != 32 {
		t.Fatal("reset passwords must be random and independently generated")
	}
	if _, err := svc.Login(ctx, user.Email, first); err != service.ErrInvalidCredentials {
		t.Fatalf("previous reset password still works: %v", err)
	}
	if _, err := svc.Login(ctx, user.Email, second); err != nil {
		t.Fatal(err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(second)); err != nil {
		t.Fatal(err)
	}
}

func captureReset(t *testing.T, reset func() error) string {
	t.Helper()
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	previous := os.Stdout
	os.Stdout = writer
	defer func() { os.Stdout = previous; _ = reader.Close(); _ = writer.Close() }()
	resetErr := reset()
	_ = writer.Close()
	os.Stdout = previous
	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if resetErr != nil {
		t.Fatal(resetErr)
	}
	_, tail, ok := strings.Cut(string(output), "Ваш новый пароль: ")
	if !ok {
		t.Fatalf("reset email was not printed: %s", output)
	}
	password, _, _ := strings.Cut(tail, "\n")
	return password
}
