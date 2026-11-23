package service

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"shopapi/internal/dto"
	"shopapi/internal/repository"
)

type ProductService struct {
	productRepo  repository.ProductRepository
	categoryRepo repository.CategoryRepository
	supplierRepo repository.SupplierRepository
}

func NewProductService(
	productRepo repository.ProductRepository,
	categoryRepo repository.CategoryRepository,
	supplierRepo repository.SupplierRepository,
) *ProductService {
	return &ProductService{
		productRepo:  productRepo,
		categoryRepo: categoryRepo,
		supplierRepo: supplierRepo,
	}
}

func (s *ProductService) CreateProduct(ctx context.Context, input dto.ProductCreateDTO) (*dto.ProductDTO, error) {
	if input.Price == nil || *input.Price < 0 {
		return nil, ErrInvalidInput
	}
	// Проверяем категорию
	if _, err := s.categoryRepo.GetByID(ctx, input.CategoryID); err != nil {
		if err == repository.ErrNotFound {
			return nil, ErrInvalidInput
		}
		return nil, err
	}
	// Проверяем поставщика
	if _, err := s.supplierRepo.GetByID(ctx, input.SupplierID); err != nil {
		if err == repository.ErrNotFound {
			return nil, ErrInvalidInput
		}
		return nil, err
	}

	product := dto.ProductToModel(&input)
	if err := s.productRepo.Create(ctx, &product); err != nil {
		return nil, err
	}

	created, err := s.productRepo.GetByID(ctx, product.ID)
	if err != nil {
		return nil, err
	}
	result := dto.ProductFromModel(created)
	return &result, nil
}

func (s *ProductService) GetProductByID(ctx context.Context, id uuid.UUID) (*dto.ProductDTO, error) {
	product, err := s.productRepo.GetByID(ctx, id)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	result := dto.ProductFromModel(product)
	return &result, nil
}

func (s *ProductService) GetAllProducts(ctx context.Context) ([]dto.ProductDTO, error) {
	products, err := s.productRepo.GetAll(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]dto.ProductDTO, 0, len(products))
	for i := range products {
		result = append(result, dto.ProductFromModel(&products[i]))
	}
	return result, nil
}

// DecreaseStock уменьшает количество товара на складе.
func (s *ProductService) DecreaseStock(ctx context.Context, id uuid.UUID, amount int) error {
	if amount <= 0 {
		return ErrInvalidInput
	}
	err := s.productRepo.DecreaseStock(ctx, id, amount)
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return ErrNotFound
	case errors.Is(err, repository.ErrInsufficientStock):
		return ErrInvalidInput
	default:
		return err
	}
}

func (s *ProductService) DeleteProduct(ctx context.Context, id uuid.UUID) error {
	err := s.productRepo.Delete(ctx, id)
	if err != nil {
		if err == repository.ErrNotFound {
			return ErrNotFound
		}
		return err
	}
	return nil
}
