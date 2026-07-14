package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"clinic-backend/handlers"
	"clinic-backend/models"
	"clinic-backend/services"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupAnnouncementHandlerRouter(t *testing.T) (*gin.Engine, *services.AnnouncementService) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	if err := db.AutoMigrate(&models.ClinicAnnouncement{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	svc := services.NewAnnouncementService(db)
	h := handlers.NewAnnouncementHandler(svc)

	r := gin.New()
	g := r.Group("/api/admin/announcements")
	{
		g.POST("", h.Create)
		g.GET("", h.List)
		g.GET("/:id", h.Get)
		g.PUT("/:id", h.Update)
		g.DELETE("/:id", h.Delete)
	}
	return r, svc
}

func doRequest(t *testing.T, r *gin.Engine, method, path string, body any) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("failed to encode body: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestAnnouncementHandler_Create_Success(t *testing.T) {
	r, _ := setupAnnouncementHandlerRouter(t)

	w := doRequest(t, r, http.MethodPost, "/api/admin/announcements", map[string]any{
		"title":      "Hello",
		"content":    "body text",
		"tag":        "normal",
		"brief":      "short",
		"expireDate": futureDate(7).Format(time.RFC3339),
		"priority":   1,
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var got models.ClinicAnnouncement
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if got.Title != "Hello" || got.ID == 0 {
		t.Errorf("unexpected announcement: %+v", got)
	}
}

func TestAnnouncementHandler_Create_Validation(t *testing.T) {
	r, _ := setupAnnouncementHandlerRouter(t)

	cases := []struct {
		name string
		body map[string]any
	}{
		{"missing title", map[string]any{"content": "c", "brief": "b", "expireDate": futureDate(1).Format(time.RFC3339)}},
		{"title too long", map[string]any{"title": "012345678901234567890", "content": "c", "brief": "b", "expireDate": futureDate(1).Format(time.RFC3339)}},
		{"past expireDate", map[string]any{"title": "T", "content": "c", "brief": "b", "expireDate": "2000-01-01T00:00:00Z"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := doRequest(t, r, http.MethodPost, "/api/admin/announcements", c.body)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
			}
		})
	}
}

func TestAnnouncementHandler_Get_NotFound(t *testing.T) {
	r, _ := setupAnnouncementHandlerRouter(t)

	w := doRequest(t, r, http.MethodGet, "/api/admin/announcements/999", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAnnouncementHandler_List(t *testing.T) {
	r, svc := setupAnnouncementHandlerRouter(t)
	for i := 0; i < 3; i++ {
		if _, err := svc.Create(services.CreateAnnouncementInput{
			Title: "T", Content: "c", Tag: "normal", Brief: "b", ExpireDate: futureDate(1), Priority: uint(i),
		}); err != nil {
			t.Fatalf("seed failed: %v", err)
		}
	}

	w := doRequest(t, r, http.MethodGet, "/api/admin/announcements?page=1&pageSize=2", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Items    []models.ClinicAnnouncement `json:"items"`
		Total    int64                       `json:"total"`
		Page     int                         `json:"page"`
		PageSize int                         `json:"pageSize"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if resp.Total != 3 || len(resp.Items) != 2 {
		t.Errorf("expected total 3 / 2 items, got total %d / %d items", resp.Total, len(resp.Items))
	}
}

func TestAnnouncementHandler_Update_Partial(t *testing.T) {
	r, svc := setupAnnouncementHandlerRouter(t)
	a, err := svc.Create(services.CreateAnnouncementInput{
		Title: "Old", Content: "c", Tag: "normal", Brief: "b", ExpireDate: futureDate(2),
	})
	if err != nil {
		t.Fatalf("seed failed: %v", err)
	}

	w := doRequest(t, r, http.MethodPut, "/api/admin/announcements/"+itoa(a.ID), map[string]any{"title": "New"})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var got models.ClinicAnnouncement
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if got.Title != "New" || got.Content != "c" {
		t.Errorf("expected title New and content unchanged, got %+v", got)
	}
}

func TestAnnouncementHandler_Delete_Success(t *testing.T) {
	r, svc := setupAnnouncementHandlerRouter(t)
	a, err := svc.Create(services.CreateAnnouncementInput{
		Title: "X", Content: "c", Tag: "normal", Brief: "b", ExpireDate: futureDate(1),
	})
	if err != nil {
		t.Fatalf("seed failed: %v", err)
	}

	w := doRequest(t, r, http.MethodDelete, "/api/admin/announcements/"+itoa(a.ID), nil)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAnnouncementHandler_InvalidID(t *testing.T) {
	r, _ := setupAnnouncementHandlerRouter(t)
	w := doRequest(t, r, http.MethodGet, "/api/admin/announcements/abc", nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func itoa(n uint) string {
	return strconv.Itoa(int(n))
}
