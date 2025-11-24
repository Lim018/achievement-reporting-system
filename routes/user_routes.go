package routes

import (
	"database/sql"
	"go-fiber/app/service"
	"go-fiber/middleware"

	"github.com/gofiber/fiber/v2"
)

func UserRoutes(app *fiber.App, db *sql.DB) {
	admin := app.Group("/api/v1", middleware.AuthRequired(), middleware.RequireRole("Admin"))

	// users
	admin.Get("/users", func(c *fiber.Ctx) error {
		return service.GetAllUsersService(c, db)
	})

	admin.Get("/users/:id", func(c *fiber.Ctx) error {
		return service.GetUserDetailService(c, db)
	})

	admin.Post("/users", func(c *fiber.Ctx) error {
		return service.CreateUserService(c, db)
	})

	admin.Put("/users/:id", func(c *fiber.Ctx) error {
		return service.UpdateUserService(c, db)
	})

	admin.Delete("/users/:id", func(c *fiber.Ctx) error {
		return service.DeleteUserService(c, db)
	})

	admin.Put("/users/:id/role", func(c *fiber.Ctx) error {
		return service.AssignRoleService(c, db)
	})

	// students
	admin.Get("/students", func(c *fiber.Ctx) error {
		return service.GetAllStudentsService(c, db)
	})

	admin.Get("/students/:id", func(c *fiber.Ctx) error {
		return service.GetStudentDetailService(c, db)
	})

	admin.Put("/students/:id/advisor", func(c *fiber.Ctx) error {
		return service.UpdateStudentAdvisorService(c, db)
	})

	// lecturers
	admin.Get("/lecturers", func(c *fiber.Ctx) error {
		return service.GetAllLecturersService(c, db)
	})

	admin.Get("/lecturers/:id/advisees", func(c *fiber.Ctx) error {
		return service.GetLecturerAdviseesService(c, db)
	})
}