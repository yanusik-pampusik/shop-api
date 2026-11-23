package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"shopapi/internal/app"
	"shopapi/internal/config"
	"syscall"
)

// @title ShopAPI
// @version 2.0
// @description Appliance store with gRPC authentication. Use /register or /auth, then Authorize with Bearer <token>.
// @BasePath /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := app.Run(ctx, config.LoadConfig()); err != nil {
		log.Fatal(err)
	}
}
