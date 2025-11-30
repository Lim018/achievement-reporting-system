package service

import (
	"database/sql"
	"go-fiber/app/model"
	"go-fiber/app/repository"
	"go-fiber/utils"

	"github.com/gofiber/fiber/v2"
)

func GetAllUsersService(c *fiber.Ctx, db *sql.DB) error {
	users, err := repository.GetAllUsers(db)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(model.APIResponse{
			Status: "error",
			Error:  "Gagal mengambil daftar pengguna",
		})
	}

	return c.JSON(model.APIResponse{
		Status: "success",
		Data:   users,
	})
}

func GetUserDetailService(c *fiber.Ctx, db *sql.DB) error {
	id := c.Params("id")

	user, err := repository.GetUserDetail(db, id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(model.APIResponse{
			Status: "error",
			Error:  "Pengguna tidak ditemukan",
		})
	}

	return c.JSON(model.APIResponse{
		Status: "success",
		Data:   user,
	})
}

func CreateUserService(c *fiber.Ctx, db *sql.DB) error {
	var req model.CreateUserRequest

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.APIResponse{
			Status: "error",
			Error:  "Body request tidak valid",
		})
	}

	hashedPass, err := utils.HashPassword(req.Password)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(model.APIResponse{
			Status: "error",
			Error:  "Gagal meng-hash password",
		})
	}

	err = repository.CreateUserTx(db, req, hashedPass)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(model.APIResponse{
			Status: "error",
			Error:  "Gagal membuat pengguna baru: " + err.Error(),
		})
	}

	return c.JSON(model.APIResponse{
		Status:  "success",
		Message: "User berhasil dibuat",
	})
}

func UpdateUserService(c *fiber.Ctx, db *sql.DB) error {
	id := c.Params("id")

	var req model.UpdateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.APIResponse{
			Status: "error",
			Error:  "Body request tidak valid",
		})
	}

	err := repository.UpdateUser(db, id, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(model.APIResponse{
			Status: "error",
			Error:  "Gagal memperbarui pengguna",
		})
	}

	return c.JSON(model.APIResponse{
		Status:  "success",
		Message: "User berhasil diperbarui",
	})
}

func DeleteUserService(c *fiber.Ctx, db *sql.DB) error {
	id := c.Params("id")

	err := repository.DeleteUser(db, id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(model.APIResponse{
			Status: "error",
			Error:  "Gagal menghapus pengguna",
		})
	}

	return c.JSON(model.APIResponse{
		Status:  "success",
		Message: "User berhasil dihapus",
	})
}

func AssignRoleService(c *fiber.Ctx, db *sql.DB) error {
	id := c.Params("id")

	var req model.AssignRoleRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.APIResponse{
			Status: "error",
			Error:  "Body request tidak valid",
		})
	}

	err := repository.UpdateUserRole(db, id, req.RoleName)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(model.APIResponse{
			Status: "error",
			Error:  "Gagal mengubah role pengguna",
		})
	}

	return c.JSON(model.APIResponse{
		Status:  "success",
		Message: "Role berhasil diperbarui",
	})
}