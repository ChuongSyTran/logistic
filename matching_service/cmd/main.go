package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"matching_service/internal/conf"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load("../.env"); err != nil {
		log.Println("No .env file found or failed to load, falling back to system environment variables")
	}

	cfg, err := conf.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	app, err := NewApp(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize Matching App: %v", err)
	}

	go func() {
		log.Printf("Starting Matching Service on :%s...", cfg.Server.GrpcPort)
		if err := app.Start(); err != nil {
			log.Fatalf("Matching Service crashed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Stopping Matching Service...")
	app.Stop()
}