package models

import "time"

type ClinicWorkSchedule struct {
	ID        uint      `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	Name      string    `gorm:"size:128;not null;column:name" json:"name"`
	StartDate time.Time `gorm:"type:date;not null;column:start_date" json:"start_date"`
	EndDate   time.Time `gorm:"type:date;not null;column:end_date" json:"end_date"`
}

func (ClinicWorkSchedule) TableName() string {
	return "clinic_work_schedule"
}
