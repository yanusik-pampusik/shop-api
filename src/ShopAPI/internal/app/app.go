package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/dig"
	"gorm.io/gorm"
	"shopapi/internal/authclient"
	"shopapi/internal/config"
	"shopapi/internal/handler"
	"shopapi/internal/models"
	"shopapi/internal/repository"
	"shopapi/internal/service"
	authpb "shopapi/proto/auth"
)

// Container supplies repositories, business services and gRPC through interfaces.
func Container(cfg *config.Config, db *gorm.DB, auth authpb.AuthServiceClient) (*dig.Container, error) {
	c := dig.New()
	providers := []any{
		func() *config.Config { return cfg },
		func() *gorm.DB { return db },
		func() authpb.AuthServiceClient { return auth },
		repository.NewClientRepository, repository.NewAddressRepository,
		repository.NewSupplierRepository, repository.NewProductRepository,
		repository.NewImageRepository, repository.NewCategoryRepository,
		handler.NewClientHandler, handler.NewSupplierHandler,
		handler.NewProductHandler, handler.NewImageHandler, handler.NewAuthHandler,
		Router,
	}
	for _, provider := range providers {
		if err := c.Provide(provider); err != nil {
			return nil, err
		}
	}
	for _, provider := range []struct {
		constructor any
		contract    any
	}{
		{service.NewClientService, new(handler.ClientService)},
		{service.NewSupplierService, new(handler.SupplierService)},
		{service.NewProductService, new(handler.ProductService)},
		{service.NewImageService, new(handler.ImageService)},
	} {
		if err := c.Provide(provider.constructor, dig.As(provider.contract)); err != nil {
			return nil, err
		}
	}
	return c, nil
}

func Run(ctx context.Context, cfg *config.Config) error {
	db, err := config.ConnectDB(cfg)
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	defer sqlDB.Close()
	if cfg.Migrate {
		startup, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		if err := db.WithContext(startup).AutoMigrate(
			&models.Address{}, &models.Category{}, &models.Supplier{},
			&models.Image{}, &models.Product{}, &models.Client{},
		); err != nil {
			return fmt.Errorf("migrate shop database: %w", err)
		}
	}
	conn, err := authclient.Connect(cfg)
	if err != nil {
		return fmt.Errorf("configure auth connection: %w", err)
	}
	defer conn.Close()
	container, err := Container(cfg, db, authpb.NewAuthServiceClient(conn))
	if err != nil {
		return fmt.Errorf("configure dependencies: %w", err)
	}
	return container.Invoke(func(router *gin.Engine) error {
		server := &http.Server{Addr: cfg.HTTPAddress, Handler: router,
			ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second,
			WriteTimeout: 30 * time.Second, IdleTimeout: time.Minute}
		result := make(chan error, 1)
		go func() { result <- server.ListenAndServe() }()
		log.Printf("ShopAPI listening on %s", cfg.HTTPAddress)
		select {
		case err := <-result:
			return err
		case <-ctx.Done():
			shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := server.Shutdown(shutdown); err != nil {
				_ = server.Close()
				return err
			}
			err := <-result
			if errors.Is(err, http.ErrServerClosed) {
				return nil
			}
			return err
		}
	})
}
