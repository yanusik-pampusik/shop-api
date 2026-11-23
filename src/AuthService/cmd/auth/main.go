package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"authservice/internal/app"
	"authservice/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if err := app.Run(ctx, cfg); err != nil {
		log.Fatal(err)
	}
}
