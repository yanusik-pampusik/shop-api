package repository_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"shopapi/internal/models"
	"shopapi/internal/repository"
)

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("SHOP_TEST_DB_DSN")
	if dsn == "" {
		t.Skip("set SHOP_TEST_DB_DSN to a separate PostgreSQL test database")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&models.Address{}, &models.Category{}, &models.Supplier{}, &models.Image{}, &models.Product{}, &models.Client{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func create(t *testing.T, db *gorm.DB, value any) {
	t.Helper()
	if err := db.Create(value).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := db.Delete(value).Error; err != nil {
			t.Error(err)
		}
	})
}

func TestAddressReplacementAndRollback(t *testing.T) {
	for _, kind := range []string{"client", "supplier"} {
		t.Run(kind, func(t *testing.T) {
			db := testDB(t)
			ctx := context.Background()
			old := models.Address{Country: "RU", City: "Old", Street: "Old"}
			create(t, db, &old)
			var owner any
			var id uuid.UUID
			var replace func(context.Context, uuid.UUID, *models.Address) error
			if kind == "client" {
				client := &models.Client{ClientName: "Test", ClientSurname: "Test", Birthday: time.Now(), Gender: "M", AddressID: old.ID}
				create(t, db, client)
				owner, id, replace = client, client.ID, repository.NewClientRepository(db).UpdateAddress
			} else {
				supplier := &models.Supplier{Name: "Test", PhoneNumber: "123", AddressID: old.ID}
				create(t, db, supplier)
				owner, id, replace = supplier, supplier.ID, repository.NewSupplierRepository(db).UpdateAddress
			}
			address := models.Address{Country: "RU", City: "New", Street: "New"}
			// Cleanup must remove the owner before the new referenced address.
			t.Cleanup(func() { db.Delete(owner); db.Delete(&address) })
			if err := replace(ctx, id, &address); err != nil {
				t.Fatal(err)
			}
			var actual struct{ AddressID uuid.UUID }
			if err := db.Model(owner).Select("address_id").Where("id = ?", id).Scan(&actual).Error; err != nil {
				t.Fatal(err)
			}
			if actual.AddressID != address.ID {
				t.Fatal("new address was not linked")
			}
			missing := models.Address{Country: "RU", City: "Missing", Street: "Missing"}
			if err := replace(ctx, uuid.New(), &missing); !errors.Is(err, repository.ErrNotFound) {
				t.Fatalf("missing owner: %v", err)
			}
			if missing.ID != uuid.Nil {
				t.Fatal("address created for missing owner")
			}

			failure := errors.New("injected owner update failure")
			if err := db.Callback().Update().Before("gorm:update").Register("test:reject_address_update", func(tx *gorm.DB) { tx.AddError(failure) }); err != nil {
				t.Fatal(err)
			}
			defer db.Callback().Update().Remove("test:reject_address_update")
			rolledBack := models.Address{Country: "RU", City: "Rollback", Street: "Rollback"}
			if err := replace(ctx, id, &rolledBack); !errors.Is(err, failure) {
				t.Fatalf("update failure: %v", err)
			}
			var count int64
			if err := db.Model(&models.Address{}).Where("id = ?", rolledBack.ID).Count(&count).Error; err != nil {
				t.Fatal(err)
			}
			if count != 0 {
				t.Fatal("failed update left an orphan address")
			}
			if err := db.Model(owner).Select("address_id").Where("id = ?", id).Scan(&actual).Error; err != nil {
				t.Fatal(err)
			}
			if actual.AddressID != address.ID {
				t.Fatal("failed update changed the owner link")
			}
		})
	}
}

func TestConcurrentStockAndSupplierConflict(t *testing.T) {
	db := testDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	address := models.Address{Country: "RU", City: "Test", Street: "Test"}
	create(t, db, &address)
	category := models.Category{Name: uuid.NewString()}
	create(t, db, &category)
	supplier := models.Supplier{Name: "Test", PhoneNumber: "123", AddressID: address.ID}
	create(t, db, &supplier)
	image := models.Image{Image: []byte("test")}
	create(t, db, &image)
	product := models.Product{Name: "Test", CategoryID: category.ID, SupplierID: supplier.ID, Price: 1, AvailableStock: 10}
	create(t, db, &product)
	products := repository.NewProductRepository(db)
	suppliers := repository.NewSupplierRepository(db)
	if err := suppliers.Delete(ctx, supplier.ID); !errors.Is(err, repository.ErrConflict) {
		t.Fatalf("foreign-key conflict: %v", err)
	}
	start := make(chan struct{})
	results := make(chan error, 24)
	for i := 0; i < 24; i++ {
		go func() { <-start; results <- products.DecreaseStock(ctx, product.ID, 1) }()
	}
	close(start)
	success := 0
	for i := 0; i < 24; i++ {
		err := <-results
		if err == nil {
			success++
		} else if !errors.Is(err, repository.ErrInsufficientStock) {
			t.Fatal(err)
		}
	}
	if success != 10 {
		t.Fatalf("expected 10 successful deductions, got %d", success)
	}
	// Linking an image must not restore the stale stock from a previously read product.
	if err := products.SetImage(ctx, product.ID, image.ID); err != nil {
		t.Fatal(err)
	}
	saved, err := products.GetByID(ctx, product.ID)
	if err != nil {
		t.Fatal(err)
	}
	if saved.AvailableStock != 0 || saved.ImageID == nil || *saved.ImageID != image.ID {
		t.Fatalf("incorrect saved product: %+v", saved)
	}
	if err := products.DecreaseStock(ctx, uuid.New(), 1); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("missing product: %v", err)
	}
	if err := products.Delete(ctx, product.ID); err != nil {
		t.Fatal(err)
	}
	if err := suppliers.Delete(ctx, supplier.ID); err != nil {
		t.Fatal(err)
	}
}
