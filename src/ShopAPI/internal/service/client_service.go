package service

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"shopapi/internal/dto"
	"shopapi/internal/repository"
)

type ClientService struct {
	clientRepo  repository.ClientRepository
	addressRepo repository.AddressRepository
}

func NewClientService(
	clientRepo repository.ClientRepository,
	addressRepo repository.AddressRepository,
) *ClientService {
	return &ClientService{
		clientRepo:  clientRepo,
		addressRepo: addressRepo,
	}
}

// CreateClient создаёт нового клиента.
func (s *ClientService) CreateClient(ctx context.Context, input dto.ClientCreateDTO) (*dto.ClientDTO, error) {
	// 1. Проверяем, что адрес существует
	if _, err := s.addressRepo.GetByID(ctx, input.AddressID); err != nil {
		if err == repository.ErrNotFound {
			return nil, ErrInvalidInput
		}
		return nil, err
	}

	// 2. Преобразуем DTO в модель
	client, err := dto.ClientToModel(&input)
	if err != nil {
		return nil, ErrInvalidInput
	}

	// 3. Сохраняем в БД
	if err := s.clientRepo.Create(ctx, &client); err != nil {
		return nil, err
	}

	// 4. Подтягиваем адрес для ответа
	created, err := s.clientRepo.GetByID(ctx, client.ID)
	if err != nil {
		return nil, err
	}

	// 5. Преобразуем в DTO
	result := dto.ClientFromModel(created)
	return &result, nil
}

// GetClientByID возвращает клиента по ID.
func (s *ClientService) GetClientByID(ctx context.Context, id uuid.UUID) (*dto.ClientDTO, error) {
	client, err := s.clientRepo.GetByID(ctx, id)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	result := dto.ClientFromModel(client)
	return &result, nil
}

// GetAllClients возвращает список клиентов с пагинацией.
func (s *ClientService) GetAllClients(ctx context.Context, limit, offset int) ([]dto.ClientDTO, error) {
	clients, err := s.clientRepo.GetAll(ctx, limit, offset)
	if err != nil {
		return nil, err
	}
	// Важно: пустой список возвращаем как пустой массив, а не nil
	result := make([]dto.ClientDTO, 0, len(clients))
	for i := range clients {
		result = append(result, dto.ClientFromModel(&clients[i]))
	}
	return result, nil
}

// GetClientsByNameAndSurname ищет клиентов по имени и фамилии.
func (s *ClientService) GetClientsByNameAndSurname(ctx context.Context, name, surname string) ([]dto.ClientDTO, error) {
	clients, err := s.clientRepo.GetByNameAndSurname(ctx, name, surname)
	if err != nil {
		return nil, err
	}
	result := make([]dto.ClientDTO, 0, len(clients))
	for i := range clients {
		result = append(result, dto.ClientFromModel(&clients[i]))
	}
	return result, nil
}

// UpdateClientAddress меняет адрес клиента.
func (s *ClientService) UpdateClientAddress(ctx context.Context, id uuid.UUID, input dto.ClientUpdateAddressDTO) error {
	address := dto.AddressToModel(&input.Address)
	err := s.clientRepo.UpdateAddress(ctx, id, &address)
	if errors.Is(err, repository.ErrNotFound) {
		return ErrNotFound
	}
	return err
}

// DeleteClient удаляет клиента по ID.
func (s *ClientService) DeleteClient(ctx context.Context, id uuid.UUID) error {
	err := s.clientRepo.Delete(ctx, id)
	if err != nil {
		if err == repository.ErrNotFound {
			return ErrNotFound
		}
		return err
	}
	return nil
}
