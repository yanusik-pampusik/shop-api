package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Email        string    `gorm:"type:varchar(100);unique;not null"`
	FirstName    string    `gorm:"type:varchar(100);not null"`
	LastName     string    `gorm:"type:varchar(100);not null"`
	PhoneNumber  string    `gorm:"type:varchar(20);not null"`
	PasswordHash string    `gorm:"type:varchar(255);not null"` // Здесь будет bcrypt хэш
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
