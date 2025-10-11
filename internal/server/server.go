package server

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/katatrina/poke-bot/internal/config"
	"github.com/katatrina/poke-bot/internal/handler"
)

type Server struct {
	config *config.Config
	router *gin.Engine
	hdl    *handler.HTTPHandler
}

func NewServer(cfg *config.Config, hdl *handler.HTTPHandler) *Server {
	router := gin.Default()

	srv := &Server{
		config: cfg,
		router: router,
		hdl:    hdl,
	}

	return srv
}

func (s *Server) SetupRoutes() {
	v1 := s.router.Group("/api/v1")

	v1.GET("/health", s.hdl.HealthCheck)
	v1.POST("/ingest", s.hdl.IngestDoc)
	v1.POST("/chat", s.hdl.Chat)

	s.router.StaticFile("/", "./web/index.html")
}

func (s *Server) Start() error {
	return s.router.Run(fmt.Sprintf(":%d", s.config.Server.Port))
}
