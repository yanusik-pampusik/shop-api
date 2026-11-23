package repository

import (
	"context"
	"errors"

	"authservice/internal/models"

	"gorm.io/gorm"
)

var ErrNotFound = errors.New("пользователь не найден")
var ErrPasswordChanged = errors.New("password changed concurrently")
var ErrDuplicateEmail = errors.New("email already exists")

type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	GetByID(ctx context.Context, userID string) (*models.User, error)
	UpdatePassword(ctx context.Context, userID string, newHash string) error
	CompareAndSwapPassword(ctx context.Context, userID, oldHash, newHash string) error
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, user *models.User) error {
	err := r.db.WithContext(ctx).Create(user).Error
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return ErrDuplicateEmail
	}
	return err
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).First(&user, "email = ?", email).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) UpdatePassword(ctx context.Context, userID string, newHash string) error {
	result := r.db.WithContext(ctx).Model(&models.User{}).Where("id = ?", userID).Update("password_hash", newHash)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *userRepository) GetByID(ctx context.Context, userID string) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).First(&user, "id = ?", userID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// CompareAndSwapPassword rejects a password changed or reset since validation.
func (r *userRepository) CompareAndSwapPassword(ctx context.Context, userID, oldHash, newHash string) error {
	result := r.db.WithContext(ctx).Model(&models.User{}).
		Where("id = ? AND password_hash = ?", userID, oldHash).Update("password_hash", newHash)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrPasswordChanged
	}
	return nil
}
