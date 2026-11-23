package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"shopapi/internal/models"
)

type productRepository struct {
	db *gorm.DB
}

func NewProductRepository(db *gorm.DB) ProductRepository {
	return &productRepository{db: db}
}

func (r *productRepository) Create(ctx context.Context, product *models.Product) error {
	return r.db.WithContext(ctx).Create(product).Error
}

func (r *productRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Product, error) {
	var product models.Product
	err := r.db.WithContext(ctx).
		Preload("Category").
		Preload("Supplier").
		Preload("Supplier.Address").
		First(&product, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &product, nil
}

func (r *productRepository) GetAll(ctx context.Context) ([]models.Product, error) {
	var products []models.Product
	err := r.db.WithContext(ctx).
		Preload("Category").
		Preload("Supplier").
		Preload("Supplier.Address").
		Find(&products).Error
	if err != nil {
		return nil, err
	}
	return products, nil
}

// DecreaseStock performs the check and deduction in one database statement.
func (r *productRepository) DecreaseStock(ctx context.Context, id uuid.UUID, amount int) error {
	if amount <= 0 {
		return ErrInsufficientStock
	}
	result := r.db.WithContext(ctx).Model(&models.Product{}).
		Where("id = ? AND available_stock >= ?", id, amount).
		Updates(map[string]any{"available_stock": gorm.Expr("available_stock - ?", amount), "last_update_date": gorm.Expr("CURRENT_TIMESTAMP")})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 0 {
		return nil
	}
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.Product{}).Where("id = ?", id).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return ErrNotFound
	}
	return ErrInsufficientStock
}

// SetImage cannot overwrite a stock change made after the product was read.
func (r *productRepository) SetImage(ctx context.Context, id, imageID uuid.UUID) error {
	result := r.db.WithContext(ctx).Model(&models.Product{}).Where("id = ?", id).Update("image_id", imageID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *productRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.Product{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
