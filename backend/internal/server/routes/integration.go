package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/gin-gonic/gin"
)

// RegisterIntegrationRoutes exposes the receiving-side integration API.
// Every handler performs its own environment gate and HMAC verification, so
// these endpoints remain inert until SUB2_INTEGRATION_ENABLED=true with valid
// credentials configured on the receiving site.
func RegisterIntegrationRoutes(v1 *gin.RouterGroup, h *handler.Handlers) {
	integration := v1.Group("/integration")
	{
		integration.GET("/health", h.Admin.Account.IntegrationHealth)
		integration.GET("/groups", h.Admin.Account.IntegrationGroups)
		integration.GET("/accounts", h.Admin.Account.IntegrationAccounts)
		integration.POST("/accounts", h.Admin.Account.IntegrationCreateAccount)
		integration.PUT("/accounts/:id", h.Admin.Account.IntegrationUpdateAccount)
		integration.DELETE("/accounts/:id", h.Admin.Account.IntegrationDeleteAccount)
	}
}
