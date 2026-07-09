package services

import (
	"errors"
	"fmt"
	"time"

	"clinic-backend/models"

	"gorm.io/gorm"
)

// ErrAnnouncementNotFound is returned when no announcement matches the given id.
var ErrAnnouncementNotFound = errors.New("announcement not found")

// AnnouncementService contains the business logic for announcement CRUD.
type AnnouncementService struct {
	db *gorm.DB
}

func NewAnnouncementService(db *gorm.DB) *AnnouncementService {
	return &AnnouncementService{db: db}
}

// CreateAnnouncementInput carries the fields a caller may set on creation.
// Server-controlled fields (ID, CreatedTime, LastEditedTime) are not included.
type CreateAnnouncementInput struct {
	Title      string
	Content    string
	Tag        string
	Brief      string
	ExpireDate time.Time
	Priority   uint
}

// UpdateAnnouncementInput uses pointers so omitted fields stay unchanged.
type UpdateAnnouncementInput struct {
	Title      *string
	Content    *string
	Tag        *string
	Brief      *string
	ExpireDate *time.Time
	Priority   *uint
}

// ListAnnouncementFilter controls listing behavior.
type ListAnnouncementFilter struct {
	Tag        string
	ActiveOnly bool
	Page       int
	PageSize   int
}

func (s *AnnouncementService) Create(in CreateAnnouncementInput) (models.ClinicAnnouncement, error) {
	now := time.Now().UTC()
	a := models.ClinicAnnouncement{
		Title:          in.Title,
		Content:        in.Content,
		Tag:            in.Tag,
		Brief:          in.Brief,
		ExpireDate:     in.ExpireDate,
		Priority:       in.Priority,
		CreatedTime:    now,
		LastEditedTime: now,
	}
	if err := s.db.Create(&a).Error; err != nil {
		return models.ClinicAnnouncement{}, fmt.Errorf("create announcement: %w", err)
	}
	return a, nil
}

func (s *AnnouncementService) GetByID(id uint) (models.ClinicAnnouncement, error) {
	var a models.ClinicAnnouncement
	if err := s.db.First(&a, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.ClinicAnnouncement{}, ErrAnnouncementNotFound
		}
		return models.ClinicAnnouncement{}, fmt.Errorf("get announcement %d: %w", id, err)
	}
	return a, nil
}

func (s *AnnouncementService) List(f ListAnnouncementFilter) ([]models.ClinicAnnouncement, int64, error) {
	q := s.db.Model(&models.ClinicAnnouncement{})
	if f.Tag != "" {
		q = q.Where("tag = ?", f.Tag)
	}
	if f.ActiveOnly {
		q = q.Where("expireDate >= ?", time.Now().UTC().Truncate(24*time.Hour))
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count announcements: %w", err)
	}

	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 {
		f.PageSize = 20
	}
	offset := (f.Page - 1) * f.PageSize

	var items []models.ClinicAnnouncement
	if err := q.
		Order("priority DESC, createdTime DESC").
		Offset(offset).
		Limit(f.PageSize).
		Find(&items).Error; err != nil {
		return nil, 0, fmt.Errorf("list announcements: %w", err)
	}
	return items, total, nil
}

func (s *AnnouncementService) Update(id uint, in UpdateAnnouncementInput) (models.ClinicAnnouncement, error) {
	var a models.ClinicAnnouncement
	if err := s.db.First(&a, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.ClinicAnnouncement{}, ErrAnnouncementNotFound
		}
		return models.ClinicAnnouncement{}, fmt.Errorf("get announcement %d for update: %w", id, err)
	}

	updates := map[string]any{}
	if in.Title != nil {
		updates["title"] = *in.Title
	}
	if in.Content != nil {
		updates["content"] = *in.Content
	}
	if in.Tag != nil {
		updates["tag"] = *in.Tag
	}
	if in.Brief != nil {
		updates["brief"] = *in.Brief
	}
	if in.ExpireDate != nil {
		updates["expireDate"] = *in.ExpireDate
	}
	if in.Priority != nil {
		updates["priority"] = *in.Priority
	}
	updates["lastEditedTime"] = time.Now().UTC()

	if len(updates) > 1 { // lastEditedTime is always present
		if err := s.db.Model(&a).Updates(updates).Error; err != nil {
			return models.ClinicAnnouncement{}, fmt.Errorf("update announcement %d: %w", id, err)
		}
	}

	// Reload to return the updated row.
	if err := s.db.First(&a, id).Error; err != nil {
		return models.ClinicAnnouncement{}, fmt.Errorf("reload announcement %d: %w", id, err)
	}
	return a, nil
}

func (s *AnnouncementService) Delete(id uint) error {
	res := s.db.Delete(&models.ClinicAnnouncement{}, id)
	if err := res.Error; err != nil {
		return fmt.Errorf("delete announcement %d: %w", id, err)
	}
	if res.RowsAffected == 0 {
		return ErrAnnouncementNotFound
	}
	return nil
}
