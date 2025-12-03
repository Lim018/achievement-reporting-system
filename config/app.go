package config

import (
    "database/sql"

    "github.com/gofiber/fiber/v2"
    "go-fiber/middleware"
)

func NewApp(db *sql.DB) *fiber.App {
    app := fiber.New()
    app.Use(middleware.LoggerMiddleware)

    return app
}
