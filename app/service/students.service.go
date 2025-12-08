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

func GetStudentAchievementsService(c *fiber.Ctx, db *sql.DB, mongoDB *mongo.Database) error {
	studentIDParam := c.Params("id")

	role := c.Locals("role").(string)
	userID := c.Locals("user_id").(string)

	// === 1. Ambil data mahasiswa ===
	student, err := repository.GetStudentByID(db, studentIDParam)
	if err != nil {
		return c.Status(404).JSON(model.APIResponse{
			Status: "error",
			Error:  "Mahasiswa tidak ditemukan",
		})
	}

	// === 2. Validasi akses ===
	if role == "Student" && userID != studentIDParam {
		return c.Status(403).JSON(model.APIResponse{
			Status: "error",
			Error:  "Tidak boleh melihat prestasi mahasiswa lain",
		})
	}

	if role == "Dosen Wali" {
		advisor, _ := student.AdvisorID, student.AdvisorID
		if advisor == nil || *advisor != userID {
			return c.Status(403).JSON(model.APIResponse{
				Status: "error",
				Error:  "Anda bukan dosen wali mahasiswa ini",
			})
		}
	}

	// === 3. Ambil list achievement reference ===
	refRepo := repository.NewAchievementRefRepo(db)
	mongoRepo := repository.NewAchievementMongoRepo(mongoDB)

	refs, err := refRepo.ListByStudentID(studentIDParam)
	if err != nil {
		return c.Status(500).JSON(model.APIResponse{
			Status: "error",
			Error:  "Gagal mengambil data prestasi",
		})
	}

	// === 4. Inject data MongoDB ke tiap ref ===
	for i := range refs {
		doc, err := mongoRepo.FindByHexID(context.Background(), refs[i].MongoID)
		if err == nil {
			refs[i].Achievement = *doc
		}
	}

	// === 5. Format response sesuai SRS ===
	result := model.StudentAchievementsResponse{
		Student:      *student,
		Achievements: refs,
	}

	return c.JSON(model.APIResponse{
		Status: "success",
		Data:   result,
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