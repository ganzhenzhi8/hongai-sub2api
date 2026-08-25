package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/gin-gonic/gin"
)

// registerIntegrationAdminRoutes keeps hongai's remote management calls under
// the normal admin JWT/step-up boundary. The remote eamon endpoints themselves
// use HMAC and are never exposed through the browser directly.
func registerIntegrationAdminRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	integration := admin.Group("/integration/eamon")
	{
		integration.GET("/groups", h.Admin.Account.IntegrationRemoteGroups)
		integration.GET("/accounts", h.Admin.Account.IntegrationRemoteAccounts)
		integration.POST("/accounts", h.Admin.Account.IntegrationPushAccount)
		integration.PUT("/accounts/:id", h.Admin.Account.IntegrationRemoteUpdate)
		integration.DELETE("/accounts/:id", h.Admin.Account.IntegrationRemoteDelete)
		integration.Any("/proxy/*path", h.Admin.Account.IntegrationRemoteAdminProxy)
	}
}
