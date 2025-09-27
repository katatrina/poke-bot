package main

import (
	"log"
	
	"github.com/katatrina/poke-bot/internal/config"
	"github.com/katatrina/poke-bot/internal/handler"
	"github.com/katatrina/poke-bot/internal/repository"
	"github.com/katatrina/poke-bot/internal/server"
	"github.com/katatrina/poke-bot/internal/service"
	"github.com/qdrant/go-client/qdrant"
	"resty.dev/v3"
)

func main() {
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatal("failed to load config:", err)
	}
	
	qdrantClient, err := qdrant.NewClient(&qdrant.Config{
		Host: cfg.Qdrant.Host,
		Port: cfg.Qdrant.Port,
	})
	if err != nil {
		log.Fatalf("failed to connect to Qdrant: %v", err)
	}
	
	vectorRepo, err := repository.NewVectorRepository(cfg, qdrantClient)
	if err != nil {
		log.Fatalf("failed to create repository: %s", err)
	}
	
	restyClient := resty.New()
	defer restyClient.Close()
	
	ragService := service.NewRAGService(cfg, vectorRepo, restyClient)
	
	hdl := handler.NewHTTPHandler(ragService)
	
	srv := server.NewServer(cfg, hdl)
	srv.SetupRoutes()
	
	if err = srv.Start(); err != nil {
		log.Fatalf("failed to start HTTP server: %v", err)
	}
}
