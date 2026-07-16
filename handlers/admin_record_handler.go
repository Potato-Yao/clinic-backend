package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"clinic-backend/models"
	"clinic-backend/services"

	"github.com/gin-gonic/gin"
)

type AdminRecordHandler struct {
	svc *services.AdminRecordService
}

func NewAdminRecordHandler(svc *services.AdminRecordService) *AdminRecordHandler {
	return &AdminRecordHandler{svc: svc}
}

type rejectRecordRequest struct {
	Reason string `json:"reason" binding:"required"`
}

func (h *AdminRecordHandler) List(c *gin.Context) {
	f := services.ListAdminRecordFilter{
		Status:   c.Query("status"),
		Page:     parseIntDefault(c, "page", 1),
		PageSize: parseIntDefault(c, "pageSize", 20),
	}
	if v := c.Query("room_id"); v != "" {
		id, err := parseUintQuery(v)
		if err == nil {
			f.RoomID = &id
		}
	}
	if v := c.Query("from_date"); v != "" {
		t, err := time.Parse("2006-01-02", v)
		if err == nil {
			f.FromDate = &t
		}
	}
	if v := c.Query("to_date"); v != "" {
		t, err := time.Parse("2006-01-02", v)
		if err == nil {
			f.ToDate = &t
		}
	}

	items, total, err := h.svc.List(f)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"items":    items,
		"total":    total,
		"page":     f.Page,
		"pageSize": f.PageSize,
	})
}

func (h *AdminRecordHandler) Get(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	v, err := h.svc.GetByID(id)
	if err != nil {
		writeRecordError(c, err)
		return
	}
	c.JSON(http.StatusOK, v)
}

type updateRecordRequest struct {
	Status     *string `json:"status"`
	WorkerDesc *string `json:"worker_desc"`
}

func (h *AdminRecordHandler) Update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	var req updateRecordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	in := services.UpdateAdminRecordInput{
		WorkerDesc: req.WorkerDesc,
	}
	if req.Status != nil {
		s := models.RecordStatus(*req.Status)
		in.Status = &s
	}

	v, err := h.svc.Update(id, in)
	if err != nil {
		writeRecordError(c, err)
		return
	}
	c.JSON(http.StatusOK, v)
}

func (h *AdminRecordHandler) Arrive(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	v, err := h.svc.MarkArrived(id)
	if err != nil {
		writeRecordError(c, err)
		return
	}
	c.JSON(http.StatusOK, v)
}

func (h *AdminRecordHandler) InProgress(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	staff, ok := contextStaff(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "missing staff context"})
		return
	}
	v, err := h.svc.MarkInProgress(id, uint(staff.ID))
	if err != nil {
		writeRecordError(c, err)
		return
	}
	c.JSON(http.StatusOK, v)
}

func (h *AdminRecordHandler) Complete(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	v, err := h.svc.MarkCompleted(id)
	if err != nil {
		writeRecordError(c, err)
		return
	}
	c.JSON(http.StatusOK, v)
}

func (h *AdminRecordHandler) Reject(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	var req rejectRecordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	v, err := h.svc.MarkRejected(id, req.Reason)
	if err != nil {
		writeRecordError(c, err)
		return
	}
	c.JSON(http.StatusOK, v)
}

func writeRecordError(c *gin.Context, err error) {
	if errors.Is(err, services.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
}

func parseUintQuery(s string) (uint, error) {
	id, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(id), nil
}
