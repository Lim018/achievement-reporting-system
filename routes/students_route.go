package routes

import (
    "database/sql"
    "go-fiber/app/service"
    "go-fiber/middleware"

    "github.com/gofiber/fiber/v2"
)

func StudentRoutes(app *fiber.App, db *sql.DB) {
    student := app.Group("/api/v1/students", middleware.AuthRequired(), middleware.RequireRole("Admin"))

    student.Get("/", func(c *fiber.Ctx) error {
        return service.GetAllStudentsService(c, db)
    })

    student.Get("/:id", func(c *fiber.Ctx) error {
        return service.GetStudentDetailService(c, db)
    })

    student.Put("/:id/advisor", func(c *fiber.Ctx) error {
        return service.UpdateStudentAdvisorService(c, db)
    })
}