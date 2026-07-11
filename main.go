package main

import (
	"log"
	"os"
	"time"

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

	ticketSvc := services.NewTicketService(db)
	ticketH := handlers.NewTicketHandler(ticketSvc)

	apiKey := os.Getenv("CLINIC_API_KEY")
	if apiKey == "" {
		log.Fatal("CLINIC_API_KEY not set")
	}
	clientAuth := handlers.ClientAuthMiddleware(apiKey, 5*time.Minute)

	r := gin.Default()
	// The DingTalk proxy sends trailing-slash paths (e.g. /api/wechat/).
	// Gin's RedirectTrailingSlash would convert POST -> GET via 301, so disable it.
	r.RedirectTrailingSlash = false

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

	// Client routes (API-key signature middleware).
	client := r.Group("/api/announcements")
	client.Use(clientAuth)
	{
		client.GET("", announcementH.List) // clients pass ?active=true
		client.GET("/:id", announcementH.Get)
	}
	serviceDateClient := r.Group("/api/service-dates")
	serviceDateClient.Use(clientAuth)
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

	// Client routes (API-key signature middleware).
	roomClient := r.Group("/api/rooms")
	roomClient.Use(clientAuth)
	{
		roomClient.GET("", roomH.List) // clients see enabled rooms only via ?enabled=true
		roomClient.GET("/:id", roomH.Get)
	}

	// Ticket routes (API-key signature middleware). Canonical paths.
	ticket := r.Group("/api/tickets")
	ticket.Use(clientAuth)
	{
		ticket.GET("", ticketH.List)
		ticket.POST("", ticketH.Create)
		ticket.GET("/working", ticketH.Working)
		ticket.GET("/finished", ticketH.Finished)
		ticket.GET("/:id", ticketH.Get)
		ticket.PUT("/:id", ticketH.Update)
		ticket.PATCH("/:id", ticketH.Update)
		ticket.DELETE("/:id", ticketH.Delete)
	}

	// Legacy /api/wechat alias — same handlers. The DingTalk proxy sends
	// trailing-slash paths, so register both forms explicitly (RedirectTrailingSlash
	// is disabled above and gin doesn't match both "/" and "" automatically).
	wechat := r.Group("/api/wechat")
	wechat.Use(clientAuth)
	{
		wechat.GET("", ticketH.List)
		wechat.GET("/", ticketH.List)
		wechat.POST("", ticketH.Create)
		wechat.POST("/", ticketH.Create)
		wechat.GET("/working", ticketH.Working)
		wechat.GET("/working/", ticketH.Working)
		wechat.GET("/finished", ticketH.Finished)
		wechat.GET("/finished/", ticketH.Finished)
		wechat.GET("/:id", ticketH.Get)
		wechat.GET("/:id/", ticketH.Get)
		wechat.PUT("/:id", ticketH.Update)
		wechat.PUT("/:id/", ticketH.Update)
		wechat.PATCH("/:id", ticketH.Update)
		wechat.PATCH("/:id/", ticketH.Update)
		wechat.DELETE("/:id", ticketH.Delete)
		wechat.DELETE("/:id/", ticketH.Delete)
	}

	if err := r.Run(":8080"); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
