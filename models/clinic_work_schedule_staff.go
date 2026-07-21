package models

type ClinicWorkScheduleStaff struct {
	ID        uint                      `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	WeekdayID uint                      `gorm:"not null;column:weekday_id" json:"weekday_id"`
	StaffID   int                       `gorm:"not null;column:staff_id" json:"staff_id"`
	Weekday   ClinicWorkScheduleWeekday `gorm:"foreignKey:WeekdayID" json:"-"`
	Staff     ClinicStaff               `gorm:"foreignKey:StaffID" json:"staff,omitempty"`
}

func (ClinicWorkScheduleStaff) TableName() string {
	return "clinic_work_schedule_staff"
}
