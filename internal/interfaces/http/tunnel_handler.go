package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/waste3d/ghost-tunnel/internal/application"
	"github.com/waste3d/ghost-tunnel/internal/interfaces/http/middlewares"
)

type TunnelHandler struct {
	tunnelService *application.TunnelService
}

func NewTunnelHandler(tunnelService *application.TunnelService) *TunnelHandler {
	return &TunnelHandler{tunnelService: tunnelService}
}

func (h *TunnelHandler) RegisterRoutes(router *gin.Engine) {

	private := router.Group("/")
	private.Use(middlewares.AuthMiddleware(h.tunnelService.GetUserRepository()))

	{
		// Регистрируем приватные маршруты в этой группе
		private.POST("/tunnels", h.CreateTunnel)
		private.DELETE("/tunnels/:subdomain", h.DeleteTunnel)
	}

	router.GET("/healthz", h.HealthCheck)
}

func (h *TunnelHandler) CreateTunnel(c *gin.Context) {
	var req application.CreateTunnelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, exists := middlewares.GetUserFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	req.UserID = user.ID

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
