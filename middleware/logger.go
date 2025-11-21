package middleware

import (
    "fmt"
    "github.com/gofiber/fiber/v2"
)

func LoggerMiddleware(c *fiber.Ctx) error {
    fmt.Println("Request:", c.Method(), c.Path())
    return c.Next()
}
