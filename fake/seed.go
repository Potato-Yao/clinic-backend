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

	if err := db.AutoMigrate(
		&models.ClinicAnnouncement{},
		&models.ClinicServiceDate{},
		&models.ClinicRoom{},
		&models.ClinicStaff{},
		&models.ClinicWorkSchedule{},
		&models.ClinicWorkScheduleWeekday{},
		&models.ClinicWorkScheduleStaff{},
	); err != nil {
		log.Fatalf("failed to migrate: %v", err)
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

	// ── Staff records ────────────────────────────────────────────────────
	staffList := []models.ClinicStaff{
		{AccountID: "manualuser", Realname: "Manual Tester", PhoneNum: "13800138000"},
		{AccountID: "zhangsan", Realname: "张三", PhoneNum: "13900139001"},
		{AccountID: "lisi", Realname: "李四", PhoneNum: "13900139002"},
		{AccountID: "wangwu", Realname: "王五", PhoneNum: "13900139003"},
	}
	for _, s := range staffList {
		if err := db.Where("account_id = ?", s.AccountID).FirstOrCreate(&s).Error; err != nil {
			log.Fatalf("seed staff %s: %v", s.AccountID, err)
		}
		fmt.Printf("staff: %s (id=%d)\n", s.AccountID, s.ID)
	}

	// ── Work schedule ─────────────────────────────────────────────────────
	weekFromNow := today.AddDate(0, 0, 7)
	twoWeeksFromNow := today.AddDate(0, 0, 14)

	schedule := models.ClinicWorkSchedule{
		Name:      "默认排班",
		StartDate: weekFromNow,
		EndDate:   twoWeeksFromNow,
	}
	if err := db.Where("name = ? AND start_date = ?", schedule.Name, schedule.StartDate).FirstOrCreate(&schedule).Error; err != nil {
		log.Fatalf("seed work schedule: %v", err)
	}
	fmt.Printf("work-schedule: %s (id=%d, %s ~ %s)\n", schedule.Name, schedule.ID, schedule.StartDate.Format("2006-01-02"), schedule.EndDate.Format("2006-01-02"))

	// ── Schedule weekdays ────────────────────────────────────────────────
	type weekdayDef struct {
		Weekday   int
		StartHour int
		StartMin  int
		EndHour   int
		EndMin    int
		Room      string
	}
	weekdayDefs := []weekdayDef{
		{Weekday: 1, StartHour: 9, StartMin: 0, EndHour: 12, EndMin: 0, Room: "中关村"},
		{Weekday: 1, StartHour: 14, StartMin: 0, EndHour: 17, EndMin: 0, Room: "沙河"},
		{Weekday: 3, StartHour: 9, StartMin: 0, EndHour: 12, EndMin: 0, Room: "中关村"},
		{Weekday: 5, StartHour: 14, StartMin: 0, EndHour: 17, EndMin: 0, Room: "沙河"},
	}

	// Build a lookup from weekday to the dummy date used for time.Time fields.
	scheduleStart := weekFromNow
	var savedWeekdays []models.ClinicWorkScheduleWeekday
	for _, wd := range weekdayDefs {
		// Use a placeholder date (the schedule start + weekday offset) for time fields.
		day := scheduleStart.AddDate(0, 0, wd.Weekday-1)
		entry := models.ClinicWorkScheduleWeekday{
			WorkScheduleID: schedule.ID,
			Weekday:        wd.Weekday,
			StartTime:      day.Add(time.Duration(wd.StartHour)*time.Hour + time.Duration(wd.StartMin)*time.Minute),
			EndTime:        day.Add(time.Duration(wd.EndHour)*time.Hour + time.Duration(wd.EndMin)*time.Minute),
		}
		var room models.ClinicRoom
		if err := db.Where("name = ?", wd.Room).First(&room).Error; err != nil {
			log.Fatalf("seed lookup room %s: %v", wd.Room, err)
		}
		entry.RoomID = room.ID

		if err := db.Where("work_schedule_id = ? AND weekday = ? AND room_id = ? AND start_time = ?",
			entry.WorkScheduleID, entry.Weekday, entry.RoomID, entry.StartTime).
			FirstOrCreate(&entry).Error; err != nil {
			log.Fatalf("seed schedule weekday: %v", err)
		}
		savedWeekdays = append(savedWeekdays, entry)
		fmt.Printf("schedule-weekday: weekday=%d room=%s (id=%d)\n", entry.Weekday, wd.Room, entry.ID)
	}

	// ── Schedule staff assignments ───────────────────────────────────────
	assignments := map[int][]string{
		1: {"zhangsan", "lisi"},
		3: {"wangwu"},
		5: {"zhangsan", "wangwu"},
	}
	for _, weekdayEntry := range savedWeekdays {
		staffAccounts, ok := assignments[weekdayEntry.Weekday]
		if !ok {
			continue
		}
		for _, account := range staffAccounts {
			var staff models.ClinicStaff
			if err := db.Where("account_id = ?", account).First(&staff).Error; err != nil {
				log.Fatalf("seed lookup staff %s: %v", account, err)
			}
			ss := models.ClinicWorkScheduleStaff{
				WeekdayID: weekdayEntry.ID,
				StaffID:   staff.ID,
			}
			if err := db.Where("weekday_id = ? AND staff_id = ?", ss.WeekdayID, ss.StaffID).FirstOrCreate(&ss).Error; err != nil {
				log.Fatalf("seed schedule staff: %v", err)
			}
			fmt.Printf("schedule-staff: weekday=%d staff=%s (id=%d)\n", weekdayEntry.Weekday, account, ss.ID)
		}
	}

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
