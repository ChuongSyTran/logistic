package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"gateway_service/internal/conf"
	"gateway_service/internal/di"

	"github.com/logistic/pkg/tracer"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type App struct {
	engine    *gin.Engine
	container *di.Container
	server    *http.Server
	shutdown  func(context.Context) error
	cfg       *conf.Config
}

func NewApp() (*App, error) {
	cfg, err := conf.LoadConfig()
	if err != nil {
		return nil, err
	}

	shutdownTracer, err := tracer.InitTracer("gateway_service", os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"))
	if err != nil {
		log.Printf("Failed to initialize tracer: %v", err)
	}

	ginEngine := gin.New()

	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = cfg.CORS.AllowOrigins
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	corsConfig.AllowHeaders = []string{"Origin", "Content-Type", "Authorization", "X-Request-ID"}
	corsConfig.ExposeHeaders = []string{"Content-Length", "X-Request-ID"}
	corsConfig.AllowCredentials = true
	corsConfig.MaxAge = 12 * time.Hour
	ginEngine.Use(cors.New(corsConfig))

	container, err := di.Injection(ginEngine, cfg)
	if err != nil {
		return nil, err
	}

	return &App{
		engine:    ginEngine,
		container: container,
		shutdown:  shutdownTracer,
		cfg:       cfg,
	}, nil
}

func (a *App) Start() error {
	a.server = &http.Server{
		Addr:         ":" + a.cfg.Server.Port,
		Handler:      a.engine,
		ReadTimeout:  a.cfg.Server.ReadTimeout,
		WriteTimeout: a.cfg.Server.WriteTimeout,
		IdleTimeout:  a.cfg.Server.IdleTimeout,
	}

	log.Printf("Starting Gateway Service on :%s...", a.cfg.Server.Port)

	go func() {
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Gateway Service crashed: %v", err)
		}
	}()
	return nil
}

func (a *App) Stop() {
	log.Println("Stopping Gateway Service gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := a.server.Shutdown(ctx); err != nil {
		log.Printf("Gateway Service forced to shutdown: %v", err)
	}

	a.container.Close()

	if a.shutdown != nil {
		if err := a.shutdown(ctx); err != nil {
			log.Printf("Failed to shutdown tracer: %v", err)
		}
	}

	log.Println("Gateway Service stopped.")
}