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
	achievement := app.Group("/api/v1/achievements", middleware.AuthRequired())

	achievement.Get("/", middleware.RequirePermission("achievement:read"), svc.ListAchievementsService,)

	achievement.Get("/:id", middleware.RequirePermission("achievement:read"), svc.GetAchievementDetailService,)

	achievement.Post("/", middleware.RequirePermission("achievement:create"), svc.CreateAchievementService,)

	achievement.Put("/:id", middleware.RequirePermission("achievement:update"), svc.UpdateAchievementService,)

	achievement.Delete("/:id", middleware.RequirePermission("achievement:delete"), svc.DeleteAchievementService,)

	achievement.Post("/:id/submit", middleware.RequirePermission("achievement:update"), svc.SubmitAchievementService,)

	achievement.Post("/:id/verify", middleware.RequirePermission("achievement:verify"), svc.VerifyAchievementService,)

	achievement.Post("/:id/reject", middleware.RequirePermission("achievement:verify"), svc.RejectAchievementService,)

	achievement.Get("/:id/history", middleware.RequirePermission("achievement:read"), svc.GetHistoryService,)

	achievement.Post("/:id/attachments", middleware.RequirePermission("achievement:update"), svc.UploadAttachmentsService,)
}