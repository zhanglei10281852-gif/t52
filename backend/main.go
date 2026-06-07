package main

import (
	"log"
	"scenic-ticket/config"
	"scenic-ticket/database"
	"scenic-ticket/handlers"
	"scenic-ticket/middleware"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	if err := database.Init(cfg); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"*"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	api := r.Group("/api")

	api.POST("/auth/login", handlers.Login)

	auth := api.Group("")
	auth.Use(middleware.AuthMiddleware())
	{
		auth.GET("/auth/me", handlers.GetCurrentUser)

		auth.GET("/ticket-types", handlers.GetTicketTypes)
		auth.GET("/gates", handlers.GetGates)

		auth.POST("/tickets/sell", middleware.SellerOrAdminMiddleware(), handlers.SellTicket)
		auth.POST("/tickets/refund", middleware.SellerOrAdminMiddleware(), handlers.RefundTicket)
		auth.GET("/tickets/:ticket_no", handlers.GetTicket)
		auth.GET("/tickets", handlers.SearchTickets)

		auth.POST("/check-in", handlers.CheckIn)
		auth.POST("/check-out", handlers.CheckOut)
		auth.GET("/in-park-count", handlers.GetInParkCount)
		auth.GET("/check-records", handlers.GetCheckRecords)

		auth.GET("/dashboard", handlers.GetDashboard)
		auth.GET("/stats/gates", handlers.GetGateStats)
		auth.GET("/stats/hourly", handlers.GetHourlyStats)
		auth.GET("/stats/daily", handlers.GetDailyStats)
		auth.GET("/stats/ticket-types", handlers.GetTicketTypeStats)
		auth.GET("/stats/slot-heatmap", handlers.GetSlotHeatmap)

		admin := auth.Group("")
		admin.Use(middleware.AdminMiddleware())
		{
			admin.POST("/ticket-types", handlers.CreateTicketType)
			admin.GET("/users", handlers.GetUsers)
		}
	}

	log.Printf("Server starting on port %s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
