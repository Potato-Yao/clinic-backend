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

func setupWorkScheduleServiceDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	if err := db.AutoMigrate(
		&models.ClinicWorkSchedule{},
		&models.ClinicWorkScheduleWeekday{},
		&models.ClinicWorkScheduleStaff{},
		&models.ClinicRoom{},
		&models.ClinicStaff{},
		&models.ClinicStaffWorkyear{},
		&models.ClinicRecordWorker{},
	); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

func seedRoom(t *testing.T, db *gorm.DB, name string) uint {
	t.Helper()
	r := models.ClinicRoom{Name: name, Address: "addr"}
	if err := db.Create(&r).Error; err != nil {
		t.Fatalf("seed room: %v", err)
	}
	return r.ID
}

func seedOneStaff(t *testing.T, db *gorm.DB) int {
	t.Helper()
	s := models.ClinicStaff{AccountID: "test-staff"}
	if err := db.Create(&s).Error; err != nil {
		t.Fatalf("seed staff: %v", err)
	}
	return s.ID
}

func seedStaffWorkYear(t *testing.T, db *gorm.DB, staffID int, year int) {
	t.Helper()
	if err := db.Create(&models.ClinicStaffWorkyear{StaffID: staffID, WorkYear: year}).Error; err != nil {
		t.Fatalf("seed staff work year: %v", err)
	}
}

func seedStaff(t *testing.T, db *gorm.DB, n int) []int {
	ids := make([]int, n)
	for i := 0; i < n; i++ {
		s := models.ClinicStaff{AccountID: fmt.Sprintf("staff-%d", i)}
		if err := db.Create(&s).Error; err != nil {
			t.Fatalf("seed staff %d: %v", i, err)
		}
		ids[i] = s.ID
	}
	return ids
}

func TestWorkScheduleService_Create_Success(t *testing.T) {
	db := setupWorkScheduleServiceDB(t)
	svc := services.NewWorkScheduleService(db)

	roomID := seedRoom(t, db, "Room A")
	staffIDs := seedStaff(t, db, 2)

	sch, err := svc.Create(services.CreateWorkScheduleInput{
		Name:      "Fall 2026",
		StartDate: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
		Enabled:   false,
		Weekdays: []services.WeekdayInput{
			{Weekday: 1, StartTime: "09:00:00", EndTime: "12:00:00", RoomID: roomID, StaffIDs: staffIDs},
		},
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if sch.ID == 0 {
		t.Fatal("expected ID assigned")
	}
	if sch.Name != "Fall 2026" {
		t.Errorf("expected name 'Fall 2026', got %q", sch.Name)
	}
	if sch.Enabled {
		t.Errorf("expected enabled false by default")
	}
}

func TestWorkScheduleService_Create_Enabled_OnlyOne(t *testing.T) {
	db := setupWorkScheduleServiceDB(t)
	svc := services.NewWorkScheduleService(db)

	_, err := svc.Create(services.CreateWorkScheduleInput{
		Name:      "First",
		StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
		Enabled:   true,
	})
	if err != nil {
		t.Fatalf("first create enabled failed: %v", err)
	}

	_, err = svc.Create(services.CreateWorkScheduleInput{
		Name:      "Second",
		StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
		Enabled:   true,
	})
	if !errors.Is(err, services.ErrWorkScheduleAlreadyEnabled) {
		t.Fatalf("expected ErrWorkScheduleAlreadyEnabled, got %v", err)
	}
}

func TestWorkScheduleService_Create_DuplicateName(t *testing.T) {
	db := setupWorkScheduleServiceDB(t)
	svc := services.NewWorkScheduleService(db)

	_, err := svc.Create(services.CreateWorkScheduleInput{
		Name:      "Same",
		StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("first create failed: %v", err)
	}
	_, err = svc.Create(services.CreateWorkScheduleInput{
		Name:      "Same",
		StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
	})
	if !errors.Is(err, services.ErrWorkScheduleNameTaken) {
		t.Fatalf("expected ErrWorkScheduleNameTaken, got %v", err)
	}
}

func TestWorkScheduleService_Create_InvalidDateRange(t *testing.T) {
	db := setupWorkScheduleServiceDB(t)
	svc := services.NewWorkScheduleService(db)

	_, err := svc.Create(services.CreateWorkScheduleInput{
		Name:      "Bad",
		StartDate: time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	})
	if !errors.Is(err, services.ErrWorkScheduleInvalidDateRange) {
		t.Fatalf("expected ErrWorkScheduleInvalidDateRange, got %v", err)
	}
}

func TestWorkScheduleService_Create_RoomNotFound(t *testing.T) {
	db := setupWorkScheduleServiceDB(t)
	svc := services.NewWorkScheduleService(db)

	_, err := svc.Create(services.CreateWorkScheduleInput{
		Name:      "Test",
		StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
		Weekdays: []services.WeekdayInput{
			{Weekday: 1, StartTime: "09:00", EndTime: "12:00", RoomID: 9999},
		},
	})
	if !errors.Is(err, services.ErrWorkScheduleRoomNotFound) {
		t.Fatalf("expected ErrWorkScheduleRoomNotFound, got %v", err)
	}
}

func TestWorkScheduleService_Create_StaffNotFound(t *testing.T) {
	db := setupWorkScheduleServiceDB(t)
	svc := services.NewWorkScheduleService(db)
	roomID := seedRoom(t, db, "R1")

	_, err := svc.Create(services.CreateWorkScheduleInput{
		Name:      "Test",
		StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
		Weekdays: []services.WeekdayInput{
			{Weekday: 1, StartTime: "09:00", EndTime: "12:00", RoomID: roomID, StaffIDs: []int{999}},
		},
	})
	if !errors.Is(err, services.ErrWorkScheduleStaffNotFound) {
		t.Fatalf("expected ErrWorkScheduleStaffNotFound, got %v", err)
	}
}

func TestWorkScheduleService_GetByID_NotFound(t *testing.T) {
	db := setupWorkScheduleServiceDB(t)
	svc := services.NewWorkScheduleService(db)

	_, err := svc.GetByID(999)
	if !errors.Is(err, services.ErrWorkScheduleNotFound) {
		t.Fatalf("expected ErrWorkScheduleNotFound, got %v", err)
	}
}

func TestWorkScheduleService_GetByID_WithNested(t *testing.T) {
	db := setupWorkScheduleServiceDB(t)
	svc := services.NewWorkScheduleService(db)

	roomID := seedRoom(t, db, "R1")
	staffID := seedOneStaff(t, db)

	created, err := svc.Create(services.CreateWorkScheduleInput{
		Name:      "Test",
		StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
		Weekdays: []services.WeekdayInput{
			{Weekday: 1, StartTime: "09:00", EndTime: "12:00", RoomID: roomID, StaffIDs: []int{staffID}},
			{Weekday: 3, StartTime: "13:00", EndTime: "17:00", RoomID: roomID},
		},
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	got, err := svc.GetByID(created.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if len(got.Weekdays) != 2 {
		t.Fatalf("expected 2 weekdays, got %d", len(got.Weekdays))
	}
	if got.Weekdays[0].Room.ID == 0 {
		t.Errorf("expected room preloaded")
	}
	if len(got.Weekdays[0].Staff) != 1 {
		t.Errorf("expected 1 staff on weekday 0, got %d", len(got.Weekdays[0].Staff))
	}
	if got.Weekdays[0].Staff[0].Staff.ID == 0 {
		t.Errorf("expected staff preloaded")
	}
	if got.Weekdays[0].Staff[0].ScheduleID != created.ID {
		t.Errorf("expected schedule_id %d, got %d", created.ID, got.Weekdays[0].Staff[0].ScheduleID)
	}
}

func TestWorkScheduleService_List_FilterEnabled(t *testing.T) {
	db := setupWorkScheduleServiceDB(t)
	svc := services.NewWorkScheduleService(db)

	_, err := svc.Create(services.CreateWorkScheduleInput{
		Name: "A", StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate: time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC), Enabled: false,
	})
	if err != nil {
		t.Fatalf("create A: %v", err)
	}
	_, err = svc.Create(services.CreateWorkScheduleInput{
		Name: "B", StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate: time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC), Enabled: true,
	})
	if err != nil {
		t.Fatalf("create B: %v", err)
	}

	allItems, total, err := svc.List(services.ListWorkScheduleFilter{})
	if err != nil {
		t.Fatalf("list all: %v", err)
	}
	if total != 2 || len(allItems) != 2 {
		t.Fatalf("expected 2 total, got %d / %d", total, len(allItems))
	}

	enabled := true
	enabledItems, total, err := svc.List(services.ListWorkScheduleFilter{Enabled: &enabled})
	if err != nil {
		t.Fatalf("list enabled: %v", err)
	}
	if total != 1 || len(enabledItems) != 1 {
		t.Fatalf("expected 1 enabled, got %d / %d", total, len(enabledItems))
	}
}

func TestWorkScheduleService_Update_Enable_OnlyOne(t *testing.T) {
	db := setupWorkScheduleServiceDB(t)
	svc := services.NewWorkScheduleService(db)

	first, err := svc.Create(services.CreateWorkScheduleInput{
		Name: "First", StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate: time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC), Enabled: true,
	})
	if err != nil {
		t.Fatalf("create first: %v", err)
	}

	second, err := svc.Create(services.CreateWorkScheduleInput{
		Name: "Second", StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate: time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC), Enabled: false,
	})
	if err != nil {
		t.Fatalf("create second: %v", err)
	}

	enabled := true
	_, err = svc.Update(second.ID, services.UpdateWorkScheduleInput{Enabled: &enabled})
	if !errors.Is(err, services.ErrWorkScheduleAlreadyEnabled) {
		t.Fatalf("expected ErrWorkScheduleAlreadyEnabled, got %v", err)
	}

	// Disable first, then enable second should work.
	disabled := false
	_, err = svc.Update(first.ID, services.UpdateWorkScheduleInput{Enabled: &disabled})
	if err != nil {
		t.Fatalf("disable first failed: %v", err)
	}
	_, err = svc.Update(second.ID, services.UpdateWorkScheduleInput{Enabled: &enabled})
	if err != nil {
		t.Fatalf("enable second after disabling first failed: %v", err)
	}
}

func TestWorkScheduleService_Update_ReplaceWeekdays(t *testing.T) {
	db := setupWorkScheduleServiceDB(t)
	svc := services.NewWorkScheduleService(db)
	roomID1 := seedRoom(t, db, "R1")
	roomID2 := seedRoom(t, db, "R2")

	created, err := svc.Create(services.CreateWorkScheduleInput{
		Name: "Test", StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate: time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
		Weekdays: []services.WeekdayInput{
			{Weekday: 1, StartTime: "09:00", EndTime: "12:00", RoomID: roomID1},
		},
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	updated, err := svc.Update(created.ID, services.UpdateWorkScheduleInput{
		Weekdays: []services.WeekdayInput{
			{Weekday: 2, StartTime: "10:00", EndTime: "14:00", RoomID: roomID2},
			{Weekday: 4, StartTime: "08:00", EndTime: "17:00", RoomID: roomID1},
		},
	})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if len(updated.Weekdays) != 2 {
		t.Fatalf("expected 2 weekdays after update, got %d", len(updated.Weekdays))
	}
	// weekday 1 should be gone, replaced by 2 and 4
	for _, wd := range updated.Weekdays {
		if wd.Weekday == 1 {
			t.Errorf("weekday 1 should have been replaced")
		}
	}
}

func TestWorkScheduleService_Update_NameDuplicate(t *testing.T) {
	db := setupWorkScheduleServiceDB(t)
	svc := services.NewWorkScheduleService(db)

	_, err := svc.Create(services.CreateWorkScheduleInput{
		Name: "A", StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate: time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("create A: %v", err)
	}
	b, err := svc.Create(services.CreateWorkScheduleInput{
		Name: "B", StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate: time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("create B: %v", err)
	}

	nameA := "A"
	_, err = svc.Update(b.ID, services.UpdateWorkScheduleInput{Name: &nameA})
	if !errors.Is(err, services.ErrWorkScheduleNameTaken) {
		t.Fatalf("expected ErrWorkScheduleNameTaken, got %v", err)
	}
}

func TestWorkScheduleService_Delete_Success(t *testing.T) {
	db := setupWorkScheduleServiceDB(t)
	svc := services.NewWorkScheduleService(db)
	roomID := seedRoom(t, db, "R1")

	created, err := svc.Create(services.CreateWorkScheduleInput{
		Name: "Test", StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate: time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
		Weekdays: []services.WeekdayInput{
			{Weekday: 1, StartTime: "09:00", EndTime: "12:00", RoomID: roomID},
		},
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	if err := svc.Delete(created.ID); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if _, err := svc.GetByID(created.ID); !errors.Is(err, services.ErrWorkScheduleNotFound) {
		t.Fatalf("expected not found after delete, got %v", err)
	}
	var wdCount int64
	if err := db.Model(&models.ClinicWorkScheduleWeekday{}).Where("work_schedule_id = ?", created.ID).Count(&wdCount).Error; err != nil {
		t.Fatalf("count weekdays: %v", err)
	}
	if wdCount != 0 {
		t.Errorf("expected 0 weekdays after delete, got %d", wdCount)
	}
}

func TestWorkScheduleService_Delete_NotFound(t *testing.T) {
	db := setupWorkScheduleServiceDB(t)
	svc := services.NewWorkScheduleService(db)

	err := svc.Delete(999)
	if !errors.Is(err, services.ErrWorkScheduleNotFound) {
		t.Fatalf("expected ErrWorkScheduleNotFound, got %v", err)
	}
}

func TestWorkScheduleService_AddStaff_Success(t *testing.T) {
	db := setupWorkScheduleServiceDB(t)
	svc := services.NewWorkScheduleService(db)
	roomID := seedRoom(t, db, "R1")
	staffID := seedOneStaff(t, db)
	seedStaffWorkYear(t, db, staffID, 2026)

	created, err := svc.Create(services.CreateWorkScheduleInput{
		Name: "Test", StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate: time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
		Weekdays: []services.WeekdayInput{
			{Weekday: 1, StartTime: "09:00", EndTime: "12:00", RoomID: roomID},
		},
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	full, err := svc.GetByID(created.ID)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if len(full.Weekdays) == 0 {
		t.Fatal("expected weekdays after create")
	}
	wdID := full.Weekdays[0].ID

	assign, err := svc.AddStaff(created.ID, services.StaffAssignmentInput{WeekdayID: wdID, StaffID: staffID})
	if err != nil {
		t.Fatalf("AddStaff failed: %v", err)
	}
	if assign.ID == 0 {
		t.Fatal("expected assignment ID assigned")
	}
	if assign.ScheduleID != created.ID {
		t.Errorf("expected schedule_id %d, got %d", created.ID, assign.ScheduleID)
	}
}

func TestWorkScheduleService_AddStaff_WeekdayNotFound(t *testing.T) {
	db := setupWorkScheduleServiceDB(t)
	svc := services.NewWorkScheduleService(db)
	roomID := seedRoom(t, db, "R1")
	staffID := seedOneStaff(t, db)
	seedStaffWorkYear(t, db, staffID, 2026)

	created, err := svc.Create(services.CreateWorkScheduleInput{
		Name: "Test", StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate: time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
		Weekdays: []services.WeekdayInput{
			{Weekday: 1, StartTime: "09:00", EndTime: "12:00", RoomID: roomID},
		},
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	_, err = svc.AddStaff(created.ID, services.StaffAssignmentInput{WeekdayID: 9999, StaffID: staffID})
	if !errors.Is(err, services.ErrWorkScheduleWeekdayNotFound) {
		t.Fatalf("expected ErrWorkScheduleWeekdayNotFound, got %v", err)
	}
}

func TestWorkScheduleService_RemoveStaff_Success(t *testing.T) {
	db := setupWorkScheduleServiceDB(t)
	svc := services.NewWorkScheduleService(db)
	roomID := seedRoom(t, db, "R1")
	staffID := seedOneStaff(t, db)

	created, err := svc.Create(services.CreateWorkScheduleInput{
		Name: "Test", StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate: time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
		Weekdays: []services.WeekdayInput{
			{Weekday: 1, StartTime: "09:00", EndTime: "12:00", RoomID: roomID, StaffIDs: []int{staffID}},
		},
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	full, err := svc.GetByID(created.ID)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if len(full.Weekdays) == 0 {
		t.Fatal("expected weekdays after create")
	}
	wdID := full.Weekdays[0].ID

	err = svc.RemoveStaff(created.ID, services.StaffAssignmentInput{WeekdayID: wdID, StaffID: staffID})
	if err != nil {
		t.Fatalf("RemoveStaff failed: %v", err)
	}
}

func TestWorkScheduleService_ListStaff(t *testing.T) {
	db := setupWorkScheduleServiceDB(t)
	svc := services.NewWorkScheduleService(db)
	roomID := seedRoom(t, db, "R1")
	staffIDs := seedStaff(t, db, 3)

	created, err := svc.Create(services.CreateWorkScheduleInput{
		Name: "Test", StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate: time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
		Weekdays: []services.WeekdayInput{
			{Weekday: 1, StartTime: "09:00", EndTime: "12:00", RoomID: roomID, StaffIDs: staffIDs[:2]},
			{Weekday: 3, StartTime: "13:00", EndTime: "17:00", RoomID: roomID, StaffIDs: staffIDs[2:]},
		},
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	staff, err := svc.ListStaff(created.ID)
	if err != nil {
		t.Fatalf("ListStaff failed: %v", err)
	}
	if len(staff) != 3 {
		t.Fatalf("expected 3 staff, got %d", len(staff))
	}
	for _, s := range staff {
		if s.ScheduleID != created.ID {
			t.Errorf("expected schedule_id %d on staff %d, got %d", created.ID, s.ID, s.ScheduleID)
		}
	}
}

func TestWorkScheduleService_StaffInMultipleSchedules(t *testing.T) {
	db := setupWorkScheduleServiceDB(t)
	svc := services.NewWorkScheduleService(db)
	roomID := seedRoom(t, db, "R1")
	staffID := seedOneStaff(t, db)

	schedule1, err := svc.Create(services.CreateWorkScheduleInput{
		Name:      "ScheduleA",
		StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
		Weekdays: []services.WeekdayInput{
			{Weekday: 1, StartTime: "09:00", EndTime: "12:00", RoomID: roomID, StaffIDs: []int{staffID}},
		},
	})
	if err != nil {
		t.Fatalf("create schedule1: %v", err)
	}

	schedule2, err := svc.Create(services.CreateWorkScheduleInput{
		Name:      "ScheduleB",
		StartDate: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
		Weekdays: []services.WeekdayInput{
			{Weekday: 3, StartTime: "13:00", EndTime: "17:00", RoomID: roomID, StaffIDs: []int{staffID}},
		},
	})
	if err != nil {
		t.Fatalf("create schedule2: %v", err)
	}

	staff1, err := svc.ListStaff(schedule1.ID)
	if err != nil {
		t.Fatalf("ListStaff schedule1: %v", err)
	}
	staff2, err := svc.ListStaff(schedule2.ID)
	if err != nil {
		t.Fatalf("ListStaff schedule2: %v", err)
	}
	if len(staff1) != 1 {
		t.Fatalf("expected 1 staff on schedule1, got %d", len(staff1))
	}
	if len(staff2) != 1 {
		t.Fatalf("expected 1 staff on schedule2, got %d", len(staff2))
	}
	if staff1[0].StaffID != staffID {
		t.Errorf("expected staff_id %d on schedule1, got %d", staffID, staff1[0].StaffID)
	}
	if staff2[0].StaffID != staffID {
		t.Errorf("expected staff_id %d on schedule2, got %d", staffID, staff2[0].StaffID)
	}
	if staff1[0].ScheduleID != schedule1.ID {
		t.Errorf("expected schedule_id %d, got %d", schedule1.ID, staff1[0].ScheduleID)
	}
	if staff2[0].ScheduleID != schedule2.ID {
		t.Errorf("expected schedule_id %d, got %d", schedule2.ID, staff2[0].ScheduleID)
	}
}

func TestWorkScheduleService_AddStaff_AutoCreateWeekday(t *testing.T) {
	db := setupWorkScheduleServiceDB(t)
	svc := services.NewWorkScheduleService(db)
	roomID := seedRoom(t, db, "R1")
	staffID := seedOneStaff(t, db)
	seedStaffWorkYear(t, db, staffID, 2026)

	created, err := svc.Create(services.CreateWorkScheduleInput{
		Name: "Test", StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate: time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
		// No weekdays — the auto-create path will make one.
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	assign, err := svc.AddStaff(created.ID, services.StaffAssignmentInput{
		RoomID:  roomID,
		Weekday: 1,
		StaffID: staffID,
	})
	if err != nil {
		t.Fatalf("AddStaff auto-create failed: %v", err)
	}
	if assign.ID == 0 {
		t.Fatal("expected assignment ID assigned")
	}
	if assign.ScheduleID != created.ID {
		t.Errorf("expected schedule_id %d, got %d", created.ID, assign.ScheduleID)
	}

	// Verify the weekday was created with default times.
	full, err := svc.GetByID(created.ID)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if len(full.Weekdays) != 1 {
		t.Fatalf("expected 1 weekday, got %d", len(full.Weekdays))
	}
	wd := full.Weekdays[0]
	if wd.Weekday != 1 {
		t.Errorf("expected weekday 1, got %d", wd.Weekday)
	}
	if wd.RoomID != roomID {
		t.Errorf("expected room_id %d, got %d", roomID, wd.RoomID)
	}
	expectedStart := "18:30:00"
	expectedEnd := "21:00:00"
	if wd.StartTime.Format("15:04:05") != expectedStart {
		t.Errorf("expected start_time %s, got %s", expectedStart, wd.StartTime.Format("15:04:05"))
	}
	if wd.EndTime.Format("15:04:05") != expectedEnd {
		t.Errorf("expected end_time %s, got %s", expectedEnd, wd.EndTime.Format("15:04:05"))
	}
}

func TestWorkScheduleService_AddStaff_StaffNotInWorkYear(t *testing.T) {
	db := setupWorkScheduleServiceDB(t)
	svc := services.NewWorkScheduleService(db)
	roomID := seedRoom(t, db, "R1")
	staffID := seedOneStaff(t, db)
	// Give staff a work year that does NOT match the schedule's 2026.
	seedStaffWorkYear(t, db, staffID, 2025)

	created, err := svc.Create(services.CreateWorkScheduleInput{
		Name: "Test", StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate: time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
		Weekdays: []services.WeekdayInput{
			{Weekday: 1, StartTime: "09:00", EndTime: "12:00", RoomID: roomID},
		},
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	full, err := svc.GetByID(created.ID)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	wdID := full.Weekdays[0].ID

	_, err = svc.AddStaff(created.ID, services.StaffAssignmentInput{WeekdayID: wdID, StaffID: staffID})
	if !errors.Is(err, services.ErrWorkScheduleStaffNotInWorkYear) {
		t.Fatalf("expected ErrWorkScheduleStaffNotInWorkYear, got %v", err)
	}
}

func TestWorkScheduleService_ListValidStaff(t *testing.T) {
	db := setupWorkScheduleServiceDB(t)
	svc := services.NewWorkScheduleService(db)
	staffIDs := seedStaff(t, db, 3)
	for _, sid := range staffIDs {
		seedStaffWorkYear(t, db, sid, 2026)
	}
	// Give only staff 2 a 2025 year as extra.
	seedStaffWorkYear(t, db, staffIDs[1], 2025)

	created, err := svc.Create(services.CreateWorkScheduleInput{
		Name: "Test", StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate: time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	items, err := svc.ListValidStaff(created.ID)
	if err != nil {
		t.Fatalf("ListValidStaff failed: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 staff valid for 2026, got %d", len(items))
	}
}
