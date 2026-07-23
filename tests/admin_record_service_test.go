package tests

import (
	"context"
	"testing"
	"time"

	"clinic-backend/models"
	"clinic-backend/services"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupAdminRecordTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:?_fk=1"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(
		&models.ClinicRoom{},
		&models.ClinicRecord{},
		&models.ClinicRecordDevice{},
		&models.ClinicRecordWorker{},
		&models.ClinicRecordArrival{},
		&models.ClinicRecordRejection{},
		&models.ClinicRecordReferral{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func yesterdayCutoff() time.Time {
	t := time.Now().UTC().AddDate(0, 0, -1)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func TestCloseExpiredRecords_NoShow(t *testing.T) {
	db := setupAdminRecordTestDB(t)
	svc := services.NewAdminRecordService(db)
	cutoff := yesterdayCutoff()
	pastDate := cutoff.AddDate(0, 0, -1)

	room := models.ClinicRoom{Name: "Test Room"}
	if err := db.Create(&room).Error; err != nil {
		t.Fatalf("seed room: %v", err)
	}

	for _, st := range []models.RecordStatus{
		models.RecordStatusPending,
		models.RecordStatusConfirmed,
		models.RecordStatusArrived,
	} {
		if err := db.Create(&models.ClinicRecord{
			User: "u", Realname: "r", PhoneNum: "p",
			Status: st, AppointmentTime: pastDate,
			QuestionDesc: "x", RoomID: room.ID,
		}).Error; err != nil {
			t.Fatalf("seed %s: %v", st, err)
		}
	}

	noShow, completed, err := svc.CloseExpiredRecords(context.Background(), cutoff)
	if err != nil {
		t.Fatalf("CloseExpiredRecords: %v", err)
	}
	if noShow != 3 {
		t.Errorf("expected 3 no_shows, got %d", noShow)
	}
	if completed != 0 {
		t.Errorf("expected 0 completed, got %d", completed)
	}

	var count int64
	db.Model(&models.ClinicRecord{}).Where("status = ?", models.RecordStatusNoShow).Count(&count)
	if count != 3 {
		t.Errorf("expected 3 no_show records, got %d", count)
	}
}

func TestCloseExpiredRecords_Completed(t *testing.T) {
	db := setupAdminRecordTestDB(t)
	svc := services.NewAdminRecordService(db)
	cutoff := yesterdayCutoff()
	pastDate := cutoff.AddDate(0, 0, -1)

	room := models.ClinicRoom{Name: "Test Room"}
	if err := db.Create(&room).Error; err != nil {
		t.Fatalf("seed room: %v", err)
	}

	rec := models.ClinicRecord{
		User: "u", Realname: "r", PhoneNum: "p",
		Status: models.RecordStatusInProgress, AppointmentTime: pastDate,
		QuestionDesc: "x", RoomID: room.ID,
	}
	if err := db.Create(&rec).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := db.Create(&models.ClinicRecordWorker{
		RecordID:   rec.ID,
		WorkerID:   1,
		WorkerDesc: "working",
	}).Error; err != nil {
		t.Fatalf("seed worker: %v", err)
	}

	noShow, completed, err := svc.CloseExpiredRecords(context.Background(), cutoff)
	if err != nil {
		t.Fatalf("CloseExpiredRecords: %v", err)
	}
	if noShow != 0 {
		t.Errorf("expected 0 no_shows, got %d", noShow)
	}
	if completed != 1 {
		t.Errorf("expected 1 completed, got %d", completed)
	}

	var updated models.ClinicRecord
	if err := db.First(&updated, rec.ID).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if updated.Status != models.RecordStatusCompleted {
		t.Errorf("expected completed, got %s", updated.Status)
	}

	var worker models.ClinicRecordWorker
	if err := db.Where("record_id = ?", rec.ID).First(&worker).Error; err != nil {
		t.Fatalf("reload worker: %v", err)
	}
	if worker.FinishTime.IsZero() {
		t.Errorf("expected finish_time to be set, got zero")
	}
}

func TestCloseExpiredRecords_IgnoresTerminal(t *testing.T) {
	db := setupAdminRecordTestDB(t)
	svc := services.NewAdminRecordService(db)
	cutoff := yesterdayCutoff()
	pastDate := cutoff.AddDate(0, 0, -1)

	room := models.ClinicRoom{Name: "Test Room"}
	if err := db.Create(&room).Error; err != nil {
		t.Fatalf("seed room: %v", err)
	}

	for _, st := range []models.RecordStatus{
		models.RecordStatusCompleted,
		models.RecordStatusRejected,
		models.RecordStatusReferred,
		models.RecordStatusNoShow,
	} {
		if err := db.Create(&models.ClinicRecord{
			User: "u", Realname: "r", PhoneNum: "p",
			Status: st, AppointmentTime: pastDate,
			QuestionDesc: "x", RoomID: room.ID,
		}).Error; err != nil {
			t.Fatalf("seed %s: %v", st, err)
		}
	}

	noShow, completed, err := svc.CloseExpiredRecords(context.Background(), cutoff)
	if err != nil {
		t.Fatalf("CloseExpiredRecords: %v", err)
	}
	if noShow != 0 {
		t.Errorf("expected 0 no_shows, got %d", noShow)
	}
	if completed != 0 {
		t.Errorf("expected 0 completed, got %d", completed)
	}
}

func TestCloseExpiredRecords_IgnoresFutureTickets(t *testing.T) {
	db := setupAdminRecordTestDB(t)
	svc := services.NewAdminRecordService(db)
	cutoff := yesterdayCutoff()
	futureDate := cutoff.AddDate(0, 0, 3)

	room := models.ClinicRoom{Name: "Test Room"}
	if err := db.Create(&room).Error; err != nil {
		t.Fatalf("seed room: %v", err)
	}

	for _, st := range []models.RecordStatus{
		models.RecordStatusConfirmed,
		models.RecordStatusInProgress,
	} {
		if err := db.Create(&models.ClinicRecord{
			User: "u", Realname: "r", PhoneNum: "p",
			Status: st, AppointmentTime: futureDate,
			QuestionDesc: "x", RoomID: room.ID,
		}).Error; err != nil {
			t.Fatalf("seed %s: %v", st, err)
		}
	}

	noShow, completed, err := svc.CloseExpiredRecords(context.Background(), cutoff)
	if err != nil {
		t.Fatalf("CloseExpiredRecords: %v", err)
	}
	if noShow != 0 {
		t.Errorf("expected 0 no_shows, got %d", noShow)
	}
	if completed != 0 {
		t.Errorf("expected 0 completed, got %d", completed)
	}
}
