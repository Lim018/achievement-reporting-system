package routes

import (
	"database/sql"
	"go-fiber/app/model"
	"go-fiber/app/service"
	"go-fiber/middleware"

	"github.com/gofiber/fiber/v2"
)

func AuthRoutes(app *fiber.App, db *sql.DB) {
	auth := app.Group("/api/v1/auth")

	auth.Post("/login", func(c *fiber.Ctx) error {
		var req model.LoginRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(model.APIResponse{
				Status: "error",
				Error:  "Request body tidak valid",
			})
		}

		if req.Username == "" || req.Password == "" {
			return c.Status(fiber.StatusBadRequest).JSON(model.APIResponse{
				Status: "error",
				Error:  "Username dan password wajib diisi",
			})
		}

		resp, err := service.LoginService(db, req)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(model.APIResponse{
				Status: "error",
				Error:  err.Error(),
			})
		}

		return c.JSON(model.APIResponse{
			Status: "success",
			Data:   resp,
		})
	})

	auth.Post("/refresh", func(c *fiber.Ctx) error {
		var req model.RefreshTokenRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(model.APIResponse{
				Status: "error",
				Error:  "Request body tidak valid",
			})
		}

		if req.RefreshToken == "" {
			return c.Status(fiber.StatusBadRequest).JSON(model.APIResponse{
				Status: "error",
				Error:  "Refresh token wajib diisi",
			})
		}

		resp, err := service.RefreshTokenService(db, req)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(model.APIResponse{
				Status: "error",
				Error:  err.Error(),
			})
		}

		return c.JSON(model.APIResponse{
			Status: "success",
			Data:   resp,
		})
	})

	auth.Post("/logout", middleware.AuthRequired(), func(c *fiber.Ctx) error {
		var req model.RefreshTokenRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(model.APIResponse{
				Status: "error",
				Error:  "Request body tidak valid",
			})
		}

		err := service.LogoutService(db, req.RefreshToken)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(model.APIResponse{
				Status: "error",
				Error:  err.Error(),
			})
		}

		return c.JSON(model.APIResponse{
			Status:  "success",
			Message: "Logout berhasil",
		})
	})

	auth.Get("/profile", middleware.AuthRequired(), func(c *fiber.Ctx) error {
		userID := c.Locals("user_id").(string)

		resp, err := service.GetProfileService(db, userID)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(model.APIResponse{
				Status: "error",
				Error:  err.Error(),
			})
		}

		return c.JSON(model.APIResponse{
			Status: "success",
			Data:   resp,
		})
	})
}
