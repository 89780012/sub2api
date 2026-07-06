package admin

import (
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type BalanceSiteHandler struct {
	service *service.BalanceSnapshotService
}

func NewBalanceSiteHandler(service *service.BalanceSnapshotService) *BalanceSiteHandler {
	return &BalanceSiteHandler{service: service}
}

type balanceSiteRequest struct {
	Platform               string  `json:"platform" binding:"required,oneof=newapi sub2api"`
	Name                   string  `json:"name" binding:"required"`
	BaseURL                string  `json:"base_url" binding:"required"`
	AuthMode               string  `json:"auth_mode" binding:"omitempty,oneof=password access_token cookie"`
	Username               string  `json:"username"`
	Email                  string  `json:"email"`
	Password               *string `json:"password"`
	AccessToken            *string `json:"access_token"`
	Enabled                *bool   `json:"enabled"`
	RefreshIntervalMinutes *int    `json:"refresh_interval_minutes"`
}

type balanceBindingRequest struct {
	SiteID        int64   `json:"site_id" binding:"required"`
	ExternalKeyID *string `json:"external_key_id"`
	KeyLast4      string  `json:"key_last4"`
}

func (h *BalanceSiteHandler) List(c *gin.Context) {
	sites, err := h.service.ListSites(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, sites)
}

func (h *BalanceSiteHandler) Create(c *gin.Context) {
	var req balanceSiteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	site, err := h.service.CreateSite(c.Request.Context(), balanceSiteInput(req))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, site)
}

func (h *BalanceSiteHandler) Update(c *gin.Context) {
	id, ok := parseBalanceIDParam(c, "id", "Invalid balance site ID")
	if !ok {
		return
	}
	var req balanceSiteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	site, err := h.service.UpdateSite(c.Request.Context(), id, balanceSiteInput(req))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, site)
}

func (h *BalanceSiteHandler) Delete(c *gin.Context) {
	id, ok := parseBalanceIDParam(c, "id", "Invalid balance site ID")
	if !ok {
		return
	}
	if err := h.service.DeleteSite(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "Balance site deleted successfully"})
}

func (h *BalanceSiteHandler) Refresh(c *gin.Context) {
	id, ok := parseBalanceIDParam(c, "id", "Invalid balance site ID")
	if !ok {
		return
	}
	result, err := h.service.RefreshSite(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *BalanceSiteHandler) RefreshAll(c *gin.Context) {
	results, err := h.service.RefreshAll(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"results": results})
}

func (h *BalanceSiteHandler) Snapshots(c *gin.Context) {
	var siteID int64
	if raw := c.Query("site_id"); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || parsed < 0 {
			response.BadRequest(c, "Invalid site_id")
			return
		}
		siteID = parsed
	}
	snapshots, err := h.service.ListSnapshots(c.Request.Context(), siteID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"items": snapshots})
}

func (h *BalanceSiteHandler) UpsertBinding(c *gin.Context) {
	accountID, ok := parseBalanceIDParam(c, "id", "Invalid account ID")
	if !ok {
		return
	}
	var req balanceBindingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	err := h.service.UpsertBinding(c.Request.Context(), accountID, service.BalanceBindingInput{
		SiteID:        req.SiteID,
		ExternalKeyID: req.ExternalKeyID,
		KeyLast4:      req.KeyLast4,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "Balance binding updated successfully"})
}

func (h *BalanceSiteHandler) DeleteBinding(c *gin.Context) {
	accountID, ok := parseBalanceIDParam(c, "id", "Invalid account ID")
	if !ok {
		return
	}
	if err := h.service.DeleteBinding(c.Request.Context(), accountID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "Balance binding deleted successfully"})
}

func balanceSiteInput(req balanceSiteRequest) service.BalanceSiteInput {
	return service.BalanceSiteInput{
		Platform:               req.Platform,
		Name:                   req.Name,
		BaseURL:                req.BaseURL,
		AuthMode:               req.AuthMode,
		Username:               req.Username,
		Email:                  req.Email,
		Password:               req.Password,
		AccessToken:            req.AccessToken,
		Enabled:                req.Enabled,
		RefreshIntervalMinutes: req.RefreshIntervalMinutes,
	}
}

func parseBalanceIDParam(c *gin.Context, name, message string) (int64, bool) {
	id, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, message)
		return 0, false
	}
	return id, true
}
