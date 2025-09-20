package server

import (
	"fmt"
	
	"github.com/gin-gonic/gin"
	"github.com/katatrina/poke-bot/internal/config"
	"github.com/katatrina/poke-bot/internal/handler"
)

type HTTPServer struct {
	config *config.Config
	router *gin.Engine
	hdl    *handler.HTTPHandler
}

func NewHTTPServer(cfg *config.Config, hdl *handler.HTTPHandler) *HTTPServer {
	router := gin.Default()
	
	srv := &HTTPServer{
		config: cfg,
		router: router,
		hdl:    hdl,
	}
	
	return srv
}

func (s *HTTPServer) SetupRoutes() {
	v1 := s.router.Group("/api/v1")
	
	v1.GET("/health", s.hdl.HealthCheck)
	v1.POST("/ingest", s.hdl.IngestDoc)
	v1.POST("/chat", s.hdl.Chat)
}

func (s *HTTPServer) Start() error {
	return s.router.Run(fmt.Sprintf(":%d", s.config.HTTPServer.Port))
}
