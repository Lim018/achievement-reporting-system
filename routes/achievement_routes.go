package routes

import (
	"database/sql"

	"github.com/gofiber/fiber/v2"
	"go-fiber/app/service"
	"go-fiber/middleware"
	"go.mongodb.org/mongo-driver/mongo"
)

func AchievementRoutes(app *fiber.App, db *sql.DB, mongoDB *mongo.Database) {
	svc := service.NewAchievementService(db, mongoDB)
	r := app.Group("/api/v1/achievements", middleware.AuthRequired())

	r.Get("/", func(c *fiber.Ctx) error { return svc.ListAchievementsService(c) })
	r.Get("/:id", func(c *fiber.Ctx) error { return svc.GetAchievementDetailService(c) })
	r.Get("/:id/history", func(c *fiber.Ctx) error { return svc.GetHistoryService(c) })

	r.Post("/", func(c *fiber.Ctx) error { return svc.CreateAchievementService(c) })
	r.Put("/:id", func(c *fiber.Ctx) error { return svc.UpdateAchievementService(c) })
	r.Delete("/:id", func(c *fiber.Ctx) error { return svc.DeleteAchievementService(c) })
	r.Post("/:id/submit", func(c *fiber.Ctx) error { return svc.SubmitAchievementService(c) })
	r.Post("/:id/attachments", func(c *fiber.Ctx) error { return svc.UploadAttachmentsService(c) })

	r.Post("/:id/verify", middleware.RequireRole("Dosen Wali"), func(c *fiber.Ctx) error { return svc.VerifyAchievementService(c) })
	r.Post("/:id/reject", middleware.RequireRole("Dosen Wali"), func(c *fiber.Ctx) error { return svc.RejectAchievementService(c) })
}