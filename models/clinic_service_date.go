package models

import "time"

// ClinicServiceDate represents a single clinic service window on a specific date and room.
type ClinicServiceDate struct {
	ID        uint      `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	Capacity  uint      `gorm:"not null;column:capacity;check:,capacity >= 0" json:"capacity"`
	RoomID    *uint     `gorm:"column:room_id;uniqueIndex:idx_room_date" json:"room_id"`
	Date      time.Time `gorm:"type:date;not null;column:date;uniqueIndex:idx_room_date" json:"date"`
	StartTime time.Time `gorm:"type:datetime;not null;column:startTime" json:"startTime"`
	EndTime   time.Time `gorm:"type:datetime;not null;column:endTime" json:"endTime"`
	Title     string    `gorm:"size:20;not null;column:title" json:"title"`
}

// TableName overrides GORM's default pluralized table name.
func (ClinicServiceDate) TableName() string {
	return "clinic_service_date"
}
