package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"shopapi/internal/models"
)

type imageRepository struct {
	db *gorm.DB
}

func NewImageRepository(db *gorm.DB) ImageRepository {
	return &imageRepository{db: db}
}

func (r *imageRepository) Create(ctx context.Context, image *models.Image) error {
	return r.db.WithContext(ctx).Create(image).Error
}

func (r *imageRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Image, error) {
	var image models.Image
	err := r.db.WithContext(ctx).First(&image, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &image, nil
}

func (r *imageRepository) GetByProductID(ctx context.Context, productID uuid.UUID) (*models.Image, error) {
	var product models.Product
	err := r.db.WithContext(ctx).First(&product, "id = ?", productID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if product.ImageID == nil {
		return nil, ErrNotFound
	}

	var image models.Image
	err = r.db.WithContext(ctx).First(&image, "id = ?", *product.ImageID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &image, nil
}

func (r *imageRepository) Update(ctx context.Context, image *models.Image) error {
	result := r.db.WithContext(ctx).
		Model(&models.Image{}).
		Where("id = ?", image.ID).
		Update("image", image.Image)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *imageRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.Image{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}