package service

import (
	"database/sql"
	"go-fiber/app/model"
	"go-fiber/app/repository"

	"github.com/gofiber/fiber/v2"
)

// GetAllLecturersService mendapatkan daftar dosen
// @Summary Get all lecturers
// @Description Mengambil daftar semua dosen
// @Tags Lecturers
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} model.APIResponse{data=[]model.LecturerResponse}
// @Router /api/v1/lecturers [get]
func GetAllLecturersService(c *fiber.Ctx, db *sql.DB) error {
	lecturers, err := repository.GetAllLecturers(db)
	if err != nil {
		return c.Status(500).JSON(model.APIResponse{
			Status: "error",
			Error:  "Gagal mengambil daftar dosen",
		})
	}

	return c.JSON(model.APIResponse{
		Status: "success",
		Data:   lecturers,
	})
}

// GetLecturerAdviseesService mendapatkan mahasiswa bimbingan
// @Summary Get advisees
// @Description Mengambil daftar mahasiswa yang dibimbing oleh dosen tertentu
// @Tags Lecturers
// @Accept json
// @Produce json
// @Param id path string true "Lecturer ID"
// @Security BearerAuth
// @Success 200 {object} model.APIResponse{data=[]model.StudentResponse}
// @Router /api/v1/lecturers/{id}/advisees [get]
func GetLecturerAdviseesService(c *fiber.Ctx, db *sql.DB) error {
	lecturerID := c.Params("id")

	advisees, err := repository.GetLecturerAdvisees(db, lecturerID)
	if err != nil {
		return c.Status(500).JSON(model.APIResponse{
			Status: "error",
			Error:  "Gagal mengambil daftar mahasiswa bimbingan",
		})
	}

	return c.JSON(model.APIResponse{
		Status: "success",
		Data:   advisees,
	})
}