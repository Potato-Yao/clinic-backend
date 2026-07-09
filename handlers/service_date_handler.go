package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"clinic-backend/services"

	"github.com/gin-gonic/gin"
)

type ServiceDateHandler struct {
	svc *services.ServiceDateService
}

func NewServiceDateHandler(svc *services.ServiceDateService) *ServiceDateHandler {
	return &ServiceDateHandler{svc: svc}
}

type createServiceDateRequest struct {
	Capacity  uint      `json:"capacity" binding:"required"`
	RoomID    uint      `json:"room_id" binding:"required"`
	Date      time.Time `json:"date" binding:"required"`
	StartTime time.Time `json:"startTime" binding:"required"`
	EndTime   time.Time `json:"endTime" binding:"required"`
	Title     string    `json:"title" binding:"required,max=20"`
}

type updateServiceDateRequest struct {
	Capacity  *uint      `json:"capacity"`
	RoomID    *uint      `json:"room_id"`
	Date      *time.Time `json:"date"`
	StartTime *time.Time `json:"startTime"`
	EndTime   *time.Time `json:"endTime"`
	Title     *string    `json:"title" binding:"omitempty,max=20"`
}

func (h *ServiceDateHandler) Create(c *gin.Context) {
	var req createServiceDateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validateServiceDateWindow(req.Date, req.StartTime, req.EndTime); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	d, err := h.svc.Create(services.CreateServiceDateInput{
		Capacity:  req.Capacity,
		RoomID:    req.RoomID,
		Date:      req.Date,
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
		Title:     req.Title,
	})
	if err != nil {
		writeServiceDateError(c, err)
		return
	}
	c.JSON(http.StatusCreated, d)
}

func (h *ServiceDateHandler) Get(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	d, err := h.svc.GetByID(id)
	if err != nil {
		writeServiceDateError(c, err)
		return
	}
	c.JSON(http.StatusOK, d)
}

func (h *ServiceDateHandler) List(c *gin.Context) {
	f := services.ListServiceDateFilter{
		ActiveOnly:  c.Query("active") == "true",
		HasCapacity: c.Query("available") == "true",
	}
	if v, err := strconv.ParseUint(c.Query("room_id"), 10, 64); err == nil && v > 0 {
		f.RoomID = new(uint(v))
	}
	if v, err := time.Parse(time.RFC3339, c.Query("from")); err == nil {
		f.FromDate = v
	}
	if v, err := strconv.Atoi(c.DefaultQuery("page", "1")); err == nil {
		f.Page = v
	}
	if v, err := strconv.Atoi(c.DefaultQuery("pageSize", "20")); err == nil {
		f.PageSize = v
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

func (h *ServiceDateHandler) Update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req updateServiceDateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate the resulting window against provided fields.
	effDate := req.Date
	effStart := req.StartTime
	effEnd := req.EndTime
	if effDate != nil && effStart != nil && effEnd != nil {
		if err := validateServiceDateWindow(*effDate, *effStart, *effEnd); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	d, err := h.svc.Update(id, services.UpdateServiceDateInput{
		Capacity:  req.Capacity,
		RoomID:    req.RoomID,
		Date:      req.Date,
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
		Title:     req.Title,
	})
	if err != nil {
		writeServiceDateError(c, err)
		return
	}
	c.JSON(http.StatusOK, d)
}

func (h *ServiceDateHandler) Delete(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.svc.Delete(id); err != nil {
		writeServiceDateError(c, err)
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

// writeServiceDateError maps service errors to HTTP statuses.
func writeServiceDateError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrServiceDateNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, services.ErrServiceDateRoomNotFound):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, services.ErrServiceDateInUse):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}

// validateServiceDateWindow rejects past dates and inverted or same-day-spanning windows.
func validateServiceDateWindow(date, start, end time.Time) error {
	today := time.Now().UTC().Truncate(24 * time.Hour)
	if date.Truncate(24 * time.Hour).Before(today) {
		return errors.New("date must not be in the past")
	}
	if !start.Before(end) {
		return errors.New("startTime must be before endTime")
	}
	day := start.Truncate(24 * time.Hour)
	if day.Add(24*time.Hour).Before(end) || !end.Truncate(24*time.Hour).Equal(day) {
		return errors.New("endTime must fall on the same day as startTime")
	}
	return nil
}
