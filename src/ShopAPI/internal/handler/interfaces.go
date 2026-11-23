package handler

import (
	"context"
	"github.com/google/uuid"
	"shopapi/internal/dto"
)

// ClientService describes the operations used by the HTTP controller.
type ClientService interface {
	CreateClient(ctx context.Context, input dto.ClientCreateDTO) (*dto.ClientDTO, error)
	GetClientByID(ctx context.Context, id uuid.UUID) (*dto.ClientDTO, error)
	GetAllClients(ctx context.Context, limit, offset int) ([]dto.ClientDTO, error)
	GetClientsByNameAndSurname(ctx context.Context, name, surname string) ([]dto.ClientDTO, error)
	UpdateClientAddress(ctx context.Context, id uuid.UUID, input dto.ClientUpdateAddressDTO) error
	DeleteClient(ctx context.Context, id uuid.UUID) error
}

// SupplierService describes the operations used by the HTTP controller.
type SupplierService interface {
	CreateSupplier(ctx context.Context, input dto.SupplierCreateDTO) (*dto.SupplierDTO, error)
	GetSupplierByID(ctx context.Context, id uuid.UUID) (*dto.SupplierDTO, error)
	GetAllSuppliers(ctx context.Context) ([]dto.SupplierDTO, error)
	UpdateSupplierAddress(ctx context.Context, id uuid.UUID, input dto.SupplierUpdateAddressDTO) error
	DeleteSupplier(ctx context.Context, id uuid.UUID) error
}

// ProductService describes the operations used by the HTTP controller.
type ProductService interface {
	CreateProduct(ctx context.Context, input dto.ProductCreateDTO) (*dto.ProductDTO, error)
	GetProductByID(ctx context.Context, id uuid.UUID) (*dto.ProductDTO, error)
	GetAllProducts(ctx context.Context) ([]dto.ProductDTO, error)
	DecreaseStock(ctx context.Context, id uuid.UUID, amount int) error
	DeleteProduct(ctx context.Context, id uuid.UUID) error
}

// ImageService describes the operations used by the HTTP controller.
type ImageService interface {
	UploadImage(ctx context.Context, productID uuid.UUID, data []byte) (uuid.UUID, error)
	GetImageByID(ctx context.Context, id uuid.UUID) ([]byte, error)
	GetImageByProductID(ctx context.Context, productID uuid.UUID) ([]byte, error)
	UpdateImage(ctx context.Context, id uuid.UUID, data []byte) error
	DeleteImage(ctx context.Context, id uuid.UUID) error
}
