package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"clinic-backend/services"

	"github.com/gin-gonic/gin"
)

type RoomHandler struct {
	svc *services.RoomService
}

func NewRoomHandler(svc *services.RoomService) *RoomHandler {
	return &RoomHandler{svc: svc}
}

type createRoomRequest struct {
	Name    string `json:"name" binding:"required,max=64"`
	Address string `json:"address" binding:"required,max=256"`
	Enabled *bool  `json:"enabled"`
}

type updateRoomRequest struct {
	Name    *string `json:"name" binding:"omitempty,max=64"`
	Address *string `json:"address" binding:"omitempty,max=256"`
	Enabled *bool   `json:"enabled"`
}

func (h *RoomHandler) Create(c *gin.Context) {
	var req createRoomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	r, err := h.svc.Create(services.CreateRoomInput{
		Name:    req.Name,
		Address: req.Address,
		Enabled: req.Enabled,
	})
	if err != nil {
		writeRoomError(c, err)
		return
	}
	c.JSON(http.StatusCreated, r)
}

func (h *RoomHandler) Get(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	r, err := h.svc.GetByID(id)
	if err != nil {
		writeRoomError(c, err)
		return
	}
	c.JSON(http.StatusOK, r)
}

func (h *RoomHandler) List(c *gin.Context) {
	f := services.ListRoomFilter{
		Name:        c.Query("name"),
		EnabledOnly: c.Query("enabled") == "true",
	}
	if v, err := strconv.Atoi(c.DefaultQuery("page", "1")); err == nil {
		f.Page = v
	}
	if v, err := strconv.Atoi(c.DefaultQuery("pageSize", "20")); err == nil {
		f.PageSize = v
	}

	items, total, err := h.svc.List(f)
	if err != nil {
		writeRoomError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"items":    items,
		"total":    total,
		"page":     f.Page,
		"pageSize": f.PageSize,
	})
}

func (h *RoomHandler) Update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req updateRoomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	r, err := h.svc.Update(id, services.UpdateRoomInput{
		Name:    req.Name,
		Address: req.Address,
		Enabled: req.Enabled,
	})
	if err != nil {
		writeRoomError(c, err)
		return
	}
	c.JSON(http.StatusOK, r)
}

func (h *RoomHandler) Delete(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.svc.Delete(id); err != nil {
		writeRoomError(c, err)
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

// writeRoomError maps service errors to HTTP statuses.
func writeRoomError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrRoomNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, services.ErrRoomNameTaken):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, services.ErrRoomInUse):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}
