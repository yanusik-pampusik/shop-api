package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"shopapi/internal/models"
)

type addressRepository struct {
	db *gorm.DB
}

func NewAddressRepository(db *gorm.DB) AddressRepository {
	return &addressRepository{db: db}
}

func (r *addressRepository) Create(ctx context.Context, address *models.Address) error {
	return r.db.WithContext(ctx).Create(address).Error
}

func (r *addressRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Address, error) {
	var address models.Address
	err := r.db.WithContext(ctx).First(&address, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &address, nil
}

func (r *addressRepository) Update(ctx context.Context, address *models.Address) error {
	result := r.db.WithContext(ctx).
		Model(&models.Address{}).
		Where("id = ?", address.ID).
		Updates(map[string]interface{}{
			"country": address.Country,
			"city":    address.City,
			"street":  address.Street,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}