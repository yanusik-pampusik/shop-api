package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"shopapi/internal/models"
)

// replaceAddress keeps the new address and its owner link in one transaction.
// Locking the owner also prevents deletion between the lookup and the update.
func replaceAddress(ctx context.Context, db *gorm.DB, owner any, id uuid.UUID, address *models.Address) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Select("id").Clauses(clause.Locking{Strength: "UPDATE"}).Take(owner, "id = ?", id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}
		if err := tx.Create(address).Error; err != nil {
			return err
		}
		// Update only the foreign key; never save the previously loaded association.
		return tx.Model(owner).Omit(clause.Associations).Update("address_id", address.ID).Error
	})
}
