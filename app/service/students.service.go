package service

import (
	"database/sql"
	"go-fiber/app/model"
	"go-fiber/app/repository"

	"github.com/gofiber/fiber/v2"
)

func GetAllStudentsService(c *fiber.Ctx, db *sql.DB) error {
	students, err := repository.GetAllStudents(db)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(model.APIResponse{
			Status: "error",
			Error:  "Gagal mengambil daftar mahasiswa",
		})
	}

	return c.JSON(model.APIResponse{
		Status: "success",
		Data:   students,
	})
}

func GetStudentDetailService(c *fiber.Ctx, db *sql.DB) error {
	studentID := c.Params("id")

	student, err := repository.GetStudentByID(db, studentID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(model.APIResponse{
			Status: "error",
			Error:  "Mahasiswa tidak ditemukan",
		})
	}

	return c.JSON(model.APIResponse{
		Status: "success",
		Data:   student,
	})
}

func UpdateStudentAdvisorService(c *fiber.Ctx, db *sql.DB) error {
	studentID := c.Params("id")

	var req model.UpdateStudentAdvisorRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.APIResponse{
			Status: "error",
			Error:  "Request body tidak valid",
		})
	}

	if req.AdvisorID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(model.APIResponse{
			Status: "error",
			Error:  "advisor_id wajib diisi",
		})
	}

	_, err := repository.GetStudentByID(db, studentID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(model.APIResponse{
			Status: "error",
			Error:  "Mahasiswa tidak ditemukan",
		})
	}

	_, err = repository.GetLecturerByID(db, req.AdvisorID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.APIResponse{
			Status: "error",
			Error:  "Dosen pembimbing tidak ditemukan",
		})
	}

	err = repository.UpdateStudentAdvisor(db, studentID, req.AdvisorID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(model.APIResponse{
			Status: "error",
			Error:  "Gagal mengupdate dosen pembimbing",
		})
	}

	return c.JSON(model.APIResponse{
		Status:  "success",
		Message: "Dosen pembimbing berhasil diupdate",
	})
}