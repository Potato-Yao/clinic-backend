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
	Title      string `json:"title" binding:"required,max=20"`
	Content    string `json:"content" binding:"required"`
	Tag        string `json:"tag" binding:"max=16"`
	Brief      string `json:"brief" binding:"required,max=64"`
	ExpireDate string `json:"expireDate" binding:"required"`
	Priority   uint   `json:"priority"`
}

type updateAnnouncementRequest struct {
	Title      *string `json:"title" binding:"omitempty,max=20"`
	Content    *string `json:"content"`
	Tag        *string `json:"tag" binding:"omitempty,max=16"`
	Brief      *string `json:"brief" binding:"omitempty,max=64"`
	ExpireDate *string `json:"expireDate"`
	Priority   *uint   `json:"priority"`
}

func (h *AnnouncementHandler) Create(c *gin.Context) {
	var req createAnnouncementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	expireDate, err := parseDateString(req.ExpireDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validateExpireDate(expireDate); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	a, err := h.svc.Create(services.CreateAnnouncementInput{
		Title:      req.Title,
		Content:    req.Content,
		Tag:        models.AnnouncementTag(req.Tag),
		Brief:      req.Brief,
		ExpireDate: expireDate,
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
	var expireDate *time.Time
	if req.ExpireDate != nil {
		parsed, err := parseDateString(*req.ExpireDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := validateExpireDate(parsed); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		expireDate = &parsed
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
		ExpireDate: expireDate,
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

func (h *AnnouncementHandler) TOS(c *gin.Context) {
	a, err := h.svc.GetTOS()
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, a)
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

// parseDateString parses a date value sent by the admin frontend.
// It accepts either "YYYY-MM-DD" or an RFC3339 timestamp for backwards compatibility.
func parseDateString(s string) (time.Time, error) {
	layouts := []string{"2006-01-02", time.RFC3339, time.RFC3339Nano}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, errors.New("expireDate must be in YYYY-MM-DD or RFC3339 format")
}

// validateExpireDate rejects expiry dates in the past.
func validateExpireDate(t time.Time) error {
	today := time.Now().UTC().Truncate(24 * time.Hour)
	if t.Truncate(24 * time.Hour).Before(today) {
		return errors.New("expireDate must not be in the past")
	}
	return nil
}
