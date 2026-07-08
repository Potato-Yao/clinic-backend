package models

// ClinicRoom represents a physical clinic location where repair appointments are held.
type ClinicRoom struct {
	ID      uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	Name    string `gorm:"not null" json:"name"`
	Address string `gorm:"not null" json:"address"`
}
