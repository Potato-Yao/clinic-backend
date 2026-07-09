package main

import (
	"log"

	"clinic-backend/handlers"
	"clinic-backend/models"
	"clinic-backend/services"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	db, err := gorm.Open(sqlite.Open("clinic.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	if err := db.AutoMigrate(&models.ClinicAnnouncement{}); err != nil {
		log.Fatalf("failed to migrate: %v", err)
	}

	announcementSvc := services.NewAnnouncementService(db)
	announcementH := handlers.NewAnnouncementHandler(announcementSvc)

	r := gin.Default()

	// Staff routes (CAS auth middleware to be added here).
	admin := r.Group("/api/admin/announcements")
	{
		admin.POST("", announcementH.Create)
		admin.GET("", announcementH.List)
		admin.GET("/:id", announcementH.Get)
		admin.PUT("/:id", announcementH.Update)
		admin.DELETE("/:id", announcementH.Delete)
	}

	// Client routes (API-key signature middleware to be added here).
	client := r.Group("/api/announcements")
	{
		client.GET("", announcementH.List) // clients pass ?active=true
		client.GET("/:id", announcementH.Get)
	}

	if err := r.Run(":8080"); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
