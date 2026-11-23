package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"

	"authservice/internal/models"
	"authservice/internal/repository"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("неверный email или пароль")
	ErrEmailExists        = errors.New("пользователь с таким email уже существует")
)

type AuthService struct {
	userRepo repository.UserRepository
}

func NewAuthService(repo repository.UserRepository) *AuthService {
	return &AuthService{userRepo: repo}
}

// 1. Создание пользователя
func (s *AuthService) Register(ctx context.Context, email, firstName, lastName, phone, password string) (*models.User, error) {
	// Проверяем, есть ли уже такой пользователь
	_, err := s.userRepo.GetByEmail(ctx, email)
	if err == nil {
		return nil, ErrEmailExists
	} else if err != repository.ErrNotFound {
		return nil, err
	}

	// Хэшируем пароль
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("ошибка хэширования: %w", err)
	}

	user := &models.User{
		Email:        email,
		FirstName:    firstName,
		LastName:     lastName,
		PhoneNumber:  phone,
		PasswordHash: string(hash),
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		if errors.Is(err, repository.ErrDuplicateEmail) {
			return nil, ErrEmailExists
		}
		return nil, err
	}

	return user, nil
}

// 2. Проверка логина и пароля (Аутентификация)
func (s *AuthService) Login(ctx context.Context, email, password string) (*models.User, error) {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	// Сравниваем присланный пароль с хэшем из базы
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	return user, nil
}

// 3. Смена пароля (например, когда пользователь знает старый)
func (s *AuthService) ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if errors.Is(err, repository.ErrNotFound) {
		return ErrInvalidCredentials
	}
	if err != nil {
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(oldPassword)); err != nil {
		return ErrInvalidCredentials
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("ошибка хэширования: %w", err)
	}
	err = s.userRepo.CompareAndSwapPassword(ctx, userID, user.PasswordHash, string(hash))
	if errors.Is(err, repository.ErrPasswordChanged) {
		return ErrInvalidCredentials
	}
	return err
}

// 4. Восстановление пароля с заглушкой отправки письма
func (s *AuthService) ResetPassword(ctx context.Context, email string) error {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return err
	}

	// Генерируем независимый случайный пароль для каждого сброса
	random := make([]byte, 24)
	if _, err := rand.Read(random); err != nil {
		return fmt.Errorf("ошибка генерации пароля: %w", err)
	}
	tempPassword := base64.RawURLEncoding.EncodeToString(random)

	// Хэшируем и сохраняем
	hash, err := bcrypt.GenerateFromPassword([]byte(tempPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("ошибка хэширования: %w", err)
	}
	err = s.userRepo.UpdatePassword(ctx, user.ID.String(), string(hash))
	if err != nil {
		return err
	}

	// ЗАГЛУШКА ОТПРАВКИ НА ПОЧТУ (вывод в консоль)
	fmt.Printf("\n=== EMAIL SENT ===\nTo: %s\nВаш новый пароль: %s\n==================\n", email, tempPassword)

	return nil
}
