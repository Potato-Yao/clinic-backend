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
	if err := db.AutoMigrate(
		&models.ClinicAnnouncement{},
		&models.ClinicServiceDate{},
		&models.ClinicRoom{},
		&models.ClinicStaff{},
		&models.ClinicStaffWorkyear{},
		&models.ClinicRecord{},
		&models.ClinicRecordDevice{},
		&models.ClinicRecordWorker{},
		&models.ClinicRecordArrival{},
		&models.ClinicRecordRejection{},
		&models.ClinicRecordReferral{},
	); err != nil {
		log.Fatalf("failed to migrate: %v", err)
	}
	if err := db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_unique_tos
		ON clinic_announcement(tag)
		WHERE tag = 'tos'
	`).Error; err != nil {
		log.Fatalf("failed to create tos unique index: %v", err)
	}

	announcementSvc := services.NewAnnouncementService(db)
	announcementH := handlers.NewAnnouncementHandler(announcementSvc)

	serviceDateSvc := services.NewServiceDateService(db)
	serviceDateH := handlers.NewServiceDateHandler(serviceDateSvc)

	roomSvc := services.NewRoomService(db)
	roomH := handlers.NewRoomHandler(roomSvc)

	ticketSvc := services.NewTicketService(db)
	ticketH := handlers.NewTicketHandler(ticketSvc)

	staffSvc := services.NewStaffService(db)
	staffH := handlers.NewStaffHandler(staffSvc)

	adminRecordSvc := services.NewAdminRecordService(db)
	adminRecordH := handlers.NewAdminRecordHandler(adminRecordSvc)

	apiKey := os.Getenv("CLINIC_API_KEY")
	if apiKey == "" {
		log.Fatal("CLINIC_API_KEY not set")
	}
	clientAuth := handlers.ClientAuthMiddleware(apiKey, 5*time.Minute)

	keycloakRealm := os.Getenv("KEYCLOAK_REALM_URL")
	keycloakClient := os.Getenv("KEYCLOAK_CLIENT_ID")
	kcAuth := handlers.NewKeycloakAuthMiddleware(keycloakRealm, keycloakClient, staffSvc)

	r := gin.Default()
	r.RedirectTrailingSlash = false

	// ── Admin: Announcements ──────────────────────────────────────────────
	// Staff and admin can read; admin only can write.
	annRead := r.Group("/api/admin/announcements")
	annRead.Use(kcAuth, handlers.RequireStaff)
	{
		annRead.GET("", announcementH.List)
		annRead.GET("/:id", announcementH.Get)
	}
	annWrite := r.Group("/api/admin/announcements")
	annWrite.Use(kcAuth, handlers.RequireAdmin)
	{
		annWrite.POST("", announcementH.Create)
		annWrite.PUT("/:id", announcementH.Update)
		annWrite.DELETE("/:id", announcementH.Delete)
	}

	// ── Admin: Service Dates ──────────────────────────────────────────────
	sdRead := r.Group("/api/admin/service-dates")
	sdRead.Use(kcAuth, handlers.RequireStaff)
	{
		sdRead.GET("", serviceDateH.List)
		sdRead.GET("/:id", serviceDateH.Get)
	}
	sdWrite := r.Group("/api/admin/service-dates")
	sdWrite.Use(kcAuth, handlers.RequireAdmin)
	{
		sdWrite.POST("", serviceDateH.Create)
		sdWrite.PUT("/:id", serviceDateH.Update)
		sdWrite.DELETE("/:id", serviceDateH.Delete)
	}

	// ── Admin: Rooms ──────────────────────────────────────────────────────
	roomRead := r.Group("/api/admin/rooms")
	roomRead.Use(kcAuth, handlers.RequireStaff)
	{
		roomRead.GET("", roomH.List)
		roomRead.GET("/:id", roomH.Get)
	}
	roomWrite := r.Group("/api/admin/rooms")
	roomWrite.Use(kcAuth, handlers.RequireAdmin)
	{
		roomWrite.POST("", roomH.Create)
		roomWrite.PUT("/:id", roomH.Update)
		roomWrite.DELETE("/:id", roomH.Delete)
	}

	// ── Admin: Records (staff + admin) ────────────────────────────────────
	records := r.Group("/api/admin/records")
	records.Use(kcAuth, handlers.RequireStaff)
	{
		records.GET("", adminRecordH.List)
		records.GET("/:id", adminRecordH.Get)
		records.PUT("/:id", adminRecordH.Update)
		records.POST("/:id/arrive", adminRecordH.Arrive)
		records.POST("/:id/in-progress", adminRecordH.InProgress)
		records.POST("/:id/complete", adminRecordH.Complete)
		records.POST("/:id/reject", adminRecordH.Reject)
	}

	// ── Admin: Staff Management (admin only) ──────────────────────────────
	staffAdm := r.Group("/api/admin/staff")
	staffAdm.Use(kcAuth, handlers.RequireAdmin)
	{
		staffAdm.GET("", staffH.List)
		staffAdm.GET("/:id", staffH.Get)
		staffAdm.POST("", staffH.Create)
		staffAdm.PUT("/:id", staffH.Update)
		staffAdm.DELETE("/:id", staffH.Delete)
	}

	// ── Client routes (API-key signature middleware). ─────────────────────
	client := r.Group("/api/announcements")
	client.Use(clientAuth)
	{
		client.GET("", announcementH.List)
		client.GET("/:id", announcementH.Get)
	}
	serviceDateClient := r.Group("/api/service-dates")
	serviceDateClient.Use(clientAuth)
	{
		serviceDateClient.GET("", serviceDateH.List)
		serviceDateClient.GET("/:id", serviceDateH.Get)
	}
	roomClient := r.Group("/api/rooms")
	roomClient.Use(clientAuth)
	{
		roomClient.GET("", roomH.List)
		roomClient.GET("/:id", roomH.Get)
	}

	// ── Ticket routes (client) ────────────────────────────────────────────
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

	// Legacy /api/wechat alias.
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
