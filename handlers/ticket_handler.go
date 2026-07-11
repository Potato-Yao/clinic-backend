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

type TicketHandler struct {
	svc *services.TicketService
}

func NewTicketHandler(svc *services.TicketService) *TicketHandler {
	return &TicketHandler{svc: svc}
}

type createTicketRequest struct {
	Realname        string `json:"realname" binding:"required"`
	PhoneNum        string `json:"phone_num" binding:"required"`
	Campus          string `json:"campus" binding:"required"`
	AppointmentTime string `json:"appointment_time" binding:"required"`
	Description     string `json:"description" binding:"required"`
	Model           string `json:"model"`
	Password        string `json:"password"`
}

type updateTicketRequest struct {
	Realname        *string `json:"realname"`
	PhoneNum        *string `json:"phone_num"`
	Campus          *string `json:"campus"`
	AppointmentTime *string `json:"appointment_time"`
	Description     *string `json:"description"`
	Model           *string `json:"model"`
	Password        *string `json:"password"`
}

func (h *TicketHandler) Create(c *gin.Context) {
	username := c.GetString("user")
	if username == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing user"})
		return
	}

	var req createTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	appointmentTime, err := time.Parse("2006-01-02", req.AppointmentTime)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "invalid appointment_time"})
		return
	}

	rec, err := h.svc.Create(services.CreateTicketInput{
		User:            username,
		Realname:        req.Realname,
		PhoneNum:        req.PhoneNum,
		Campus:          req.Campus,
		AppointmentTime: appointmentTime,
		Description:     req.Description,
		Model:           req.Model,
		Password:        req.Password,
	})
	if err != nil {
		writeTicketError(c, err)
		return
	}

	view, err := h.svc.View(rec)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, view)
}

func (h *TicketHandler) List(c *gin.Context) {
	h.listTickets(c, h.svc.List)
}

func (h *TicketHandler) Finished(c *gin.Context) {
	h.listTickets(c, h.svc.Finished)
}

func (h *TicketHandler) listTickets(c *gin.Context, fetch func(string, services.ListTicketFilter) ([]models.ClinicRecord, int64, error)) {
	username := c.GetString("user")
	if username == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing user"})
		return
	}

	f := services.ListTicketFilter{
		Page:     parseIntDefault(c, "page", 1),
		PageSize: parseIntDefault(c, "pageSize", 20),
	}
	items, total, err := fetch(username, f)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	views := make([]services.TicketView, 0, len(items))
	for _, rec := range items {
		v, err := h.svc.View(rec)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		views = append(views, v)
	}
	c.JSON(http.StatusOK, gin.H{
		"items":    views,
		"total":    total,
		"page":     f.Page,
		"pageSize": f.PageSize,
	})
}

func (h *TicketHandler) Working(c *gin.Context) {
	username := c.GetString("user")
	if username == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing user"})
		return
	}

	rec, err := h.svc.Working(username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if rec == nil {
		c.JSON(http.StatusOK, gin.H{"count": 0})
		return
	}
	view, err := h.svc.View(*rec)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"count": 1,
		"data":  view,
	})
}

func (h *TicketHandler) Get(c *gin.Context) {
	username := c.GetString("user")
	if username == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing user"})
		return
	}
	id, ok := parseID(c)
	if !ok {
		return
	}

	rec, err := h.svc.GetForUser(id, username)
	if err != nil {
		writeTicketError(c, err)
		return
	}
	view, err := h.svc.View(rec)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, view)
}

func (h *TicketHandler) Update(c *gin.Context) {
	username := c.GetString("user")
	if username == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing user"})
		return
	}
	id, ok := parseID(c)
	if !ok {
		return
	}

	var req updateTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	in := services.UpdateTicketInput{
		Realname:    req.Realname,
		PhoneNum:    req.PhoneNum,
		Campus:      req.Campus,
		Description: req.Description,
		Model:       req.Model,
		Password:    req.Password,
	}
	if req.AppointmentTime != nil {
		t, err := time.Parse("2006-01-02", *req.AppointmentTime)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"msg": "invalid appointment_time"})
			return
		}
		in.AppointmentTime = &t
	}

	rec, err := h.svc.UpdateForUser(id, username, in)
	if err != nil {
		writeTicketError(c, err)
		return
	}
	view, err := h.svc.View(rec)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, view)
}

func (h *TicketHandler) Delete(c *gin.Context) {
	username := c.GetString("user")
	if username == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing user"})
		return
	}
	id, ok := parseID(c)
	if !ok {
		return
	}

	if err := h.svc.DeleteForUser(id, username); err != nil {
		writeTicketError(c, err)
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

func writeTicketError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrTicketNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, services.ErrTicketForbidden):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case errors.Is(err, services.ErrTicketDateClosed),
		errors.Is(err, services.ErrTicketNoCapacity),
		errors.Is(err, services.ErrTicketOneWorking),
		errors.Is(err, services.ErrTicketPastTime),
		errors.Is(err, services.ErrTicketRoomMissing),
		errors.Is(err, services.ErrTicketInvalidDate):
		c.JSON(http.StatusBadRequest, gin.H{"msg": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}

func parseIntDefault(c *gin.Context, key string, def int) int {
	v := c.Query(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 {
		return def
	}
	return n
}
