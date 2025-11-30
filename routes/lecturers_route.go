package routes

import (
    "database/sql"
    "go-fiber/app/service"
    "go-fiber/middleware"

    "github.com/gofiber/fiber/v2"
)

func LecturerRoutes(app *fiber.App, db *sql.DB) {
    lecturer := app.Group("/api/v1/lecturers", middleware.AuthRequired(), middleware.RequireRole("Admin"))

    lecturer.Get("/", func(c *fiber.Ctx) error {
        return service.GetAllLecturersService(c, db)
    })

    lecturer.Get("/:id/advisees", func(c *fiber.Ctx) error {
        return service.GetLecturerAdviseesService(c, db)
    })
}