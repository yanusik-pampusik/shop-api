package service

import (
	"context"

	"github.com/google/uuid"

	"shopapi/internal/models"
	"shopapi/internal/repository"
)

type ImageService struct {
	imageRepo   repository.ImageRepository
	productRepo repository.ProductRepository
}

func NewImageService(
	imageRepo repository.ImageRepository,
	productRepo repository.ProductRepository,
) *ImageService {
	return &ImageService{
		imageRepo:   imageRepo,
		productRepo: productRepo,
	}
}

// UploadImage добавляет изображение и привязывает его к товару.
func (s *ImageService) UploadImage(ctx context.Context, productID uuid.UUID, data []byte) (uuid.UUID, error) {
	// Проверяем, что товар существует
	_, err := s.productRepo.GetByID(ctx, productID)
	if err != nil {
		if err == repository.ErrNotFound {
			return uuid.Nil, ErrNotFound
		}
		return uuid.Nil, err
	}

	// Создаём изображение
	image := models.Image{Image: data}
	if err := s.imageRepo.Create(ctx, &image); err != nil {
		return uuid.Nil, err
	}

	// Привязываем к товару
	if err := s.productRepo.SetImage(ctx, productID, image.ID); err != nil {
		if err == repository.ErrNotFound {
			return uuid.Nil, ErrNotFound
		}
		return uuid.Nil, err
	}

	return image.ID, nil
}

// GetImageByID возвращает изображение по ID.
func (s *ImageService) GetImageByID(ctx context.Context, id uuid.UUID) ([]byte, error) {
	image, err := s.imageRepo.GetByID(ctx, id)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return image.Image, nil
}

// GetImageByProductID возвращает изображение конкретного товара.
func (s *ImageService) GetImageByProductID(ctx context.Context, productID uuid.UUID) ([]byte, error) {
	image, err := s.imageRepo.GetByProductID(ctx, productID)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return image.Image, nil
}

// UpdateImage заменяет содержимое изображения.
func (s *ImageService) UpdateImage(ctx context.Context, id uuid.UUID, data []byte) error {
	image, err := s.imageRepo.GetByID(ctx, id)
	if err != nil {
		if err == repository.ErrNotFound {
			return ErrNotFound
		}
		return err
	}
	image.Image = data
	return s.imageRepo.Update(ctx, image)
}

// DeleteImage удаляет изображение и отвязывает его от товара.
func (s *ImageService) DeleteImage(ctx context.Context, id uuid.UUID) error {
	err := s.imageRepo.Delete(ctx, id)
	if err != nil {
		if err == repository.ErrNotFound {
			return ErrNotFound
		}
		return err
	}
	return nil
}
