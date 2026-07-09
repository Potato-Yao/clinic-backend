package tests

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"clinic-backend/models"
	"clinic-backend/services"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupServiceDateServiceDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	if err := db.AutoMigrate(&models.ClinicRoom{}, &models.ClinicServiceDate{}, &models.ClinicRecord{}); err != nil {
		t.Fatalf("failed to migrate models: %v", err)
	}
	return db
}

func mustCreateRoom(t *testing.T, db *gorm.DB, id uint) {
	t.Helper()
	room := models.ClinicRoom{ID: id, Address: "A", Enabled: true}
	room.Name = fmt.Sprintf("R-%d", id)
	if err := db.Create(&room).Error; err != nil {
		t.Fatalf("seed room failed: %v", err)
	}
}

func futureDateTime(daysFromNow int, hour, minute int) time.Time {
	d := futureDate(daysFromNow)
	return time.Date(d.Year(), d.Month(), d.Day(), hour, minute, 0, 0, time.UTC)
}

func validServiceDateInput(roomID uint, daysFromNow int, cap uint) services.CreateServiceDateInput {
	return services.CreateServiceDateInput{
		Capacity:  cap,
		RoomID:    roomID,
		Date:      futureDate(daysFromNow),
		StartTime: futureDateTime(daysFromNow, 9, 0),
		EndTime:   futureDateTime(daysFromNow, 12, 0),
		Title:     "Open Clinic",
	}
}

func TestServiceDateService_Create(t *testing.T) {
	db := setupServiceDateServiceDB(t)
	svc := services.NewServiceDateService(db)
	mustCreateRoom(t, db, 1)

	d, err := svc.Create(validServiceDateInput(1, 5, 10))
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if d.ID == 0 {
		t.Fatal("expected ID to be assigned")
	}
	if d.Capacity != 10 || d.Title != "Open Clinic" {
		t.Errorf("unexpected service date: %+v", d)
	}
}

func TestServiceDateService_GetByID_NotFound(t *testing.T) {
	db := setupServiceDateServiceDB(t)
	svc := services.NewServiceDateService(db)

	_, err := svc.GetByID(999)
	if !errors.Is(err, services.ErrServiceDateNotFound) {
		t.Fatalf("expected ErrServiceDateNotFound, got %v", err)
	}
}

func TestServiceDateService_Create_RoomNotFound(t *testing.T) {
	db := setupServiceDateServiceDB(t)
	svc := services.NewServiceDateService(db)

	_, err := svc.Create(validServiceDateInput(999, 5, 10))
	if !errors.Is(err, services.ErrServiceDateRoomNotFound) {
		t.Fatalf("expected ErrServiceDateRoomNotFound, got %v", err)
	}
}

func TestServiceDateService_List_Filter(t *testing.T) {
	db := setupServiceDateServiceDB(t)
	svc := services.NewServiceDateService(db)

	room1 := uint(1)
	room2 := uint(2)
	mustCreateServiceDate(t, db, svc, validServiceDateInput(room1, -1, 5)) // past
	mustCreateServiceDate(t, db, svc, validServiceDateInput(room1, 3, 5))
	mustCreateServiceDate(t, db, svc, validServiceDateInput(room2, 7, 5))

	t.Run("all", func(t *testing.T) {
		items, total, err := svc.List(services.ListServiceDateFilter{})
		if err != nil {
			t.Fatalf("list failed: %v", err)
		}
		if total != 3 || len(items) != 3 {
			t.Fatalf("expected 3 items, got %d (total %d)", len(items), total)
		}
		// Order should be by date asc, so the past one comes first.
		if !items[0].Date.Before(futureDate(0)) {
			t.Errorf("expected first item to be past date, got %v", items[0].Date)
		}
	})

	t.Run("active_only", func(t *testing.T) {
		items, total, err := svc.List(services.ListServiceDateFilter{ActiveOnly: true})
		if err != nil {
			t.Fatalf("list failed: %v", err)
		}
		if total != 2 || len(items) != 2 {
			t.Fatalf("expected 2 active items, got %d (total %d)", len(items), total)
		}
	})

	t.Run("room_filter", func(t *testing.T) {
		items, total, err := svc.List(services.ListServiceDateFilter{RoomID: &room2, ActiveOnly: true})
		if err != nil {
			t.Fatalf("list failed: %v", err)
		}
		if total != 1 || len(items) != 1 || *items[0].RoomID != room2 {
			t.Fatalf("expected 1 room2 item, got %d (total %d)", len(items), total)
		}
	})

	t.Run("from_filter", func(t *testing.T) {
		items, total, err := svc.List(services.ListServiceDateFilter{FromDate: futureDate(4)})
		if err != nil {
			t.Fatalf("list failed: %v", err)
		}
		if total != 1 || len(items) != 1 {
			t.Fatalf("expected 1 item on/after day 4, got %d (total %d)", len(items), total)
		}
	})
}

func TestServiceDateService_List_HasCapacity(t *testing.T) {
	db := setupServiceDateServiceDB(t)
	svc := services.NewServiceDateService(db)

	roomID := uint(1)
	mustCreateRoom(t, db, roomID)
	d, err := svc.Create(validServiceDateInput(roomID, 5, 2))
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	// Seed two accepted records to fill the date.
	for i := 0; i < 2; i++ {
		if err := db.Create(&models.ClinicRecord{
			Realname:        "x",
			PhoneNum:        "000",
			Status:          models.RecordStatusConfirmed,
			AppointmentTime: d.Date,
			RoomID:          roomID,
		}).Error; err != nil {
			t.Fatalf("seed record failed: %v", err)
		}
	}

	items, total, err := svc.List(services.ListServiceDateFilter{ActiveOnly: true, HasCapacity: true})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if total != 1 {
		t.Fatalf("total counts all rows; expected 1, got %d", total)
	}
	if len(items) != 0 {
		t.Fatalf("expected 0 available items since full, got %d", len(items))
	}
}

func TestServiceDateService_Update_Partial(t *testing.T) {
	db := setupServiceDateServiceDB(t)
	svc := services.NewServiceDateService(db)

	d := mustCreateServiceDate(t, db, svc, validServiceDateInput(1, 5, 10))
	newTitle := "Closed Day"
	updated, err := svc.Update(d.ID, services.UpdateServiceDateInput{Title: &newTitle})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if updated.Title != "Closed Day" || updated.Capacity != 10 {
		t.Errorf("unexpected: %+v", updated)
	}
}

func TestServiceDateService_Update_NotFound(t *testing.T) {
	db := setupServiceDateServiceDB(t)
	svc := services.NewServiceDateService(db)

	newTitle := "x"
	_, err := svc.Update(999, services.UpdateServiceDateInput{Title: &newTitle})
	if !errors.Is(err, services.ErrServiceDateNotFound) {
		t.Fatalf("expected ErrServiceDateNotFound, got %v", err)
	}
}

func TestServiceDateService_Update_RoomNotFound(t *testing.T) {
	db := setupServiceDateServiceDB(t)
	svc := services.NewServiceDateService(db)
	d := mustCreateServiceDate(t, db, svc, validServiceDateInput(1, 5, 10))

	badRoom := uint(999)
	_, err := svc.Update(d.ID, services.UpdateServiceDateInput{RoomID: &badRoom})
	if !errors.Is(err, services.ErrServiceDateRoomNotFound) {
		t.Fatalf("expected ErrServiceDateRoomNotFound, got %v", err)
	}
}

func TestServiceDateService_Update_InUse(t *testing.T) {
	db := setupServiceDateServiceDB(t)
	svc := services.NewServiceDateService(db)

	d := mustCreateServiceDate(t, db, svc, validServiceDateInput(1, 5, 10))
	if err := db.Create(&models.ClinicRecord{
		Realname:        "x",
		PhoneNum:        "000",
		Status:          models.RecordStatusPending,
		AppointmentTime: d.Date,
		RoomID:          1,
	}).Error; err != nil {
		t.Fatalf("seed record failed: %v", err)
	}

	newTitle := "Too Late"
	_, err := svc.Update(d.ID, services.UpdateServiceDateInput{Title: &newTitle})
	if !errors.Is(err, services.ErrServiceDateInUse) {
		t.Fatalf("expected ErrServiceDateInUse, got %v", err)
	}
}

func TestServiceDateService_Delete_Success(t *testing.T) {
	db := setupServiceDateServiceDB(t)
	svc := services.NewServiceDateService(db)

	d := mustCreateServiceDate(t, db, svc, validServiceDateInput(1, 5, 10))
	if err := svc.Delete(d.ID); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if _, err := svc.GetByID(d.ID); !errors.Is(err, services.ErrServiceDateNotFound) {
		t.Fatalf("expected not found after delete, got %v", err)
	}
}

func TestServiceDateService_Delete_NotFound(t *testing.T) {
	db := setupServiceDateServiceDB(t)
	svc := services.NewServiceDateService(db)

	if err := svc.Delete(999); !errors.Is(err, services.ErrServiceDateNotFound) {
		t.Fatalf("expected ErrServiceDateNotFound, got %v", err)
	}
}

func TestServiceDateService_Delete_InUse(t *testing.T) {
	db := setupServiceDateServiceDB(t)
	svc := services.NewServiceDateService(db)

	d := mustCreateServiceDate(t, db, svc, validServiceDateInput(1, 5, 10))
	if err := db.Create(&models.ClinicRecord{
		Realname:        "x",
		PhoneNum:        "000",
		Status:          models.RecordStatusPending,
		AppointmentTime: d.Date,
		RoomID:          1,
	}).Error; err != nil {
		t.Fatalf("seed record failed: %v", err)
	}

	if err := svc.Delete(d.ID); !errors.Is(err, services.ErrServiceDateInUse) {
		t.Fatalf("expected ErrServiceDateInUse, got %v", err)
	}
}

func mustCreateServiceDate(t *testing.T, db *gorm.DB, svc *services.ServiceDateService, in services.CreateServiceDateInput) models.ClinicServiceDate {
	t.Helper()
	mustEnsureRoom(t, db, in.RoomID)
	d, err := svc.Create(in)
	if err != nil {
		t.Fatalf("seed service date create failed: %v", err)
	}
	return d
}

// mustEnsureRoom inserts a room with the given id if it does not already exist.
func mustEnsureRoom(t *testing.T, db *gorm.DB, roomID uint) {
	t.Helper()
	var n int64
	if err := db.Model(&models.ClinicRoom{}).Where("id = ?", roomID).Count(&n).Error; err != nil {
		t.Fatalf("count room %d: %v", roomID, err)
	}
	if n == 0 {
		mustCreateRoom(t, db, roomID)
	}
}
