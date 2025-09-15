package cmd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	
	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	
	"github.com/katatrina/go-rag-chatbot/internal/modules/chat"
	"github.com/katatrina/go-rag-chatbot/internal/modules/ingest"
	"github.com/katatrina/go-rag-chatbot/internal/shared/config"
	"github.com/katatrina/go-rag-chatbot/internal/shared/logger"
)

var (
	cfgFile string
	loadKB  bool
)

var rootCmd = &cobra.Command{
	Use:   "go-rag-chatbot",
	Short: "A RAG chatbot server built with Go",
	Long: `A Retrieval-Augmented Generation chatbot server that can ingest documents,
create embeddings, and provide intelligent responses using vector similarity search.`,
	Run: runServer,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./configs/app-config.yaml)")
	rootCmd.PersistentFlags().BoolVar(&loadKB, "load-kb", false, "automatically load knowledge base on startup")
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath("./configs")
		viper.SetConfigType("yaml")
		viper.SetConfigName("app-config")
	}
	
	viper.AutomaticEnv()
	
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

func runServer(cmd *cobra.Command, args []string) {
	// Initialize logger
	logger.Init()
	
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}
	
	// Initialize modules
	ingestModule := ingest.NewModule(cfg)
	chatModule := chat.NewModule(cfg)
	
	// Setup Gin router
	router := gin.Default()
	
	// Serve static files
	router.Static("/static", "./web/static")
	router.LoadHTMLGlob("web/*.html")
	
	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})
	
	// Register module routes
	ingestModule.RegisterRoutes(router)
	chatModule.RegisterRoutes(router)
	
	// Serve main page
	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})
	
	// Auto-load knowledge base if flag is set
	if loadKB {
		slog.Info("Auto-loading knowledge base...")
		if err := ingestModule.AutoLoadKB(context.Background()); err != nil {
			slog.Error("Failed to auto-load knowledge base", "error", err)
		} else {
			slog.Info("Knowledge base loaded successfully")
		}
	}
	
	// Setup HTTP server with graceful shutdown
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: router,
	}
	
	go func() {
		slog.Info("Starting server", "port", cfg.Server.Port)
		if err = srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Failed to start server", "error", err)
			os.Exit(1)
		}
	}()
	
	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("Shutting down server...")
	
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err = srv.Shutdown(ctx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
	}
	
	slog.Info("Server exited")
}
