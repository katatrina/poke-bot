package main

import (
	"log"
	
	"github.com/katatrina/poke-bot/internal/config"
	"github.com/katatrina/poke-bot/internal/server"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatal("failed to load config:", err)
	}
	
	// Create and start server
	srv := server.NewHTTPServer(cfg)
	srv.SetupRoutes()
	
	if err = srv.Start(); err != nil {
		log.Fatalf("failed to start HTTP server: %v", err)
	}
}
