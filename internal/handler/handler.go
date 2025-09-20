package handler

import (
	"net/http"
	
	"github.com/gin-gonic/gin"
)

type HTTPHandler struct {
}

func NewHTTPHandler() *HTTPHandler {
	return &HTTPHandler{}
}

func (hdl *HTTPHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}

func (hdl *HTTPHandler) Ingest(c *gin.Context) {

}

func (hdl *HTTPHandler) Chat(c *gin.Context) {

}
