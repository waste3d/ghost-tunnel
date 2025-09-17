package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/waste3d/ghost-tunnel/internal/application"
)

type TunnelHandler struct {
	tunnelService *application.TunnelService
}

func NewTunnelHandler(tunnelService *application.TunnelService) *TunnelHandler {
	return &TunnelHandler{tunnelService: tunnelService}
}

func (h *TunnelHandler) RegisterRoutes(router *gin.Engine) {
	router.POST("/tunnels", h.CreateTunnel)
	router.DELETE("/tunnels/:subdomain", h.DeleteTunnel)
	router.GET("/health", h.HealthCheck)
}

func (h *TunnelHandler) CreateTunnel(c *gin.Context) {
	var req application.CreateTunnelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tunnel, err := h.tunnelService.CreateTunnel(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, tunnel)
}

func (h *TunnelHandler) DeleteTunnel(c *gin.Context) {
	subdomain := c.Param("subdomain")
	if subdomain == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "subdomain is required"})
		return
	}

	err := h.tunnelService.DeleteTunnel(c.Request.Context(), subdomain)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Tunnel deleted successfully"})
}

func (h *TunnelHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "OK"})
}
