package routes

import (
    "database/sql"
    "go-fiber/app/service"
    "go-fiber/middleware"

    "github.com/gofiber/fiber/v2"
)

func UserRoutes(app *fiber.App, db *sql.DB) {
    user := app.Group("/api/v1/users", middleware.AuthRequired(), middleware.RequireRole("Admin"))

    user.Get("/", func(c *fiber.Ctx) error {
        return service.GetAllUsersService(c, db)
    })

    user.Get("/:id", func(c *fiber.Ctx) error {
        return service.GetUserDetailService(c, db)
    })

    user.Post("/", func(c *fiber.Ctx) error {
        return service.CreateUserService(c, db)
    })

    user.Put("/:id", func(c *fiber.Ctx) error {
        return service.UpdateUserService(c, db)
    })

    user.Delete("/:id", func(c *fiber.Ctx) error {
        return service.DeleteUserService(c, db)
    })

    user.Put("/:id/role", func(c *fiber.Ctx) error {
        return service.AssignRoleService(c, db)
    })
}