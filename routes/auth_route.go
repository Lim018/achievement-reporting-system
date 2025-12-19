package routes

import (
	"database/sql"
	"go-fiber/app/model"
	"go-fiber/app/service"
	"go-fiber/middleware"

	"github.com/gofiber/fiber/v2"
)

type AuthHandler struct {
	DB *sql.DB
}

func AuthRoutes(app *fiber.App, db *sql.DB) {
	handler := &AuthHandler{DB: db}
	auth := app.Group("/api/v1/auth")

	auth.Post("/login", handler.Login)
	auth.Post("/refresh", handler.Refresh)
	auth.Post("/logout", middleware.AuthRequired(), handler.Logout)
	auth.Get("/profile", middleware.AuthRequired(), handler.GetProfile)
}

// Login menangani autentikasi pengguna
// @Summary Login user
// @Description Masuk ke sistem menggunakan username/email dan password
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body model.LoginRequest true "Login Credentials"
// @Success 200 {object} model.APIResponse{data=model.LoginResponse}
// @Failure 400 {object} model.APIResponse
// @Failure 401 {object} model.APIResponse
// @Router /api/v1/auth/login [post]
func (h *AuthHandler) Login(c *fiber.Ctx) error {
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

	resp, err := service.LoginService(h.DB, req)
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
}

// Refresh memperbarui access token
// @Summary Refresh token
// @Description Mendapatkan access token baru menggunakan refresh token
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body model.RefreshTokenRequest true "Refresh Token"
// @Success 200 {object} model.APIResponse{data=model.LoginResponse}
// @Failure 400 {object} model.APIResponse
// @Failure 401 {object} model.APIResponse
// @Router /api/v1/auth/refresh [post]
func (h *AuthHandler) Refresh(c *fiber.Ctx) error {
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

	resp, err := service.RefreshTokenService(h.DB, req)
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
}

// Logout keluar dari sistem
// @Summary Logout user
// @Description Invalidate refresh token (Client side action mainly)
// @Tags Auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body model.RefreshTokenRequest true "Refresh Token"
// @Success 200 {object} model.APIResponse
// @Failure 400 {object} model.APIResponse
// @Router /api/v1/auth/logout [post]
func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	var req model.RefreshTokenRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.APIResponse{
			Status: "error",
			Error:  "Request body tidak valid",
		})
	}

	err := service.LogoutService(h.DB, req.RefreshToken)
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
}

// GetProfile mendapatkan profil user login
// @Summary Get user profile
// @Description Mengambil data profil pengguna yang sedang login
// @Tags Auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} model.APIResponse{data=model.UserResponse}
// @Failure 404 {object} model.APIResponse
// @Router /api/v1/auth/profile [get]
func (h *AuthHandler) GetProfile(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)

	resp, err := service.GetProfileService(h.DB, userID)
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
}