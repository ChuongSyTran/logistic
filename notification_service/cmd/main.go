package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"notification_service/internal/conf"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found or error reading it")
	}

	cfg, err := conf.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Println("Starting notification_service...")
	app, err := NewApp(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize notification_service: %v", err)
	}

	go func() {
		if err := app.Run(); err != nil {
			log.Fatalf("Failed to run notification_service: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	app.Stop()
}
