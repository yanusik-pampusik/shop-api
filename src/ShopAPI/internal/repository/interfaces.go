package repository

import (
	"context"

	"github.com/google/uuid"
	"shopapi/internal/models"
)

// ClientRepository — контракт для работы с клиентами.
type ClientRepository interface {
	Create(ctx context.Context, client *models.Client) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Client, error)
	GetAll(ctx context.Context, limit, offset int) ([]models.Client, error)
	GetByNameAndSurname(ctx context.Context, name, surname string) ([]models.Client, error)
	UpdateAddress(ctx context.Context, id uuid.UUID, address *models.Address) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// AddressRepository — контракт для работы с адресами.
type AddressRepository interface {
	Create(ctx context.Context, address *models.Address) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Address, error)
	Update(ctx context.Context, address *models.Address) error
}

// SupplierRepository — контракт для работы с поставщиками.
type SupplierRepository interface {
	Create(ctx context.Context, supplier *models.Supplier) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Supplier, error)
	GetAll(ctx context.Context) ([]models.Supplier, error)
	UpdateAddress(ctx context.Context, id uuid.UUID, address *models.Address) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// ProductRepository — контракт для работы с товарами.
type ProductRepository interface {
	Create(ctx context.Context, product *models.Product) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Product, error)
	GetAll(ctx context.Context) ([]models.Product, error)
	DecreaseStock(ctx context.Context, id uuid.UUID, amount int) error
	SetImage(ctx context.Context, id, imageID uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// ImageRepository — контракт для работы с изображениями.
type ImageRepository interface {
	Create(ctx context.Context, image *models.Image) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Image, error)
	GetByProductID(ctx context.Context, productID uuid.UUID) (*models.Image, error)
	Update(ctx context.Context, image *models.Image) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// CategoryRepository — контракт для работы с категориями.
type CategoryRepository interface {
	Create(ctx context.Context, category *models.Category) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Category, error)
	GetAll(ctx context.Context) ([]models.Category, error)
}
