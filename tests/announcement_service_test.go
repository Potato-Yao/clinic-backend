package tests

import (
	"errors"
	"testing"
	"time"

	"clinic-backend/models"
	"clinic-backend/services"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupAnnouncementServiceDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	if err := db.AutoMigrate(&models.ClinicAnnouncement{}); err != nil {
		t.Fatalf("failed to migrate ClinicAnnouncement model: %v", err)
	}
	return db
}

func futureDate(days int) time.Time {
	return time.Now().UTC().Add(time.Duration(days) * 24 * time.Hour).Truncate(24 * time.Hour)
}

func TestAnnouncementService_Create(t *testing.T) {
	db := setupAnnouncementServiceDB(t)
	svc := services.NewAnnouncementService(db)

	before := time.Now().UTC()
	a, err := svc.Create(services.CreateAnnouncementInput{
		Title:      "System Maintenance",
		Content:    "The clinic will be closed.",
		Tag:        "normal",
		Brief:      "Clinic closed",
		ExpireDate: futureDate(7),
		Priority:   2,
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if a.ID == 0 {
		t.Fatal("expected ID to be assigned")
	}
	if a.CreatedTime.Before(before) {
		t.Errorf("expected CreatedTime to be set, got %v", a.CreatedTime)
	}
	if !a.LastEditedTime.Equal(a.CreatedTime) {
		t.Errorf("expected LastEditedTime == CreatedTime on create, got %v vs %v", a.LastEditedTime, a.CreatedTime)
	}
}

func TestAnnouncementService_GetByID_NotFound(t *testing.T) {
	db := setupAnnouncementServiceDB(t)
	svc := services.NewAnnouncementService(db)

	_, err := svc.GetByID(999)
	if !errors.Is(err, services.ErrAnnouncementNotFound) {
		t.Fatalf("expected ErrAnnouncementNotFound, got %v", err)
	}
}

func TestAnnouncementService_List_FilterAndSort(t *testing.T) {
	db := setupAnnouncementServiceDB(t)
	svc := services.NewAnnouncementService(db)

	// Two active, one expired; varying priority and tag.
	seed := []services.CreateAnnouncementInput{
		{Title: "A", Content: "c", Tag: "normal", Brief: "b", ExpireDate: futureDate(5), Priority: 1},
		{Title: "B", Content: "c", Tag: "normal", Brief: "b", ExpireDate: futureDate(10), Priority: 3},
		{Title: "C", Content: "c", Tag: "pinned", Brief: "b", ExpireDate: futureDate(-1), Priority: 5},
	}
	for _, in := range seed {
		if _, err := svc.Create(in); err != nil {
			t.Fatalf("seed create failed: %v", err)
		}
	}

	t.Run("all", func(t *testing.T) {
		items, total, err := svc.List(services.ListAnnouncementFilter{})
		if err != nil {
			t.Fatalf("list failed: %v", err)
		}
		if total != 3 || len(items) != 3 {
			t.Fatalf("expected 3 items, got %d (total %d)", len(items), total)
		}
		// priority desc: C(5), B(3), A(1)
		if items[0].Title != "C" || items[1].Title != "B" || items[2].Title != "A" {
			t.Errorf("unexpected order: %s, %s, %s", items[0].Title, items[1].Title, items[2].Title)
		}
	})

	t.Run("active_only", func(t *testing.T) {
		items, total, err := svc.List(services.ListAnnouncementFilter{ActiveOnly: true})
		if err != nil {
			t.Fatalf("list failed: %v", err)
		}
		if total != 2 || len(items) != 2 {
			t.Fatalf("expected 2 active items, got %d (total %d)", len(items), total)
		}
		for _, a := range items {
			if a.Title == "C" {
				t.Errorf("expired item C should not appear in active list")
			}
		}
	})

	t.Run("tag_filter", func(t *testing.T) {
		items, total, err := svc.List(services.ListAnnouncementFilter{Tag: "pinned"})
		if err != nil {
			t.Fatalf("list failed: %v", err)
		}
		if total != 1 || len(items) != 1 || items[0].Title != "C" {
			t.Fatalf("expected 1 alert item, got %d (total %d)", len(items), total)
		}
	})
}

func TestAnnouncementService_List_Pagination(t *testing.T) {
	db := setupAnnouncementServiceDB(t)
	svc := services.NewAnnouncementService(db)

	for i := 0; i < 5; i++ {
		if _, err := svc.Create(services.CreateAnnouncementInput{
			Title:      "T",
			Content:    "c",
			Tag:        "normal",
			Brief:      "b",
			ExpireDate: futureDate(1),
			Priority:   uint(i),
		}); err != nil {
			t.Fatalf("seed create failed: %v", err)
		}
	}

	items, total, err := svc.List(services.ListAnnouncementFilter{Page: 2, PageSize: 2})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if total != 5 {
		t.Fatalf("expected total 5, got %d", total)
	}
	if len(items) != 2 {
		t.Fatalf("expected page size 2, got %d", len(items))
	}
}

func TestAnnouncementService_Update_PartialAndTimestamp(t *testing.T) {
	db := setupAnnouncementServiceDB(t)
	svc := services.NewAnnouncementService(db)

	a, err := svc.Create(services.CreateAnnouncementInput{
		Title:      "Old",
		Content:    "old content",
		Tag:        "normal",
		Brief:      "old brief",
		ExpireDate: futureDate(3),
		Priority:   1,
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	created := a.LastEditedTime

	time.Sleep(10 * time.Millisecond)
	newTitle := "New Title"
	updated, err := svc.Update(a.ID, services.UpdateAnnouncementInput{Title: &newTitle})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if updated.Title != "New Title" {
		t.Errorf("expected title updated, got %q", updated.Title)
	}
	if updated.Content != "old content" {
		t.Errorf("expected content unchanged, got %q", updated.Content)
	}
	if !updated.LastEditedTime.After(created) {
		t.Errorf("expected LastEditedTime to advance, got %v (was %v)", updated.LastEditedTime, created)
	}
}

func TestAnnouncementService_Update_NotFound(t *testing.T) {
	db := setupAnnouncementServiceDB(t)
	svc := services.NewAnnouncementService(db)

	newTitle := "x"
	_, err := svc.Update(999, services.UpdateAnnouncementInput{Title: &newTitle})
	if !errors.Is(err, services.ErrAnnouncementNotFound) {
		t.Fatalf("expected ErrAnnouncementNotFound, got %v", err)
	}
}

func TestAnnouncementService_Delete(t *testing.T) {
	db := setupAnnouncementServiceDB(t)
	svc := services.NewAnnouncementService(db)

	a, err := svc.Create(services.CreateAnnouncementInput{
		Title:      "X",
		Content:    "c",
		Tag:        "normal",
		Brief:      "b",
		ExpireDate: futureDate(1),
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if err := svc.Delete(a.ID); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if _, err := svc.GetByID(a.ID); !errors.Is(err, services.ErrAnnouncementNotFound) {
		t.Fatalf("expected not found after delete, got %v", err)
	}
}

func TestAnnouncementService_Delete_NotFound(t *testing.T) {
	db := setupAnnouncementServiceDB(t)
	svc := services.NewAnnouncementService(db)

	if err := svc.Delete(999); !errors.Is(err, services.ErrAnnouncementNotFound) {
		t.Fatalf("expected ErrAnnouncementNotFound, got %v", err)
	}
}

func TestAnnouncementService_Create_DefaultTag(t *testing.T) {
	db := setupAnnouncementServiceDB(t)
	svc := services.NewAnnouncementService(db)

	a, err := svc.Create(services.CreateAnnouncementInput{
		Title:      "Default Tag",
		Content:    "test",
		Brief:      "test",
		ExpireDate: futureDate(1),
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if a.Tag != models.AnnouncementTagNormal {
		t.Errorf("expected default tag %q, got %q", models.AnnouncementTagNormal, a.Tag)
	}
}

func TestAnnouncementService_Create_InvalidTag(t *testing.T) {
	db := setupAnnouncementServiceDB(t)
	svc := services.NewAnnouncementService(db)

	_, err := svc.Create(services.CreateAnnouncementInput{
		Title:      "Bad Tag",
		Content:    "test",
		Tag:        "invalid",
		Brief:      "test",
		ExpireDate: futureDate(1),
	})
	if !errors.Is(err, services.ErrAnnouncementInvalidTag) {
		t.Fatalf("expected ErrAnnouncementInvalidTag, got %v", err)
	}
}

func TestAnnouncementService_GetTOS_Found(t *testing.T) {
	db := setupAnnouncementServiceDB(t)
	svc := services.NewAnnouncementService(db)

	_, err := svc.Create(services.CreateAnnouncementInput{
		Title:      "Terms of Service",
		Content:    "You agree to...",
		Tag:        "tos",
		Brief:      "TOS brief",
		ExpireDate: futureDate(30),
	})
	if err != nil {
		t.Fatalf("seed tos failed: %v", err)
	}

	a, err := svc.GetTOS()
	if err != nil {
		t.Fatalf("GetTOS failed: %v", err)
	}
	if a.Tag != models.AnnouncementTagTOS {
		t.Errorf("expected tag tos, got %q", a.Tag)
	}
	if a.Title != "Terms of Service" {
		t.Errorf("expected title %q, got %q", "Terms of Service", a.Title)
	}
}

func TestAnnouncementService_GetTOS_NotFound(t *testing.T) {
	db := setupAnnouncementServiceDB(t)
	svc := services.NewAnnouncementService(db)

	_, err := svc.GetTOS()
	if !errors.Is(err, services.ErrAnnouncementNotFound) {
		t.Fatalf("expected ErrAnnouncementNotFound, got %v", err)
	}
}

func TestAnnouncementService_Create_DuplicateTOS(t *testing.T) {
	db := setupAnnouncementServiceDB(t)
	svc := services.NewAnnouncementService(db)

	_, err := svc.Create(services.CreateAnnouncementInput{
		Title:      "TOS One",
		Content:    "terms",
		Tag:        "tos",
		Brief:      "tos",
		ExpireDate: futureDate(1),
	})
	if err != nil {
		t.Fatalf("first tos create failed: %v", err)
	}

	_, err = svc.Create(services.CreateAnnouncementInput{
		Title:      "TOS Two",
		Content:    "terms",
		Tag:        "tos",
		Brief:      "tos",
		ExpireDate: futureDate(1),
	})
	if !errors.Is(err, services.ErrAnnouncementTOSAlreadyExists) {
		t.Fatalf("expected ErrAnnouncementTOSAlreadyExists, got %v", err)
	}
}

func TestAnnouncementService_Update_DuplicateTOS(t *testing.T) {
	db := setupAnnouncementServiceDB(t)
	svc := services.NewAnnouncementService(db)

	a1, err := svc.Create(services.CreateAnnouncementInput{
		Title:      "TOS One",
		Content:    "terms",
		Tag:        "tos",
		Brief:      "tos",
		ExpireDate: futureDate(1),
	})
	if err != nil {
		t.Fatalf("first tos create failed: %v", err)
	}

	a2, err := svc.Create(services.CreateAnnouncementInput{
		Title:      "Normal",
		Content:    "test",
		Tag:        "normal",
		Brief:      "test",
		ExpireDate: futureDate(1),
	})
	if err != nil {
		t.Fatalf("normal create failed: %v", err)
	}

	tos := models.AnnouncementTagTOS
	_, err = svc.Update(a2.ID, services.UpdateAnnouncementInput{Tag: &tos})
	if !errors.Is(err, services.ErrAnnouncementTOSAlreadyExists) {
		t.Fatalf("expected ErrAnnouncementTOSAlreadyExists, got %v", err)
	}

	// Updating the existing TOS to itself should succeed.
	updated, err := svc.Update(a1.ID, services.UpdateAnnouncementInput{Tag: &tos})
	if err != nil {
		t.Fatalf("re-setting tos on same announcement should succeed: %v", err)
	}
	if updated.Tag != models.AnnouncementTagTOS {
		t.Errorf("expected tag to remain tos, got %q", updated.Tag)
	}
}
