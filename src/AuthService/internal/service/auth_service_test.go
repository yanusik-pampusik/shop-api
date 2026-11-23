package service_test

import (
	"authservice/internal/models"
	"authservice/internal/repository"
	"authservice/internal/service"
	"context"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"testing"
)

// Фейковый репозиторий для тестов
type mockRepo struct {
	users map[string]*models.User
}

func (m *mockRepo) Create(ctx context.Context, user *models.User) error {
	user.ID = uuid.New()
	m.users[user.Email] = user
	return nil
}

func (m *mockRepo) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	user, ok := m.users[email]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return user, nil
}

func (m *mockRepo) UpdatePassword(ctx context.Context, userID string, newHash string) error {
	user, err := m.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	user.PasswordHash = newHash
	return nil
}

func (m *mockRepo) GetByID(ctx context.Context, userID string) (*models.User, error) {
	for _, user := range m.users {
		if user.ID.String() == userID {
			return user, nil
		}
	}
	return nil, repository.ErrNotFound
}

func TestRegisterAndLogin(t *testing.T) {
	repo := &mockRepo{users: make(map[string]*models.User)}
	svc := service.NewAuthService(repo)
	ctx := context.Background()

	// 1. Тест регистрации
	user, err := svc.Register(ctx, "test@test.com", "Ivan", "Ivanov", "+7999", "secret123")
	assert.NoError(t, err)
	assert.NotNil(t, user)

	// Проверяем, что пароль не хранится в открытом виде
	assert.NotEqual(t, "secret123", user.PasswordHash)

	// 2. Тест успешного логина
	_, err = svc.Login(ctx, "test@test.com", "secret123")
	assert.NoError(t, err)

	// 3. Тест логина с неверным паролем
	_, err = svc.Login(ctx, "test@test.com", "wrong_pass")
	assert.Equal(t, service.ErrInvalidCredentials, err)
}

func (m *mockRepo) CompareAndSwapPassword(ctx context.Context, userID, oldHash, newHash string) error {
	user, err := m.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if user.PasswordHash != oldHash {
		return repository.ErrPasswordChanged
	}
	user.PasswordHash = newHash
	return nil
}
