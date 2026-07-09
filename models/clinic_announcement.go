package models

import "time"

// ClinicAnnouncement represents a staff-published announcement or terms of service
// with an expiration date and a display priority.
type ClinicAnnouncement struct {
	ID             uint      `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	Title          string    `gorm:"size:20;not null;column:title" json:"title"`
	Content        string    `gorm:"type:text;not null;column:content" json:"content"`
	Tag            string    `gorm:"size:16;not null;column:tag" json:"tag"`
	CreatedTime    time.Time `gorm:"not null;column:createdTime" json:"createdTime"`
	LastEditedTime time.Time `gorm:"not null;column:lastEditedTime" json:"lastEditedTime"`
	ExpireDate     time.Time `gorm:"type:date;not null;column:expireDate" json:"expireDate"`
	Priority       uint      `gorm:"not null;column:priority" json:"priority"`
	Brief          string    `gorm:"size:64;not null;column:brief" json:"brief"`
}

// TableName overrides GORM's default pluralized table name.
func (ClinicAnnouncement) TableName() string {
	return "clinic_announcement"
}
