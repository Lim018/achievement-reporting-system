package routes

import (
	"database/sql"

	"github.com/gofiber/fiber/v2"
	"go-fiber/app/service"
	"go-fiber/middleware"
	"go.mongodb.org/mongo-driver/mongo"
)

func ReportsRoutes(app *fiber.App, db *sql.DB, mongoDB *mongo.Database) {
	svc := service.NewReportsService(db, mongoDB)
	
	reports := app.Group("/api/v1/reports", middleware.AuthRequired())

	// GET /api/v1/reports/statistics
	// Accessible by: Admin (full stats), Dosen Wali (advisee stats), Mahasiswa (own stats)
	reports.Get("/statistics", 
		middleware.RequirePermission("report:system"), 
		svc.GetSystemStatistics,
	)

	// GET /api/v1/reports/student/:id
	// Accessible by: Admin (any student), Dosen Wali (advisee only), Mahasiswa (self only)
	reports.Get("/student/:id", 
		middleware.RequirePermission("achievement:read"), 
		svc.GetStudentReport,
	)
}