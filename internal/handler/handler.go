package handler

import (
	"net/http"
	
	"github.com/gin-gonic/gin"
	"github.com/katatrina/poke-bot/internal/service"
)

type HTTPHandler struct {
	ragService *service.RAGService
}

func NewHTTPHandler(ragService *service.RAGService) *HTTPHandler {
	return &HTTPHandler{
		ragService: ragService,
	}
}

func (hdl *HTTPHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}

func (hdl *HTTPHandler) IngestDoc(c *gin.Context) {
	var req service.IngestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request format",
			"details": err.Error(),
		})
		return
	}
	
	if err := req.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	
	if err := hdl.ragService.IngestPokemonData(c.Request.Context(), &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to ingest document",
			"details": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message": "document ingested successfully",
	})
}

func (hdl *HTTPHandler) Chat(c *gin.Context) {
	var req service.ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request format",
			"details": err.Error(),
		})
		return
	}
	
	if err := req.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	
	// Process the chat request
	resp, err := hdl.ragService.Chat(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to process chat request",
			"details": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, resp)
}
