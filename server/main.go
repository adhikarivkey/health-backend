package main

import (
	"log"
	"time"

	"health-backend/external"
	"health-backend/agnoshealth/config"
	"health-backend/agnoshealth/handler"
	"health-backend/agnoshealth/middleware"
	"health-backend/agnoshealth/model"

	"github.com/gin-gonic/gin"
)

func main() {

	// Load config
	config.Load()
	cfg := config.AppConfig

	// Set Gin mode
	if cfg.GinMode != "" {
		gin.SetMode(cfg.GinMode)
	}

	// Connect DB
	db, err := config.Connect(cfg)
	if err != nil {
		log.Fatalf("❌ Failed to connect to DB: %v", err)
	}

	// Auto migrate
	if err := db.AutoMigrate(
		&model.Hospital{},
		&model.Staff{},
		&model.Patient{},
	); err != nil {
		log.Fatalf("❌ Migration failed: %v", err)
	}

	log.Println("✅ Database connected and migrated successfully")

	// External HIS client
	hisClient := external.NewHISClient(cfg.HISBaseURL)

	// Handlers
	staffHandler := handler.NewStaffHandler(db, cfg.JWTSecret)
	patientHandler := handler.NewPatientHandler(db, hisClient)

	// Router
	r := gin.Default()

	r.SetTrustedProxies(nil)

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "timestamp": time.Now()})
	})

	// Public routes
	r.POST("/staff/create", staffHandler.Create)
	r.POST("/staff/login", staffHandler.Login)

	// Protected routes
	protected := r.Group("/")
	protected.Use(middleware.Auth(cfg.JWTSecret))
	{
		protected.GET("/patient/search", patientHandler.Search)
		protected.POST("/patient/search", patientHandler.Search)
	}

	// Start server
	port := cfg.Port
	if port == "" {
		port = "8000"
	}

	log.Printf("🚀 Server running on port %s", port)

	if err := r.Run(":" + port); err != nil {
		log.Fatalf("❌ Server failed: %v", err)
	}
}
