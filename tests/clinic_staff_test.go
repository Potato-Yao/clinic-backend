package tests

import (
	"testing"

	"clinic-backend/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupStaffTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:?_fk=1"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	if err := db.AutoMigrate(&models.ClinicStaff{}, &models.ClinicStaffWorkyear{}); err != nil {
		t.Fatalf("failed to migrate staff models: %v", err)
	}

	return db
}

func TestClinicStaff_CreateAndRetrieve(t *testing.T) {
	db := setupStaffTestDB(t)

	staff := models.ClinicStaff{
		ID:        42,
		AccountID: "cas:student42",
		Realname:  "Alice Smith",
		PhoneNum:  "123-456-7890",
	}

	if err := db.Create(&staff).Error; err != nil {
		t.Fatalf("failed to create staff: %v", err)
	}

	var retrieved models.ClinicStaff
	if err := db.First(&retrieved, staff.ID).Error; err != nil {
		t.Fatalf("failed to retrieve staff: %v", err)
	}

	if retrieved.AccountID != staff.AccountID {
		t.Errorf("expected account_id %q, got %q", staff.AccountID, retrieved.AccountID)
	}
	if retrieved.Realname != staff.Realname {
		t.Errorf("expected realname %q, got %q", staff.Realname, retrieved.Realname)
	}
	if retrieved.PhoneNum != staff.PhoneNum {
		t.Errorf("expected phone_num %q, got %q", staff.PhoneNum, retrieved.PhoneNum)
	}
}

func TestClinicStaff_AccountIDUnique(t *testing.T) {
	db := setupStaffTestDB(t)

	if err := db.Create(&models.ClinicStaff{ID: 1, AccountID: "duplicate"}).Error; err != nil {
		t.Fatalf("failed to create first staff: %v", err)
	}

	if err := db.Create(&models.ClinicStaff{ID: 2, AccountID: "duplicate"}).Error; err == nil {
		t.Fatal("expected duplicate account_id to fail")
	}
}

func TestClinicStaffWorkyear_CreateAndRetrieve(t *testing.T) {
	db := setupStaffTestDB(t)

	staff := models.ClinicStaff{ID: 7, AccountID: "cas:staff7"}
	if err := db.Create(&staff).Error; err != nil {
		t.Fatalf("failed to create staff: %v", err)
	}

	workyear := models.ClinicStaffWorkyear{StaffID: staff.ID, WorkYear: 2026}
	if err := db.Create(&workyear).Error; err != nil {
		t.Fatalf("failed to create workyear: %v", err)
	}

	var retrieved models.ClinicStaffWorkyear
	if err := db.First(&retrieved, []interface{}{workyear.StaffID, workyear.WorkYear}).Error; err != nil {
		t.Fatalf("failed to retrieve workyear: %v", err)
	}

	if retrieved.StaffID != workyear.StaffID {
		t.Errorf("expected staff_id %d, got %d", workyear.StaffID, retrieved.StaffID)
	}
	if retrieved.WorkYear != workyear.WorkYear {
		t.Errorf("expected work_year %d, got %d", workyear.WorkYear, retrieved.WorkYear)
	}
}

func TestClinicStaffWorkyear_CompositeKeyPreventsDuplicate(t *testing.T) {
	db := setupStaffTestDB(t)

	staff := models.ClinicStaff{ID: 8, AccountID: "cas:staff8"}
	if err := db.Create(&staff).Error; err != nil {
		t.Fatalf("failed to create staff: %v", err)
	}

	workyear := models.ClinicStaffWorkyear{StaffID: staff.ID, WorkYear: 2025}
	if err := db.Create(&workyear).Error; err != nil {
		t.Fatalf("failed to create first workyear: %v", err)
	}

	if err := db.Create(&models.ClinicStaffWorkyear{StaffID: staff.ID, WorkYear: 2025}).Error; err == nil {
		t.Fatal("expected duplicate composite key to fail")
	}
}
