package services

import (
	"errors"
	"fmt"

	"clinic-backend/models"

	"gorm.io/gorm"
)

var (
	ErrStaffNotFound = errors.New("staff not found")
	ErrStaffConflict = errors.New("staff account already exists")
)

type StaffService struct {
	db *gorm.DB
}

func NewStaffService(db *gorm.DB) *StaffService {
	return &StaffService{db: db}
}

type CreateStaffInput struct {
	AccountID string
	Realname  string
	PhoneNum  string
}

type UpdateStaffInput struct {
	Realname *string
	PhoneNum *string
}

func (s *StaffService) GetOrCreateByAccountID(accountID, realname string) (models.ClinicStaff, error) {
	var staff models.ClinicStaff
	err := s.db.Where("account_id = ?", accountID).First(&staff).Error
	if err == nil {
		if realname != "" && staff.Realname != realname {
			s.db.Model(&staff).Update("realname", realname)
		}
		return staff, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return models.ClinicStaff{}, fmt.Errorf("lookup staff %s: %w", accountID, err)
	}

	staff = models.ClinicStaff{
		AccountID: accountID,
		Realname:  realname,
	}
	if err := s.db.Create(&staff).Error; err != nil {
		return models.ClinicStaff{}, fmt.Errorf("create staff %s: %w", accountID, err)
	}
	return staff, nil
}

func (s *StaffService) GetByID(id int) (models.ClinicStaff, error) {
	var staff models.ClinicStaff
	if err := s.db.First(&staff, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.ClinicStaff{}, ErrStaffNotFound
		}
		return models.ClinicStaff{}, fmt.Errorf("get staff %d: %w", id, err)
	}
	return staff, nil
}

func (s *StaffService) List() ([]models.ClinicStaff, error) {
	var staff []models.ClinicStaff
	if err := s.db.Order("id ASC").Find(&staff).Error; err != nil {
		return nil, fmt.Errorf("list staff: %w", err)
	}
	return staff, nil
}

func (s *StaffService) Update(id int, in UpdateStaffInput) (models.ClinicStaff, error) {
	var staff models.ClinicStaff
	if err := s.db.First(&staff, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.ClinicStaff{}, ErrStaffNotFound
		}
		return models.ClinicStaff{}, fmt.Errorf("get staff %d for update: %w", id, err)
	}

	updates := map[string]any{}
	if in.Realname != nil {
		updates["realname"] = *in.Realname
	}
	if in.PhoneNum != nil {
		updates["phone_num"] = *in.PhoneNum
	}
	if len(updates) > 0 {
		if err := s.db.Model(&staff).Updates(updates).Error; err != nil {
			return models.ClinicStaff{}, fmt.Errorf("update staff %d: %w", id, err)
		}
	}

	if err := s.db.First(&staff, id).Error; err != nil {
		return models.ClinicStaff{}, fmt.Errorf("reload staff %d: %w", id, err)
	}
	return staff, nil
}

func (s *StaffService) Delete(id int) error {
	res := s.db.Delete(&models.ClinicStaff{}, id)
	if err := res.Error; err != nil {
		return fmt.Errorf("delete staff %d: %w", id, err)
	}
	if res.RowsAffected == 0 {
		return ErrStaffNotFound
	}
	return nil
}
