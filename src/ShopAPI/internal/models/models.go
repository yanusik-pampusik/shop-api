package models

import (
	"time"

	"github.com/google/uuid"
)

// Address — адрес клиента или поставщика.
type Address struct {
	ID      uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Country string    `gorm:"type:varchar(100);not null" json:"country"`
	City    string    `gorm:"type:varchar(100);not null" json:"city"`
	Street  string    `gorm:"type:varchar(200);not null" json:"street"`
}

// Category — категория товара.
type Category struct {
	ID   uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name string    `gorm:"type:varchar(100);not null;unique" json:"name"`
}

// Supplier — поставщик товаров.
type Supplier struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name        string    `gorm:"type:varchar(200);not null" json:"name"`
	AddressID   uuid.UUID `gorm:"type:uuid;not null" json:"address_id"`
	Address     Address   `gorm:"foreignKey:AddressID;constraint:OnDelete:RESTRICT" json:"address"`
	PhoneNumber string    `gorm:"type:varchar(20);not null" json:"phone_number"`
}

// Image — изображение товара.
type Image struct {
	ID    uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Image []byte    `gorm:"type:bytea;not null" json:"-"`
}

// Product — товар магазина.
type Product struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name           string    `gorm:"type:varchar(200);not null" json:"name"`
	CategoryID     uuid.UUID `gorm:"type:uuid;not null" json:"category_id"`
	Category       Category  `gorm:"foreignKey:CategoryID;constraint:OnDelete:RESTRICT" json:"category"`
	Price          float64   `gorm:"type:numeric(10,2);not null;check:price >= 0" json:"price"`
	AvailableStock int       `gorm:"not null;check:available_stock >= 0" json:"available_stock"`
	LastUpdateDate time.Time `gorm:"not null;default:now()" json:"last_update_date"`
	SupplierID     uuid.UUID `gorm:"type:uuid;not null" json:"supplier_id"`
	Supplier       Supplier  `gorm:"foreignKey:SupplierID;constraint:OnDelete:RESTRICT" json:"supplier"`
	ImageID        *uuid.UUID `gorm:"type:uuid" json:"image_id,omitempty"`
	Image          *Image    `gorm:"foreignKey:ImageID;constraint:OnDelete:SET NULL" json:"-"`
}

// Client — клиент магазина.
type Client struct {
	ID               uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ClientName       string    `gorm:"type:varchar(100);not null" json:"client_name"`
	ClientSurname    string    `gorm:"type:varchar(100);not null" json:"client_surname"`
	Birthday         time.Time `gorm:"type:date;not null" json:"birthday"`
	Gender           string    `gorm:"type:char(1);not null;check:gender IN ('M','F')" json:"gender"`
	RegistrationDate time.Time `gorm:"not null;default:now()" json:"registration_date"`
	AddressID        uuid.UUID `gorm:"type:uuid;not null" json:"address_id"`
	Address          Address   `gorm:"foreignKey:AddressID;constraint:OnDelete:RESTRICT" json:"address"`
}