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
	if err := db.AutoMigrate(&models.ClinicAnnouncement{}, &models.ClinicServiceDate{}, &models.ClinicRoom{}); err != nil {
		log.Fatalf("failed to migrate: %v", err)
	}

	announcementSvc := services.NewAnnouncementService(db)
	announcementH := handlers.NewAnnouncementHandler(announcementSvc)

	serviceDateSvc := services.NewServiceDateService(db)
	serviceDateH := handlers.NewServiceDateHandler(serviceDateSvc)

	roomSvc := services.NewRoomService(db)
	roomH := handlers.NewRoomHandler(roomSvc)

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

	serviceDateAdmin := r.Group("/api/admin/service-dates")
	{
		serviceDateAdmin.POST("", serviceDateH.Create)
		serviceDateAdmin.GET("", serviceDateH.List)
		serviceDateAdmin.GET("/:id", serviceDateH.Get)
		serviceDateAdmin.PUT("/:id", serviceDateH.Update)
		serviceDateAdmin.DELETE("/:id", serviceDateH.Delete)
	}

	// Client routes (API-key signature middleware to be added here).
	client := r.Group("/api/announcements")
	{
		client.GET("", announcementH.List) // clients pass ?active=true
		client.GET("/:id", announcementH.Get)
	}
	serviceDateClient := r.Group("/api/service-dates")
	{
		serviceDateClient.GET("", serviceDateH.List) // clients pass ?active=true&available=true
		serviceDateClient.GET("/:id", serviceDateH.Get)
	}

	roomAdmin := r.Group("/api/admin/rooms")
	{
		roomAdmin.POST("", roomH.Create)
		roomAdmin.GET("", roomH.List)
		roomAdmin.GET("/:id", roomH.Get)
		roomAdmin.PUT("/:id", roomH.Update)
		roomAdmin.DELETE("/:id", roomH.Delete)
	}

	// Client routes (API-key signature middleware to be added here).
	roomClient := r.Group("/api/rooms")
	{
		roomClient.GET("", roomH.List) // clients see enabled rooms only via ?enabled=true
		roomClient.GET("/:id", roomH.Get)
	}

	if err := r.Run(":8080"); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
