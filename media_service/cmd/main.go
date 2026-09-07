package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"media_service/internal/conf"
)

func main() {
	cfg, err := conf.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	app, err := NewApp(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize Media App: %v", err)
	}

	go func() {
		log.Printf("Starting Media Service on :%s...", cfg.Server.GrpcPort)
		if err := app.Start(); err != nil {
			log.Fatalf("Media Service crashed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Stopping Media Service...")
	app.Stop()
}