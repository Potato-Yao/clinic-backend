package models

// ClinicStaff represents a clinic staff member who logs in via the university CAS system.
type ClinicStaff struct {
	ID        int    `gorm:"primaryKey;not null;column:id" json:"id"`
	AccountID string `gorm:"type:text;not null;unique;column:account_id" json:"account_id"`
	Realname  string `gorm:"size:50;column:realname" json:"realname"`
	PhoneNum  string `gorm:"size:20;column:phone_num" json:"phone_num"`
}

// TableName overrides GORM's default pluralized table name.
func (ClinicStaff) TableName() string {
	return "clinic_staff"
}
