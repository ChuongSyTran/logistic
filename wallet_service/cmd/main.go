package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"wallet_service/internal/conf"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg, err := conf.LoadConfig()
	if err != nil {
		log.Fatalf("[Wallet] Failed to load config: %v", err)
	}

	app, err := NewApp(ctx, cfg)
	if err != nil {
		log.Fatalf("[Wallet] Failed to initialize app: %v", err)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("[Wallet] Shutdown signal received")
		cancel()
		app.Stop()
	}()

	if err := app.Start(); err != nil {
		log.Fatalf("[Wallet] Server error: %v", err)
	}
}
