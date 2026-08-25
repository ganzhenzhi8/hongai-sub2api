package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
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

	// Reuse the original Sub2API account handlers behind HMAC authentication.
	// This keeps the remote page behavior aligned with the original account
	// manager without opening unrelated admin areas such as users or payments.
	remoteAdmin := integration.Group("/admin")
	remoteAdmin.Use(h.Admin.Account.IntegrationAuth)
	denyCredentialExport := middleware.StepUpAuthMiddleware(func(c *gin.Context) {
		c.AbortWithStatusJSON(403, gin.H{"error": "remote credential export is disabled"})
	})
	registerAccountRoutes(remoteAdmin, h, denyCredentialExport)
	remoteAdmin.GET("/groups/all", h.Admin.Group.GetAll)
	remoteAdmin.GET("/proxies/all", h.Admin.Proxy.GetAll)
	registerScheduledTestRoutes(remoteAdmin, h)
	registerGrokOAuthRoutes(remoteAdmin, h)
	registerCNProviderRoutes(remoteAdmin, h)
	registerErrorPassthroughRoutes(remoteAdmin, h)
	registerTLSFingerprintProfileRoutes(remoteAdmin, h)
	settings := remoteAdmin.Group("/settings")
	{
		settings.GET("/web-search-emulation", h.Admin.Setting.GetWebSearchEmulationConfig)
		settings.PUT("/web-search-emulation", h.Admin.Setting.UpdateWebSearchEmulationConfig)
		settings.POST("/web-search-emulation/test", h.Admin.Setting.TestWebSearchEmulation)
		settings.POST("/web-search-emulation/reset-usage", h.Admin.Setting.ResetWebSearchUsage)
	}
}
