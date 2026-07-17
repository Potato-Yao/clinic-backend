package main

import (
	"fmt"
	"log"
	"time"

	"clinic-backend/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	db, err := gorm.Open(sqlite.Open("clinic.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}

	now := time.Now().UTC()
	today := now.Truncate(24 * time.Hour)

	// ── Rooms ──────────────────────────────────────────────────────────────
	rooms := []models.ClinicRoom{
		{Name: "中关村", Address: "中关村校区", Enabled: true},
		{Name: "沙河", Address: "沙河校区", Enabled: true},
	}
	for _, r := range rooms {
		if err := db.Where("name = ?", r.Name).FirstOrCreate(&r).Error; err != nil {
			log.Fatalf("seed room %s: %v", r.Name, err)
		}
		fmt.Printf("room: %s (id=%d)\n", r.Name, r.ID)
	}

	// Fetch the room IDs we just created.
	var roomZG, roomSH models.ClinicRoom
	db.Where("name = ?", "中关村").First(&roomZG)
	db.Where("name = ?", "沙河").First(&roomSH)

	// ── Service dates (tomorrow and day after) ─────────────────────────────
	tomorrow := today.AddDate(0, 0, 1)
	dayAfter := today.AddDate(0, 0, 2)

	dates := []models.ClinicServiceDate{
		{
			Capacity:  15,
			RoomID:    &roomZG.ID,
			Date:      tomorrow,
			StartTime: tomorrow.Add(18*time.Hour + 30*time.Minute),
			EndTime:   tomorrow.Add(21 * time.Hour),
			Title:     "正常服务",
		},
		{
			Capacity:  10,
			RoomID:    &roomSH.ID,
			Date:      dayAfter,
			StartTime: dayAfter.Add(18*time.Hour + 30*time.Minute),
			EndTime:   dayAfter.Add(21 * time.Hour),
			Title:     "正常服务",
		},
	}
	for _, d := range dates {
		if err := db.Where("room_id = ? AND date = ?", d.RoomID, d.Date).FirstOrCreate(&d).Error; err != nil {
			log.Fatalf("seed service date: %v", err)
		}
		fmt.Printf("service-date: %s room=%d (id=%d)\n", d.Date.Format("2006-01-02"), *d.RoomID, d.ID)
	}

	// ── Staff record for the fake CAS user ────────────────────────────────
	staff := models.ClinicStaff{
		AccountID: "manualuser",
		Realname:  "Manual Tester",
		PhoneNum:  "13800138000",
	}
	if err := db.Where("account_id = ?", staff.AccountID).FirstOrCreate(&staff).Error; err != nil {
		log.Fatalf("seed staff: %v", err)
	}
	fmt.Printf("staff: %s (id=%d)\n", staff.AccountID, staff.ID)

	// ── Announcement ──────────────────────────────────────────────────────
	ann := models.ClinicAnnouncement{
		Title:          "欢迎使用诊所管理系统",
		Content:        "这是诊所管理系统的测试实例。\n\n请查看公告、服务时间和管理工单。",
		Tag:            models.AnnouncementTagPinned,
		CreatedTime:    now,
		LastEditedTime: now,
		ExpireDate:     today.AddDate(0, 0, 30),
		Priority:       1,
		Brief:          "欢迎使用！",
	}
	if err := db.Where("title = ?", ann.Title).FirstOrCreate(&ann).Error; err != nil {
		log.Fatalf("seed announcement: %v", err)
	}
	fmt.Printf("announcement: %s (id=%d)\n", ann.Title, ann.ID)

	fmt.Println("\nSeed complete. Run 'go run main.go' to start the backend.")
}
