package service

import (
	"database/sql"
	"go-fiber/app/model"
	"go-fiber/app/repository"

	"github.com/gofiber/fiber/v2"
)

func GetAllLecturersService(c *fiber.Ctx, db *sql.DB) error {
	lecturers, err := repository.GetAllLecturers(db)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(model.APIResponse{
			Status: "error",
			Error:  "Gagal mengambil daftar dosen",
		})
	}

	return c.JSON(model.APIResponse{
		Status: "success",
		Data:   lecturers,
	})
}

func GetLecturerAdviseesService(c *fiber.Ctx, db *sql.DB) error {
	advisees, err := repository.GetLecturerAdvisees(db, c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(model.APIResponse{
			Status: "error",
			Error:  "Gagal mengambil daftar mahasiswa bimbingan",
		})
	}

	return c.JSON(model.APIResponse{
		Status: "success",
		Data:   advisees,
	})
}