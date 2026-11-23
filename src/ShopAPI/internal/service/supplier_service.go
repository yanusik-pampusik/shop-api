package service

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"shopapi/internal/dto"
	"shopapi/internal/repository"
)

type SupplierService struct {
	supplierRepo repository.SupplierRepository
	addressRepo  repository.AddressRepository
}

func NewSupplierService(
	supplierRepo repository.SupplierRepository,
	addressRepo repository.AddressRepository,
) *SupplierService {
	return &SupplierService{
		supplierRepo: supplierRepo,
		addressRepo:  addressRepo,
	}
}

func (s *SupplierService) CreateSupplier(ctx context.Context, input dto.SupplierCreateDTO) (*dto.SupplierDTO, error) {
	// Проверяем, что адрес существует
	if _, err := s.addressRepo.GetByID(ctx, input.AddressID); err != nil {
		if err == repository.ErrNotFound {
			return nil, ErrInvalidInput
		}
		return nil, err
	}

	supplier := dto.SupplierToModel(&input)
	if err := s.supplierRepo.Create(ctx, &supplier); err != nil {
		return nil, err
	}

	created, err := s.supplierRepo.GetByID(ctx, supplier.ID)
	if err != nil {
		return nil, err
	}

	result := dto.SupplierFromModel(created)
	return &result, nil
}

func (s *SupplierService) GetSupplierByID(ctx context.Context, id uuid.UUID) (*dto.SupplierDTO, error) {
	supplier, err := s.supplierRepo.GetByID(ctx, id)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	result := dto.SupplierFromModel(supplier)
	return &result, nil
}

func (s *SupplierService) GetAllSuppliers(ctx context.Context) ([]dto.SupplierDTO, error) {
	suppliers, err := s.supplierRepo.GetAll(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]dto.SupplierDTO, 0, len(suppliers))
	for i := range suppliers {
		result = append(result, dto.SupplierFromModel(&suppliers[i]))
	}
	return result, nil
}

func (s *SupplierService) UpdateSupplierAddress(ctx context.Context, id uuid.UUID, input dto.SupplierUpdateAddressDTO) error {
	address := dto.AddressToModel(&input.Address)
	err := s.supplierRepo.UpdateAddress(ctx, id, &address)
	if errors.Is(err, repository.ErrNotFound) {
		return ErrNotFound
	}
	return err
}

func (s *SupplierService) DeleteSupplier(ctx context.Context, id uuid.UUID) error {
	err := s.supplierRepo.Delete(ctx, id)
	if errors.Is(err, repository.ErrConflict) {
		return ErrConflict
	}
	if err != nil {
		if err == repository.ErrNotFound {
			return ErrNotFound
		}
		return err
	}
	return nil
}
