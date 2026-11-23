package dto

import "github.com/google/uuid"

// --- Address ---

type AddressDTO struct {
	ID      uuid.UUID `json:"id"`
	Country string    `json:"country" binding:"required"`
	City    string    `json:"city" binding:"required"`
	Street  string    `json:"street" binding:"required"`
}

// AddressCreateDTO — для создания адреса (без id).
type AddressCreateDTO struct {
	Country string `json:"country" binding:"required"`
	City    string `json:"city" binding:"required"`
	Street  string `json:"street" binding:"required"`
}

// --- Category ---

type CategoryDTO struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name" binding:"required"`
}

// --- Client ---

// ClientCreateDTO — то, что приходит в POST /clients.
type ClientCreateDTO struct {
	ClientName    string    `json:"client_name" binding:"required"`
	ClientSurname string    `json:"client_surname" binding:"required"`
	Birthday      string    `json:"birthday" binding:"required"` // формат "2006-01-02"
	Gender        string    `json:"gender" binding:"required,oneof=M F"`
	AddressID     uuid.UUID `json:"address_id" binding:"required"`
}

// ClientDTO — то, что уходит в ответе.
type ClientDTO struct {
	ID               uuid.UUID  `json:"id"`
	ClientName       string     `json:"client_name"`
	ClientSurname    string     `json:"client_surname"`
	Birthday         string     `json:"birthday"`
	Gender           string     `json:"gender"`
	RegistrationDate string     `json:"registration_date"`
	AddressID        uuid.UUID  `json:"address_id"`
	Address          AddressDTO `json:"address"`
}

// ClientUpdateAddressDTO — для PATCH /clients/:id/address.
type ClientUpdateAddressDTO struct {
	Address AddressCreateDTO `json:"address" binding:"required"`
}

// --- Supplier ---

type SupplierCreateDTO struct {
	Name        string    `json:"name" binding:"required"`
	AddressID   uuid.UUID `json:"address_id" binding:"required"`
	PhoneNumber string    `json:"phone_number" binding:"required"`
}

type SupplierDTO struct {
	ID          uuid.UUID  `json:"id"`
	Name        string     `json:"name"`
	AddressID   uuid.UUID  `json:"address_id"`
	Address     AddressDTO `json:"address"`
	PhoneNumber string     `json:"phone_number"`
}

type SupplierUpdateAddressDTO struct {
	Address AddressCreateDTO `json:"address" binding:"required"`
}

// --- Product ---

type ProductCreateDTO struct {
	Name           string    `json:"name" binding:"required"`
	CategoryID     uuid.UUID `json:"category_id" binding:"required"`
	Price          *float64  `json:"price" binding:"required,gte=0"`
	AvailableStock int       `json:"available_stock" binding:"gte=0"`
	SupplierID     uuid.UUID `json:"supplier_id" binding:"required"`
}

type ProductDTO struct {
	ID             uuid.UUID   `json:"id"`
	Name           string      `json:"name"`
	CategoryID     uuid.UUID   `json:"category_id"`
	Category       CategoryDTO `json:"category"`
	Price          float64     `json:"price"`
	AvailableStock int         `json:"available_stock"`
	LastUpdateDate string      `json:"last_update_date"`
	SupplierID     uuid.UUID   `json:"supplier_id"`
	Supplier       SupplierDTO `json:"supplier"`
	ImageID        *uuid.UUID  `json:"image_id,omitempty"`
}

// ProductDecreaseStockDTO — для PATCH /products/:id/decrease.
type ProductDecreaseStockDTO struct {
	Amount int `json:"amount" binding:"required,gt=0"`
}

// --- Image ---

type ImageUploadDTO struct {
	ProductID uuid.UUID `json:"product_id" binding:"required"`
}
