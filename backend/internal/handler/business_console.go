package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"avibe-platform/backend/internal/repository"
	"avibe-platform/backend/internal/service"
)

// BusinessConsoleHandler serves the R10 business console: the caller's
// company profile plus owner-keyed store/device/schedule management.
// Unlike the legacy /partner/stores path (partner-profile keyed), these
// endpoints resolve ownership by owner_user_id so BUSINESS-only accounts
// work without a creator profile.
type BusinessConsoleHandler struct {
	Repos    *repository.Repos
	Services *service.Services
}

// BusinessProfile returns the caller's company profile.
func (h *BusinessConsoleHandler) BusinessProfile(c *gin.Context) {
	p, err := h.Services.Business.BusinessProfile(c.Request.Context(), CurrentUserID(c))
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, p)
}

// RegisterStore files a store business for admin verification.
func (h *BusinessConsoleHandler) RegisterStore(c *gin.Context) {
	var req registerStoreRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", err.Error())
		return
	}
	licenses := req.LicenseCount
	if licenses == 0 {
		licenses = 1
	}
	b, err := h.Services.Store.RegisterBusinessForOwner(c.Request.Context(), CurrentUserID(c), req.BusinessNumber, req.StoreName, licenses)
	if err != nil {
		AbortError(c, err)
		return
	}
	created(c, b)
}

// ListStores returns the caller's store businesses.
func (h *BusinessConsoleHandler) ListStores(c *gin.Context) {
	list, err := h.Services.Store.ListBusinessesByOwner(c.Request.Context(), CurrentUserID(c))
	if err != nil {
		AbortError(c, err)
		return
	}
	okList(c, list)
}

// CreateStoreDevice registers a player device and returns its pairing code.
func (h *BusinessConsoleHandler) CreateStoreDevice(c *gin.Context) {
	var req createStoreDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", err.Error())
		return
	}
	dev, err := h.Services.Store.CreateDeviceForOwner(c.Request.Context(), CurrentUserID(c), c.Param("id"), req.DeviceName)
	if err != nil {
		AbortError(c, err)
		return
	}
	created(c, dev)
}

// ListStoreDevices returns a store's player devices.
func (h *BusinessConsoleHandler) ListStoreDevices(c *gin.Context) {
	list, err := h.Services.Store.ListDevicesForOwner(c.Request.Context(), CurrentUserID(c), c.Param("id"))
	if err != nil {
		AbortError(c, err)
		return
	}
	okList(c, list)
}

// UpdateStoreDevice renames or re-volumes a player (remote volume).
func (h *BusinessConsoleHandler) UpdateStoreDevice(c *gin.Context) {
	var req updateStoreDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", err.Error())
		return
	}
	dev, err := h.Services.Store.UpdateDeviceForOwner(c.Request.Context(), CurrentUserID(c), c.Param("deviceId"), req.DeviceName, req.Volume)
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, dev)
}

// RevokeStoreDevice deactivates a player.
func (h *BusinessConsoleHandler) RevokeStoreDevice(c *gin.Context) {
	if err := h.Services.Store.RevokeDeviceForOwner(c.Request.Context(), CurrentUserID(c), c.Param("deviceId")); err != nil {
		AbortError(c, err)
		return
	}
	ok(c, gin.H{"revoked": true})
}

// ListStoreSchedules returns a store's weekly schedule.
func (h *BusinessConsoleHandler) ListStoreSchedules(c *gin.Context) {
	list, err := h.Services.Store.ListSchedulesForOwner(c.Request.Context(), CurrentUserID(c), c.Param("id"))
	if err != nil {
		AbortError(c, err)
		return
	}
	okList(c, list)
}

// UpsertStoreSchedule sets one weekday row for a store.
func (h *BusinessConsoleHandler) UpsertStoreSchedule(c *gin.Context) {
	var req upsertStoreScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", err.Error())
		return
	}
	s, err := h.Services.Store.UpsertScheduleForOwner(c.Request.Context(), CurrentUserID(c), c.Param("id"), req.Weekday, req.OpenTime, req.CloseTime, req.PlaylistID, req.IsClosed)
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, s)
}

// StoreDeviceLogs returns a device's recent play logs.
func (h *BusinessConsoleHandler) StoreDeviceLogs(c *gin.Context) {
	logs, err := h.Services.Store.DeviceLogsForOwner(c.Request.Context(), CurrentUserID(c), c.Param("deviceId"), queryInt(c, "limit", 50))
	if err != nil {
		AbortError(c, err)
		return
	}
	okList(c, logs)
}
