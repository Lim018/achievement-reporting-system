package service

import (
	"context"
	"database/sql"
	"go-fiber/app/model"
	"go-fiber/app/repository"
	"go.mongodb.org/mongo-driver/mongo"

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

func GetStudentAchievementsService(
	c *fiber.Ctx,
	db *sql.DB,
	mongoDB *mongo.Database,
) error {

	studentID := c.Params("id")
	role := c.Locals("role").(string)
	userID := c.Locals("user_id").(string)

	student, err := repository.GetStudentByID(db, studentID)
	if err != nil {
		return c.Status(404).JSON(model.APIResponse{
			Status: "error",
			Error:  "Mahasiswa tidak ditemukan",
		})
	}

	// 🔐 RBAC
	if role == "Student" && userID != student.UserID {
		return c.Status(403).JSON(model.APIResponse{
			Status: "error",
			Error:  "Tidak boleh melihat prestasi mahasiswa lain",
		})
	}

	if role == "Dosen Wali" {
		if student.AdvisorID == nil || *student.AdvisorID != userID {
			return c.Status(403).JSON(model.APIResponse{
				Status: "error",
				Error:  "Anda bukan dosen wali mahasiswa ini",
			})
		}
	}

	refRepo := repository.NewAchievementRefRepo(db)
	mongoRepo := repository.NewAchievementMongoRepo(mongoDB)

	refs, err := refRepo.ListByStudentID(student.ID)
	if err != nil {
		return c.Status(500).JSON(model.APIResponse{
			Status: "error",
			Error:  "Gagal mengambil data prestasi",
		})
	}

	for i := range refs {
		doc, err := mongoRepo.FindByHexID(context.Background(), refs[i].MongoID)
		if err == nil {
			refs[i].Achievement = *doc
		}
	}

	return c.JSON(model.APIResponse{
		Status: "success",
		Data: model.StudentAchievementsResponse{
			Student:      *student,
			Achievements: refs,
		},
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