package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"clinic-backend/services"

	"github.com/gin-gonic/gin"
)

type WorkScheduleHandler struct {
	svc *services.WorkScheduleService
}

func NewWorkScheduleHandler(svc *services.WorkScheduleService) *WorkScheduleHandler {
	return &WorkScheduleHandler{svc: svc}
}

type weekdayRequest struct {
	Weekday   int    `json:"weekday" binding:"required"`
	StartTime string `json:"start_time" binding:"required"`
	EndTime   string `json:"end_time" binding:"required"`
	RoomID    uint   `json:"room_id" binding:"required"`
	StaffIDs  []int  `json:"staff_ids"`
}

type createWorkScheduleRequest struct {
	Name      string           `json:"name" binding:"required,max=128"`
	StartDate string           `json:"start_date" binding:"required"`
	EndDate   string           `json:"end_date" binding:"required"`
	Enabled   *bool            `json:"enabled"`
	Weekdays  []weekdayRequest `json:"weekdays"`
}

type updateWorkScheduleRequest struct {
	Name      *string          `json:"name" binding:"omitempty,max=128"`
	StartDate *string          `json:"start_date"`
	EndDate   *string          `json:"end_date"`
	Enabled   *bool            `json:"enabled"`
	Weekdays  []weekdayRequest `json:"weekdays"`
}

type staffAssignmentRequest struct {
	WeekdayID uint `json:"weekday_id"`
	StaffID   int  `json:"staff_id" binding:"required"`
	RoomID    uint `json:"room_id"`
	Weekday   int  `json:"weekday"`
}

func (h *WorkScheduleHandler) Create(c *gin.Context) {
	var req createWorkScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	startDate, err := parseDateString(req.StartDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("start_date: %v", err)})
		return
	}
	endDate, err := parseDateString(req.EndDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("end_date: %v", err)})
		return
	}
	enabled := false
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	weekdays := make([]services.WeekdayInput, len(req.Weekdays))
	for i, wd := range req.Weekdays {
		if wd.StaffIDs == nil {
			wd.StaffIDs = []int{}
		}
		weekdays[i] = services.WeekdayInput{
			Weekday:   wd.Weekday,
			StartTime: wd.StartTime,
			EndTime:   wd.EndTime,
			RoomID:    wd.RoomID,
			StaffIDs:  wd.StaffIDs,
		}
	}

	sch, err := h.svc.Create(services.CreateWorkScheduleInput{
		Name:      req.Name,
		StartDate: startDate,
		EndDate:   endDate,
		Enabled:   enabled,
		Weekdays:  weekdays,
	})
	if err != nil {
		writeWorkScheduleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, sch)
}

func (h *WorkScheduleHandler) Get(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	sch, err := h.svc.GetByID(id)
	if err != nil {
		writeWorkScheduleError(c, err)
		return
	}
	// Staff-only requests must only see enabled schedules.
	role, roleOk := contextRole(c)
	if roleOk && role == RoleStaff && !sch.Enabled {
		c.JSON(http.StatusNotFound, gin.H{"error": services.ErrWorkScheduleNotFound.Error()})
		return
	}
	c.JSON(http.StatusOK, sch)
}

func (h *WorkScheduleHandler) List(c *gin.Context) {
	f := services.ListWorkScheduleFilter{
		Enabled: boolPtr(true),
	}
	f.Page, f.PageSize = parsePagination(c)
	items, total, err := h.svc.List(f)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	paginatedResponse(c, items, total, f.Page, f.PageSize)
}

func (h *WorkScheduleHandler) ListAll(c *gin.Context) {
	f := services.ListWorkScheduleFilter{}
	if v := c.Query("enabled"); v != "" {
		b, err := strconv.ParseBool(v)
		if err == nil {
			f.Enabled = &b
		}
	}
	f.Page, f.PageSize = parsePagination(c)
	items, total, err := h.svc.List(f)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	paginatedResponse(c, items, total, f.Page, f.PageSize)
}

func (h *WorkScheduleHandler) Update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req updateWorkScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var startDate, endDate *time.Time
	if req.StartDate != nil {
		parsed, err := parseDateString(*req.StartDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("start_date: %v", err)})
			return
		}
		startDate = &parsed
	}
	if req.EndDate != nil {
		parsed, err := parseDateString(*req.EndDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("end_date: %v", err)})
			return
		}
		endDate = &parsed
	}

	var weekdays []services.WeekdayInput
	if req.Weekdays != nil {
		weekdays = make([]services.WeekdayInput, len(req.Weekdays))
		for i, wd := range req.Weekdays {
			if wd.StaffIDs == nil {
				wd.StaffIDs = []int{}
			}
			weekdays[i] = services.WeekdayInput{
				Weekday:   wd.Weekday,
				StartTime: wd.StartTime,
				EndTime:   wd.EndTime,
				RoomID:    wd.RoomID,
				StaffIDs:  wd.StaffIDs,
			}
		}
	}

	sch, err := h.svc.Update(id, services.UpdateWorkScheduleInput{
		Name:      req.Name,
		StartDate: startDate,
		EndDate:   endDate,
		Enabled:   req.Enabled,
		Weekdays:  weekdays,
	})
	if err != nil {
		writeWorkScheduleError(c, err)
		return
	}
	c.JSON(http.StatusOK, sch)
}

func (h *WorkScheduleHandler) Delete(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.svc.Delete(id); err != nil {
		writeWorkScheduleError(c, err)
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

func (h *WorkScheduleHandler) AddStaff(c *gin.Context) {
	scheduleID, ok := parseID(c)
	if !ok {
		return
	}
	var req staffAssignmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.WeekdayID == 0 && req.RoomID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "must provide weekday_id or (room_id + weekday)"})
		return
	}
	assign, err := h.svc.AddStaff(scheduleID, services.StaffAssignmentInput{
		WeekdayID: req.WeekdayID,
		StaffID:   req.StaffID,
		RoomID:    req.RoomID,
		Weekday:   req.Weekday,
	})
	if err != nil {
		writeWorkScheduleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, assign)
}

func (h *WorkScheduleHandler) RemoveStaff(c *gin.Context) {
	scheduleID, ok := parseID(c)
	if !ok {
		return
	}
	var req staffAssignmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.svc.RemoveStaff(scheduleID, services.StaffAssignmentInput{
		WeekdayID: req.WeekdayID,
		StaffID:   req.StaffID,
	}); err != nil {
		writeWorkScheduleError(c, err)
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

func (h *WorkScheduleHandler) ListValidStaff(c *gin.Context) {
	scheduleID, ok := parseID(c)
	if !ok {
		return
	}
	staff, err := h.svc.ListValidStaff(scheduleID)
	if err != nil {
		writeWorkScheduleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": staff})
}

func (h *WorkScheduleHandler) ListStaff(c *gin.Context) {
	scheduleID, ok := parseID(c)
	if !ok {
		return
	}
	staff, err := h.svc.ListStaff(scheduleID)
	if err != nil {
		writeWorkScheduleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": staff})
}

var workScheduleErrorMappings = []errStatus{
	{services.ErrWorkScheduleNotFound, http.StatusNotFound},
	{services.ErrWorkScheduleNameTaken, http.StatusConflict},
	{services.ErrWorkScheduleAlreadyEnabled, http.StatusConflict},
	{services.ErrWorkScheduleInvalidDateRange, http.StatusBadRequest},
	{services.ErrWorkScheduleInvalidWeekday, http.StatusBadRequest},
	{services.ErrWorkScheduleInvalidTimeWindow, http.StatusBadRequest},
	{services.ErrWorkScheduleRoomNotFound, http.StatusNotFound},
	{services.ErrWorkScheduleStaffNotFound, http.StatusNotFound},
	{services.ErrWorkScheduleStaffNotInWorkYear, http.StatusBadRequest},
	{services.ErrWorkScheduleWeekdayNotFound, http.StatusNotFound},
}

func writeWorkScheduleError(c *gin.Context, err error) {
	writeMappedError(c, err, workScheduleErrorMappings)
}

func boolPtr(v bool) *bool { return &v }
