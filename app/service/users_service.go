package service

import (
	"context"
	"database/sql"
	"time"

	"go-fiber/app/model"
	"go-fiber/app/repository"
	"go-fiber/utils"

	"github.com/gofiber/fiber/v2"
)

type UsersService struct {
	Repo *repository.UsersRepo
}

func NewUsersService(db *sql.DB) *UsersService {
	return &UsersService{
		Repo: repository.NewUsersRepo(db),
	}
}

// GetAllUsersService mendapatkan semua pengguna
// @Summary Get all users
// @Description Mengambil daftar semua pengguna (Admin only)
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} model.APIResponse{data=[]model.UserResponse}
// @Failure 500 {object} model.APIResponse
// @Router /api/v1/users [get]
func (s *UsersService) GetAllUsersService(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	users, err := s.Repo.GetAllUsers(ctx)
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

// GetUserDetailService mendapatkan detail pengguna
// @Summary Get user detail
// @Description Mengambil detail pengguna berdasarkan ID
// @Tags Users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Security BearerAuth
// @Success 200 {object} model.APIResponse{data=model.UserResponse}
// @Failure 404 {object} model.APIResponse
// @Router /api/v1/users/{id} [get]
func (s *UsersService) GetUserDetailService(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	id := c.Params("id")

	user, err := s.Repo.GetUserWithRoleInfo(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.Status(fiber.StatusNotFound).JSON(model.APIResponse{
				Status: "error",
				Error:  "Pengguna tidak ditemukan",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(model.APIResponse{
			Status: "error",
			Error:  "Gagal mengambil detail pengguna",
		})
	}

	return c.JSON(model.APIResponse{
		Status: "success",
		Data:   user,
	})
}

// CreateUserService membuat pengguna baru
// @Summary Create new user
// @Description Membuat pengguna baru (Admin only)
// @Tags Users
// @Accept json
// @Produce json
// @Param request body model.CreateUserRequest true "User Data"
// @Security BearerAuth
// @Success 200 {object} model.APIResponse
// @Failure 400 {object} model.APIResponse
// @Router /api/v1/users [post]
func (s *UsersService) CreateUserService(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var req model.CreateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.APIResponse{
			Status: "error",
			Error:  "Body request tidak valid",
		})
	}

	if req.Username == "" || req.Email == "" || req.Password == "" || req.FullName == "" || req.RoleName == "" {
		return c.Status(fiber.StatusBadRequest).JSON(model.APIResponse{
			Status: "error",
			Error:  "Field wajib tidak boleh kosong",
		})
	}

	if req.RoleName == "Mahasiswa" {
		if req.StudentID == nil || *req.StudentID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(model.APIResponse{
				Status: "error",
				Error:  "student_id wajib diisi untuk role Mahasiswa",
			})
		}
	}

	if req.RoleName == "Dosen Wali" {
		if req.LecturerID == nil || *req.LecturerID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(model.APIResponse{
				Status: "error",
				Error:  "lecturer_id wajib diisi untuk role Dosen Wali",
			})
		}
	}

	hashedPass, err := utils.HashPassword(req.Password)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(model.APIResponse{
			Status: "error",
			Error:  "Gagal meng-hash password",
		})
	}

	err = s.Repo.CreateUserTx(ctx, req, hashedPass)
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

// UpdateUserService memperbarui pengguna
// @Summary Update user
// @Description Memperbarui data pengguna
// @Tags Users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Param request body model.UpdateUserRequest true "Update Data"
// @Security BearerAuth
// @Success 200 {object} model.APIResponse
// @Failure 400 {object} model.APIResponse
// @Router /api/v1/users/{id} [put]
func (s *UsersService) UpdateUserService(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	id := c.Params("id")

	var req model.UpdateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.APIResponse{
			Status: "error",
			Error:  "Body request tidak valid",
		})
	}

	_, err := s.Repo.GetUserDetail(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.Status(fiber.StatusNotFound).JSON(model.APIResponse{
				Status: "error",
				Error:  "Pengguna tidak ditemukan",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(model.APIResponse{
			Status: "error",
			Error:  "Gagal memeriksa pengguna",
		})
	}

	if req.AdvisorID != nil && *req.AdvisorID != "" {
		exists, err := s.Repo.CheckAdvisorExists(ctx, *req.AdvisorID)
		if err != nil || !exists {
			return c.Status(fiber.StatusBadRequest).JSON(model.APIResponse{
				Status: "error",
				Error:  "Dosen pembimbing tidak ditemukan",
			})
		}
	}

	err = s.Repo.UpdateUserTx(ctx, id, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(model.APIResponse{
			Status: "error",
			Error:  "Gagal memperbarui pengguna: " + err.Error(),
		})
	}

	return c.JSON(model.APIResponse{
		Status:  "success",
		Message: "User berhasil diperbarui",
	})
}

// DeleteUserService menghapus pengguna
// @Summary Delete user
// @Description Menghapus pengguna secara permanen (Admin only)
// @Tags Users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Security BearerAuth
// @Success 200 {object} model.APIResponse
// @Router /api/v1/users/{id} [delete]
func (s *UsersService) DeleteUserService(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	id := c.Params("id")

	_, err := s.Repo.GetUserDetail(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.Status(fiber.StatusNotFound).JSON(model.APIResponse{
				Status: "error",
				Error:  "Pengguna tidak ditemukan",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(model.APIResponse{
			Status: "error",
			Error:  "Gagal memeriksa pengguna",
		})
	}

	err = s.Repo.DeleteUser(ctx, id)
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

func (s *UsersService) AssignRoleService(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	id := c.Params("id")

	var req model.AssignRoleRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.APIResponse{
			Status: "error",
			Error:  "Body request tidak valid",
		})
	}

	if req.RoleName == "" {
		return c.Status(fiber.StatusBadRequest).JSON(model.APIResponse{
			Status: "error",
			Error:  "role_name wajib diisi",
		})
	}

	err := s.Repo.UpdateUserRole(ctx, id, req.RoleName)
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

func GetAllUsersService(c *fiber.Ctx, db *sql.DB) error {
	service := NewUsersService(db)
	return service.GetAllUsersService(c)
}

func GetUserDetailService(c *fiber.Ctx, db *sql.DB) error {
	service := NewUsersService(db)
	return service.GetUserDetailService(c)
}

func CreateUserService(c *fiber.Ctx, db *sql.DB) error {
	service := NewUsersService(db)
	return service.CreateUserService(c)
}

func UpdateUserService(c *fiber.Ctx, db *sql.DB) error {
	service := NewUsersService(db)
	return service.UpdateUserService(c)
}

func DeleteUserService(c *fiber.Ctx, db *sql.DB) error {
	service := NewUsersService(db)
	return service.DeleteUserService(c)
}

func AssignRoleService(c *fiber.Ctx, db *sql.DB) error {
	service := NewUsersService(db)
	return service.AssignRoleService(c)
}