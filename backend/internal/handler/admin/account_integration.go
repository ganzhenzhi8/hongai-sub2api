package admin

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/integrationauth"
	"github.com/Wei-Shaw/sub2api/internal/pkg/integrationclient"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// integrationEnabled is deliberately environment-gated. No route can mutate
// accounts until the operator explicitly enables it on that site.
func integrationEnabled() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv("SUB2_INTEGRATION_ENABLED")), "true")
}

func integrationCredentials() (string, string) {
	return strings.TrimSpace(os.Getenv("SUB2_INTEGRATION_ID")), strings.TrimSpace(os.Getenv("SUB2_INTEGRATION_SECRET"))
}

var (
	integrationNonceMu sync.Mutex
	integrationNonces  = make(map[string]time.Time)
)

func (h *AccountHandler) verifyIntegration(c *gin.Context) bool {
	if !integrationEnabled() {
		c.JSON(http.StatusNotFound, gin.H{"error": "integration disabled"})
		return false
	}
	id, secret := integrationCredentials()
	if id == "" || secret == "" || strings.TrimSpace(c.GetHeader(integrationauth.HeaderID)) != id {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "integration unauthorized"})
		return false
	}
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, 16<<20))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return false
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(body))
	if err := integrationauth.Verify(c.Request, secret, body, time.Now(), integrationauth.DefaultClockSkew); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "integration signature rejected"})
		return false
	}
	nonce := strings.TrimSpace(c.GetHeader(integrationauth.HeaderNonce))
	nonceKey := id + ":" + nonce
	now := time.Now()
	integrationNonceMu.Lock()
	defer integrationNonceMu.Unlock()
	for key, usedAt := range integrationNonces {
		if now.Sub(usedAt) > 10*time.Minute {
			delete(integrationNonces, key)
		}
	}
	if usedAt, loaded := integrationNonces[nonceKey]; loaded {
		if now.Sub(usedAt) <= 10*time.Minute {
			c.JSON(http.StatusConflict, gin.H{"error": "integration nonce already used"})
			return false
		}
		delete(integrationNonces, nonceKey)
	}
	if len(integrationNonces) >= 10000 {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "integration replay cache is full"})
		return false
	}
	integrationNonces[nonceKey] = now
	return true
}

// IntegrationAuth protects the full-fidelity account management bridge. The
// receiver remains disabled unless the explicit integration environment gate
// is enabled, and every request is signed and replay-protected.
func (h *AccountHandler) IntegrationAuth(c *gin.Context) {
	if !h.verifyIntegration(c) {
		c.Abort()
		return
	}
	c.Next()
}

func (h *AccountHandler) IntegrationHealth(c *gin.Context) {
	if !h.verifyIntegration(c) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "sub2api-integration"})
}

func (h *AccountHandler) IntegrationGroups(c *gin.Context) {
	if !h.verifyIntegration(c) {
		return
	}
	groups, err := h.adminService.GetAllGroupsIncludingInactive(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list groups"})
		return
	}
	type groupView struct {
		ID       int64  `json:"id"`
		Name     string `json:"name"`
		Platform string `json:"platform"`
		Status   string `json:"status"`
	}
	result := make([]groupView, 0, len(groups))
	for _, group := range groups {
		result = append(result, groupView{ID: group.ID, Name: group.Name, Platform: group.Platform, Status: group.Status})
	}
	c.JSON(http.StatusOK, gin.H{"groups": result})
}

type integrationAccountRequest struct {
	RequestID       string         `json:"request_id"`
	ItemID          string         `json:"item_id"`
	Name            string         `json:"name" binding:"required"`
	Notes           *string        `json:"notes"`
	Platform        string         `json:"platform" binding:"required"`
	Type            string         `json:"type" binding:"required"`
	Credentials     map[string]any `json:"credentials" binding:"required"`
	Extra           map[string]any `json:"extra"`
	ProxyID         *int64         `json:"proxy_id"`
	Concurrency     int            `json:"concurrency"`
	Priority        int            `json:"priority"`
	RateMultiplier  *float64       `json:"rate_multiplier"`
	LoadFactor      *int           `json:"load_factor"`
	GroupIDs        []int64        `json:"group_ids"`
	ExpiresAt       *int64         `json:"expires_at"`
	AutoPauseExpiry *bool          `json:"auto_pause_on_expired"`
	ProbeEnabled    *bool          `json:"upstream_billing_probe_enabled"`
}

func (r integrationAccountRequest) createInput() *service.CreateAccountInput {
	extra := cloneIntegrationExtra(r.Extra)
	if r.RequestID != "" {
		extra["integration_request_id"] = r.RequestID
	}
	if r.ItemID != "" {
		extra["integration_item_id"] = r.ItemID
	}
	return &service.CreateAccountInput{
		Name: r.Name, Notes: r.Notes, Platform: r.Platform, Type: r.Type,
		Credentials: r.Credentials, Extra: extra, ProxyID: r.ProxyID,
		Concurrency: r.Concurrency, Priority: r.Priority, RateMultiplier: r.RateMultiplier,
		LoadFactor: r.LoadFactor, GroupIDs: r.GroupIDs, ExpiresAt: r.ExpiresAt,
		AutoPauseOnExpired: r.AutoPauseExpiry, ProbeEnabled: r.ProbeEnabled, SkipMixedChannelCheck: true,
	}
}

func cloneIntegrationExtra(value map[string]any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	result := make(map[string]any, len(value)+2)
	for key, item := range value {
		// Integration metadata is server-owned and cannot be spoofed by callers.
		if key == "integration_request_id" || key == "integration_item_id" {
			continue
		}
		result[key] = item
	}
	return result
}

func (h *AccountHandler) IntegrationAccounts(c *gin.Context) {
	if !h.verifyIntegration(c) {
		return
	}
	page := parseIntegrationInt(c.Query("page"), 1)
	pageSize := parseIntegrationInt(c.Query("page_size"), 100)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 500 {
		pageSize = 100
	}
	groupID := int64(0)
	if raw := strings.TrimSpace(c.Query("group_id")); raw != "" {
		groupID, _ = strconv.ParseInt(raw, 10, 64)
	}
	accounts, total, err := h.adminService.ListAccounts(c.Request.Context(), page, pageSize, c.Query("platform"), c.Query("type"), c.Query("status"), c.Query("search"), groupID, "", "updated_at", "desc")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list accounts"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"accounts": sanitizeIntegrationAccounts(accounts), "total": total, "page": page, "page_size": pageSize})
}

func (h *AccountHandler) IntegrationCreateAccount(c *gin.Context) {
	if !h.verifyIntegration(c) {
		return
	}
	var req integrationAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account payload"})
		return
	}
	if req.RequestID != "" && req.ItemID != "" {
		if existing := h.findIntegrationAccount(c, req.RequestID, req.ItemID); existing != nil {
			c.JSON(http.StatusOK, gin.H{"account": sanitizeIntegrationAccount(*existing), "replayed": true})
			return
		}
	}
	account, err := h.adminService.CreateAccount(c.Request.Context(), req.createInput())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to create account"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"account": sanitizeIntegrationAccount(*account), "replayed": false})
}

type integrationAccountUpdateRequest struct {
	Name           string         `json:"name"`
	Notes          *string        `json:"notes"`
	Type           string         `json:"type"`
	Credentials    map[string]any `json:"credentials"`
	Extra          map[string]any `json:"extra"`
	ProxyID        *int64         `json:"proxy_id"`
	Concurrency    *int           `json:"concurrency"`
	Priority       *int           `json:"priority"`
	RateMultiplier *float64       `json:"rate_multiplier"`
	LoadFactor     *int           `json:"load_factor"`
	Status         string         `json:"status"`
	GroupIDs       *[]int64       `json:"group_ids"`
	ExpiresAt      *int64         `json:"expires_at"`
	AutoPause      *bool          `json:"auto_pause_on_expired"`
	Schedulable    *bool          `json:"schedulable"`
}

func (h *AccountHandler) IntegrationUpdateAccount(c *gin.Context) {
	if !h.verifyIntegration(c) {
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account id"})
		return
	}
	var req integrationAccountUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account payload"})
		return
	}
	account, err := h.adminService.UpdateAccount(c.Request.Context(), id, &service.UpdateAccountInput{
		Name: req.Name, Notes: req.Notes, Type: req.Type, Credentials: req.Credentials,
		Extra: req.Extra, ProxyID: req.ProxyID, Concurrency: req.Concurrency, Priority: req.Priority,
		RateMultiplier: req.RateMultiplier, LoadFactor: req.LoadFactor, Status: req.Status,
		GroupIDs: req.GroupIDs, ExpiresAt: req.ExpiresAt, AutoPauseOnExpired: req.AutoPause,
		SkipMixedChannelCheck: true,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to update account"})
		return
	}
	if req.Schedulable != nil {
		account, err = h.adminService.SetAccountSchedulable(c.Request.Context(), id, *req.Schedulable)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "failed to update schedulable state"})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"account": sanitizeIntegrationAccount(*account)})
}

func (h *AccountHandler) IntegrationDeleteAccount(c *gin.Context) {
	if !h.verifyIntegration(c) {
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account id"})
		return
	}
	if err := h.adminService.DeleteAccount(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to delete account"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": true})
}

func remoteIntegrationClient() (*integrationclient.Client, error) {
	if !strings.EqualFold(strings.TrimSpace(os.Getenv("SUB2_EAMON_INTEGRATION_ENABLED")), "true") {
		return nil, os.ErrNotExist
	}
	return integrationclient.New(
		os.Getenv("SUB2_EAMON_INTEGRATION_URL"),
		os.Getenv("SUB2_EAMON_INTEGRATION_ID"),
		os.Getenv("SUB2_EAMON_INTEGRATION_SECRET"),
	)
}

func (h *AccountHandler) IntegrationRemoteGroups(c *gin.Context) {
	client, err := remoteIntegrationClient()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "eamon integration disabled"})
		return
	}
	var result map[string]any
	if err := client.DoJSON(c.Request.Context(), http.MethodGet, "/api/v1/integration/groups", nil, &result); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "eamon groups unavailable"})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *AccountHandler) IntegrationRemoteAccounts(c *gin.Context) {
	client, err := remoteIntegrationClient()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "eamon integration disabled"})
		return
	}
	query := c.Request.URL.Query()
	path := "/api/v1/integration/accounts"
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}
	var result map[string]any
	if err := client.DoJSON(c.Request.Context(), http.MethodGet, path, nil, &result); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "eamon accounts unavailable"})
		return
	}
	c.JSON(http.StatusOK, result)
}

// IntegrationPushAccount creates the selected copy/copies. It deliberately
// does not deduplicate by credential: users may create multiple copies of an
// account for separate groups. request_id/item_id only make transport retries
// safe on the remote side.
func (h *AccountHandler) IntegrationPushAccount(c *gin.Context) {
	var payload struct {
		Targets        []string                  `json:"targets" binding:"required,min=1"`
		Account        integrationAccountRequest `json:"account" binding:"required"`
		LocalGroupIDs  []int64                   `json:"local_group_ids"`
		RemoteGroupIDs []int64                   `json:"remote_group_ids"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid push payload"})
		return
	}
	results := make(map[string]any, len(payload.Targets))
	for _, target := range payload.Targets {
		switch strings.ToLower(strings.TrimSpace(target)) {
		case "hongai", "local":
			localAccount := payload.Account
			localAccount.GroupIDs = payload.LocalGroupIDs
			account, err := h.adminService.CreateAccount(c.Request.Context(), localAccount.createInput())
			if err != nil {
				results["hongai"] = gin.H{"ok": false, "error": "local account creation failed"}
				continue
			}
			results["hongai"] = gin.H{"ok": true, "account": sanitizeIntegrationAccount(*account)}
		case "eamon", "eamon88":
			client, err := remoteIntegrationClient()
			if err != nil {
				results["eamon88"] = gin.H{"ok": false, "error": "eamon integration disabled"}
				continue
			}
			remoteAccount := payload.Account
			remoteAccount.GroupIDs = payload.RemoteGroupIDs
			var response map[string]any
			if err := client.DoJSON(c.Request.Context(), http.MethodPost, "/api/v1/integration/accounts", remoteAccount, &response); err != nil {
				results["eamon88"] = gin.H{"ok": false, "error": "eamon account creation failed"}
				continue
			}
			results["eamon88"] = gin.H{"ok": true, "result": response}
		default:
			results[target] = gin.H{"ok": false, "error": "unknown target"}
		}
	}
	c.JSON(http.StatusOK, gin.H{"results": results})
}

func (h *AccountHandler) IntegrationRemoteUpdate(c *gin.Context) {
	client, err := remoteIntegrationClient()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "eamon integration disabled"})
		return
	}
	var payload integrationAccountUpdateRequest
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account payload"})
		return
	}
	var result map[string]any
	path := "/api/v1/integration/accounts/" + c.Param("id")
	if err := client.DoJSON(c.Request.Context(), http.MethodPut, path, payload, &result); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "eamon account update failed"})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *AccountHandler) IntegrationRemoteDelete(c *gin.Context) {
	client, err := remoteIntegrationClient()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "eamon integration disabled"})
		return
	}
	var result map[string]any
	path := "/api/v1/integration/accounts/" + c.Param("id")
	if err := client.DoJSON(c.Request.Context(), http.MethodDelete, path, nil, &result); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "eamon account deletion failed"})
		return
	}
	c.JSON(http.StatusOK, result)
}

// IntegrationRemoteAdminProxy forwards only the account-management routes
// that are registered by RegisterIntegrationRoutes. It never exposes a
// caller-controlled host and therefore cannot be used as a general proxy.
func (h *AccountHandler) IntegrationRemoteAdminProxy(c *gin.Context) {
	client, err := remoteIntegrationClient()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "eamon integration disabled"})
		return
	}

	bridgePath := "/" + strings.TrimLeft(c.Param("path"), "/")
	if !integrationAdminPathAllowed(bridgePath) {
		c.JSON(http.StatusNotFound, gin.H{"error": "integration admin route not available"})
		return
	}
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, 32<<20))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}
	remotePath := "/api/v1/integration/admin" + bridgePath
	if rawQuery := c.Request.URL.Query().Encode(); rawQuery != "" {
		remotePath += "?" + rawQuery
	}
	resp, err := client.Do(c.Request.Context(), c.Request.Method, remotePath, body)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "eamon account management unavailable"})
		return
	}
	defer resp.Body.Close()
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/json; charset=utf-8"
	}
	if etag := resp.Header.Get("ETag"); etag != "" {
		c.Header("ETag", etag)
	}
	if cacheControl := resp.Header.Get("Cache-Control"); cacheControl != "" {
		c.Header("Cache-Control", cacheControl)
	}
	c.Header("Content-Type", contentType)
	c.Status(resp.StatusCode)
	remaining := int64(32 << 20)
	buffer := make([]byte, 32<<10)
	for remaining > 0 {
		readCount, readErr := resp.Body.Read(buffer)
		if readCount > 0 {
			if _, writeErr := c.Writer.Write(buffer[:readCount]); writeErr != nil {
				return
			}
			c.Writer.Flush()
			remaining -= int64(readCount)
		}
		if readErr != nil {
			return
		}
	}
}

func integrationAdminPathAllowed(path string) bool {
	for _, prefix := range []string{
		"/accounts",
		"/groups/all",
		"/proxies/all",
		"/scheduled-test-plans",
		"/grok",
		"/cn-providers",
		"/settings/web-search-emulation",
		"/error-passthrough-rules",
		"/tls-fingerprint-profiles",
	} {
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			return true
		}
	}
	return false
}

func parseIntegrationInt(raw string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return fallback
	}
	return value
}

func (h *AccountHandler) findIntegrationAccount(c *gin.Context, requestID, itemID string) *service.Account {
	accounts, _, err := h.adminService.ListAccounts(c.Request.Context(), 1, 10000, "", "", "", "", 0, "", "id", "desc")
	if err != nil {
		return nil
	}
	for i := range accounts {
		if stringValue(accounts[i].Extra, "integration_request_id") == requestID && stringValue(accounts[i].Extra, "integration_item_id") == itemID {
			return &accounts[i]
		}
	}
	return nil
}

func stringValue(values map[string]any, key string) string {
	if values == nil {
		return ""
	}
	value, _ := values[key].(string)
	return value
}

func sanitizeIntegrationAccounts(accounts []service.Account) []gin.H {
	result := make([]gin.H, 0, len(accounts))
	for _, account := range accounts {
		result = append(result, sanitizeIntegrationAccount(account))
	}
	return result
}

func sanitizeIntegrationAccount(account service.Account) gin.H {
	extra := cloneIntegrationExtra(account.Extra)
	delete(extra, "integration_request_id")
	delete(extra, "integration_item_id")
	return gin.H{
		"id": account.ID, "name": account.Name, "platform": account.Platform, "type": account.Type,
		"extra": extra, "proxy_id": account.ProxyID, "concurrency": account.Concurrency,
		"priority": account.Priority, "rate_multiplier": account.BillingRateMultiplier(),
		"load_factor": account.LoadFactor, "status": account.Status, "schedulable": account.Schedulable,
		"group_ids": account.GroupIDs, "expires_at": account.ExpiresAt,
		"auto_pause_on_expired": account.AutoPauseOnExpired,
	}
}
