package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"time"

	"authservice/internal/config"
	"authservice/internal/models"
	"authservice/internal/repository"
	"authservice/internal/service"
	"authservice/internal/token"
	"authservice/internal/transport"
	authpb "authservice/proto/auth"

	"go.uber.org/dig"
	"google.golang.org/grpc"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Container wires implementations through interfaces; constructors do not read env.
func Container(cfg config.Config, db *gorm.DB) (*dig.Container, error) {
	c := dig.New()
	providers := []any{
		func() config.Config { return cfg },
		func() *gorm.DB { return db },
		repository.NewUserRepository,
		func(repo repository.UserRepository) transport.Users { return service.NewAuthService(repo) },
		func(cfg config.Config) (token.Manager, error) {
			return token.New(cfg.JWTSecret, cfg.JWTIssuer, cfg.JWTAudience, cfg.TokenTTL)
		},
		transport.New,
		func(handler *transport.Server) *grpc.Server {
			server := grpc.NewServer(grpc.MaxRecvMsgSize(64 * 1024))
			authpb.RegisterAuthServiceServer(server, handler)
			return server
		},
	}
	for _, provider := range providers {
		if err := c.Provide(provider); err != nil {
			return nil, err
		}
	}
	return c, nil
}

func Run(ctx context.Context, cfg config.Config) error {
	db, err := gorm.Open(postgres.Open(cfg.DatabaseDSN), &gorm.Config{
		TranslateError:       true,
		DisableAutomaticPing: true,
		// User operations contain credentials: do not log SQL parameter values.
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return fmt.Errorf("open auth database: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("get auth database connection: %w", err)
	}
	defer sqlDB.Close()
	startup, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(startup); err != nil {
		return fmt.Errorf("connect auth database: %w", err)
	}
	if err := db.WithContext(startup).AutoMigrate(&models.User{}); err != nil {
		return fmt.Errorf("migrate auth database: %w", err)
	}
	container, err := Container(cfg, db)
	if err != nil {
		return fmt.Errorf("configure dependencies: %w", err)
	}
	return container.Invoke(func(server *grpc.Server) error {
		listener, err := net.Listen("tcp", cfg.GRPCAddress)
		if err != nil {
			return fmt.Errorf("listen for gRPC: %w", err)
		}
		defer listener.Close()
		log.Printf("AuthService gRPC listening on %s (HTTP/2)", listener.Addr())
		return Serve(ctx, server, listener)
	})
}

// Serve drains active RPCs on shutdown, with a bounded wait.
func Serve(ctx context.Context, server *grpc.Server, listener net.Listener) error {
	result := make(chan error, 1)
	go func() { result <- server.Serve(listener) }()
	select {
	case err := <-result:
		if errors.Is(err, grpc.ErrServerStopped) {
			return nil
		}
		return err
	case <-ctx.Done():
		stopped := make(chan struct{})
		go func() { server.GracefulStop(); close(stopped) }()
		timer := time.NewTimer(5 * time.Second)
		defer timer.Stop()
		select {
		case <-stopped:
		case <-timer.C:
			server.Stop()
			<-stopped
		}
		err := <-result
		if errors.Is(err, grpc.ErrServerStopped) {
			return nil
		}
		return err
	}
}
