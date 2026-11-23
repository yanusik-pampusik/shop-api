package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"shopapi/internal/models"
)

// ErrNotFound — общая ошибка «запись не найдена».
var ErrNotFound = errors.New("запись не найдена")

type clientRepository struct {
	db *gorm.DB
}

// NewClientRepository создаёт репозиторий клиентов.
func NewClientRepository(db *gorm.DB) ClientRepository {
	return &clientRepository{db: db}
}

func (r *clientRepository) Create(ctx context.Context, client *models.Client) error {
	return r.db.WithContext(ctx).Create(client).Error
}

func (r *clientRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Client, error) {
	var client models.Client
	err := r.db.WithContext(ctx).
		Preload("Address").
		First(&client, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &client, nil
}

func (r *clientRepository) GetAll(ctx context.Context, limit, offset int) ([]models.Client, error) {
	var clients []models.Client
	query := r.db.WithContext(ctx).Preload("Address")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Find(&clients).Error; err != nil {
		return nil, err
	}
	return clients, nil
}

func (r *clientRepository) GetByNameAndSurname(ctx context.Context, name, surname string) ([]models.Client, error) {
	var clients []models.Client
	err := r.db.WithContext(ctx).
		Preload("Address").
		Where("client_name = ? AND client_surname = ?", name, surname).
		Find(&clients).Error
	if err != nil {
		return nil, err
	}
	return clients, nil
}

func (r *clientRepository) UpdateAddress(ctx context.Context, id uuid.UUID, address *models.Address) error {
	return replaceAddress(ctx, r.db, &models.Client{}, id, address)
}

func (r *clientRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.Client{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
