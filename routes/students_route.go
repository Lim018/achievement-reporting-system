package routes

import (
    "database/sql"
    "go-fiber/app/service"
    "go-fiber/middleware"
	"go.mongodb.org/mongo-driver/mongo"

    "github.com/gofiber/fiber/v2"
)

func StudentRoutes(app *fiber.App, db *sql.DB, mongoDB *mongo.Database) {
    student := app.Group("/api/v1/students", middleware.AuthRequired(), middleware.RequirePermission("user:manage"))

    student.Get("/", func(c *fiber.Ctx) error {
        return service.GetAllStudentsService(c, db)
    })

    student.Get("/:id", func(c *fiber.Ctx) error {
        return service.GetStudentDetailService(c, db)
    })

    student.Get("/:id/achievements", func(c *fiber.Ctx) error {
        return service.GetStudentAchievementsService(c, db, mongoDB)
    },
)

    student.Put("/:id/advisor", func(c *fiber.Ctx) error {
        return service.UpdateStudentAdvisorService(c, db)
    })
}