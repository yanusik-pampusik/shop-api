package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"

	"shopapi/internal/models"
)

type supplierRepository struct {
	db *gorm.DB
}

func NewSupplierRepository(db *gorm.DB) SupplierRepository {
	return &supplierRepository{db: db}
}

func (r *supplierRepository) Create(ctx context.Context, supplier *models.Supplier) error {
	return r.db.WithContext(ctx).Create(supplier).Error
}

func (r *supplierRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Supplier, error) {
	var supplier models.Supplier
	err := r.db.WithContext(ctx).
		Preload("Address").
		First(&supplier, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &supplier, nil
}

func (r *supplierRepository) GetAll(ctx context.Context) ([]models.Supplier, error) {
	var suppliers []models.Supplier
	err := r.db.WithContext(ctx).Preload("Address").Find(&suppliers).Error
	if err != nil {
		return nil, err
	}
	return suppliers, nil
}

func (r *supplierRepository) UpdateAddress(ctx context.Context, id uuid.UUID, address *models.Address) error {
	return replaceAddress(ctx, r.db, &models.Supplier{}, id, address)
}

func (r *supplierRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.Supplier{}, "id = ?", id)
	if result.Error != nil {
		var pgErr *pgconn.PgError
		if errors.As(result.Error, &pgErr) && pgErr.Code == "23503" {
			return ErrConflict
		}
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
