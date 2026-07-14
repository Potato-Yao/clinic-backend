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

type AnnouncementHandler struct {
	svc *services.AnnouncementService
}

func NewAnnouncementHandler(svc *services.AnnouncementService) *AnnouncementHandler {
	return &AnnouncementHandler{svc: svc}
}

type createAnnouncementRequest struct {
	Title      string    `json:"title" binding:"required,max=20"`
	Content    string    `json:"content" binding:"required"`
	Tag        string    `json:"tag" binding:"max=16"`
	Brief      string    `json:"brief" binding:"required,max=64"`
	ExpireDate time.Time `json:"expireDate" binding:"required"`
	Priority   uint      `json:"priority"`
}

type updateAnnouncementRequest struct {
	Title      *string    `json:"title" binding:"omitempty,max=20"`
	Content    *string    `json:"content"`
	Tag        *string    `json:"tag" binding:"omitempty,max=16"`
	Brief      *string    `json:"brief" binding:"omitempty,max=64"`
	ExpireDate *time.Time `json:"expireDate"`
	Priority   *uint      `json:"priority"`
}

func (h *AnnouncementHandler) Create(c *gin.Context) {
	var req createAnnouncementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validateExpireDate(req.ExpireDate); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	a, err := h.svc.Create(services.CreateAnnouncementInput{
		Title:      req.Title,
		Content:    req.Content,
		Tag:        models.AnnouncementTag(req.Tag),
		Brief:      req.Brief,
		ExpireDate: req.ExpireDate,
		Priority:   req.Priority,
	})
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, a)
}

func (h *AnnouncementHandler) Get(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	a, err := h.svc.GetByID(id)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, a)
}

func (h *AnnouncementHandler) List(c *gin.Context) {
	f := services.ListAnnouncementFilter{
		Tag:        models.AnnouncementTag(c.Query("tag")),
		ActiveOnly: c.Query("active") == "true",
	}
	f.Page, f.PageSize = parsePagination(c)

	items, total, err := h.svc.List(f)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	paginatedResponse(c, items, total, f.Page, f.PageSize)
}

func (h *AnnouncementHandler) Update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req updateAnnouncementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.ExpireDate != nil {
		if err := validateExpireDate(*req.ExpireDate); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	var tag *models.AnnouncementTag
	if req.Tag != nil {
		t := models.AnnouncementTag(*req.Tag)
		tag = &t
	}

	a, err := h.svc.Update(id, services.UpdateAnnouncementInput{
		Title:      req.Title,
		Content:    req.Content,
		Tag:        tag,
		Brief:      req.Brief,
		ExpireDate: req.ExpireDate,
		Priority:   req.Priority,
	})
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, a)
}

func (h *AnnouncementHandler) Delete(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.svc.Delete(id); err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

var announcementErrorMappings = []errStatus{
	{services.ErrAnnouncementNotFound, http.StatusNotFound},
	{services.ErrAnnouncementInvalidTag, http.StatusBadRequest},
	{services.ErrAnnouncementTOSAlreadyExists, http.StatusConflict},
}

func writeServiceError(c *gin.Context, err error) {
	writeMappedError(c, err, announcementErrorMappings)
}

func parseID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return 0, false
	}
	return uint(id), true
}

// validateExpireDate rejects expiry dates in the past.
func validateExpireDate(t time.Time) error {
	today := time.Now().UTC().Truncate(24 * time.Hour)
	if t.Truncate(24 * time.Hour).Before(today) {
		return errors.New("expireDate must not be in the past")
	}
	return nil
}
