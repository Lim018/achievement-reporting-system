package service

import (
	"context"
	"database/sql"
	"time"

	"go-fiber/app/model"
	"go-fiber/app/repository"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type StudentsService struct {
	StudentsRepo *repository.StudentsRepo
	AchRefRepo   *repository.AchievementRefRepo
	MongoRepo    *repository.AchievementMongoRepo
}

func NewStudentsService(db *sql.DB, mongoDB *mongo.Database) *StudentsService {
	return &StudentsService{
		StudentsRepo: repository.NewStudentsRepo(db),
		AchRefRepo:   repository.NewAchievementRefRepo(db),
		MongoRepo:    repository.NewAchievementMongoRepo(mongoDB),
	}
}

func (s *StudentsService) GetAllStudentsService(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	students, err := s.StudentsRepo.GetAllStudents(ctx)
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

func (s *StudentsService) GetStudentDetailService(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	studentID := c.Params("id")

	student, err := s.StudentsRepo.GetStudentByID(ctx, studentID)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.Status(fiber.StatusNotFound).JSON(model.APIResponse{
				Status: "error",
				Error:  "Mahasiswa tidak ditemukan",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(model.APIResponse{
			Status: "error",
			Error:  "Gagal mengambil detail mahasiswa",
		})
	}

	return c.JSON(model.APIResponse{
		Status: "success",
		Data:   student,
	})
}

func (s *StudentsService) GetStudentAchievementsService(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	studentID := c.Params("id")
	role := c.Locals("role")
	userID := c.Locals("user_id")

	if role == nil || userID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(model.APIResponse{
			Status: "error",
			Error:  "Unauthorized",
		})
	}

	roleStr := role.(string)
	userIDStr := userID.(string)

	student, err := s.StudentsRepo.GetStudentByID(ctx, studentID)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.Status(404).JSON(model.APIResponse{
				Status: "error",
				Error:  "Mahasiswa tidak ditemukan",
			})
		}
		return c.Status(500).JSON(model.APIResponse{
			Status: "error",
			Error:  "Gagal mengambil data mahasiswa",
		})
	}

	if roleStr == "Mahasiswa" && userIDStr != student.UserID {
		return c.Status(403).JSON(model.APIResponse{
			Status: "error",
			Error:  "Tidak boleh melihat prestasi mahasiswa lain",
		})
	}

	if roleStr == "Dosen Wali" {
		if student.AdvisorID == nil || *student.AdvisorID != userIDStr {
			return c.Status(403).JSON(model.APIResponse{
				Status: "error",
				Error:  "Anda bukan dosen wali mahasiswa ini",
			})
		}
	}

	refs, err := s.AchRefRepo.ListForStudent(student.ID)
	if err != nil {
		return c.Status(500).JSON(model.APIResponse{
			Status: "error",
			Error:  "Gagal mengambil data prestasi",
		})
	}

	if len(refs) == 0 {
		return c.JSON(model.APIResponse{
			Status: "success",
			Data: model.StudentAchievementsResponse{
				Student:      *student,
				Achievements: []model.AchievementDetailResponse{},
			},
		})
	}

	mongoIDs := make([]string, len(refs))
	for i, ref := range refs {
		mongoIDs[i] = ref.MongoID
	}

	projection := bson.M{
		"_id":             1,
		"studentId":       1,
		"achievementType": 1,
		"title":           1,
		"description":     1,
		"details":         1,
		"attachments":     1,
		"tags":            1,
		"points":          1,
		"createdAt":       1,
		"updatedAt":       1,
	}

	achievements, err := s.MongoRepo.BatchFindByHexIDs(ctx, mongoIDs, projection)
	if err != nil {
		achievements = make(map[string]*model.Achievement)
	}

	for i := range refs {
		if ach, ok := achievements[refs[i].MongoID]; ok && ach != nil {
			refs[i].Achievement = *ach
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

func (s *StudentsService) UpdateStudentAdvisorService(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

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

	_, err := s.StudentsRepo.GetStudentByID(ctx, studentID)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.Status(fiber.StatusNotFound).JSON(model.APIResponse{
				Status: "error",
				Error:  "Mahasiswa tidak ditemukan",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(model.APIResponse{
			Status: "error",
			Error:  "Gagal memeriksa mahasiswa",
		})
	}

	lecturersRepo := repository.NewLecturersRepo(s.StudentsRepo.DB)
	_, err = lecturersRepo.GetLecturerByID(ctx, req.AdvisorID)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.Status(fiber.StatusBadRequest).JSON(model.APIResponse{
				Status: "error",
				Error:  "Dosen pembimbing tidak ditemukan",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(model.APIResponse{
			Status: "error",
			Error:  "Gagal memeriksa dosen pembimbing",
		})
	}

	err = s.StudentsRepo.UpdateStudentAdvisor(ctx, studentID, req.AdvisorID)
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

func GetAllStudentsService(c *fiber.Ctx, db *sql.DB, mongoDB *mongo.Database) error {
	service := NewStudentsService(db, mongoDB)
	return service.GetAllStudentsService(c)
}

func GetStudentDetailService(c *fiber.Ctx, db *sql.DB, mongoDB *mongo.Database) error {
	service := NewStudentsService(db, mongoDB)
	return service.GetStudentDetailService(c)
}

func GetStudentAchievementsService(c *fiber.Ctx, db *sql.DB, mongoDB *mongo.Database) error {
	service := NewStudentsService(db, mongoDB)
	return service.GetStudentAchievementsService(c)
}

func UpdateStudentAdvisorService(c *fiber.Ctx, db *sql.DB, mongoDB *mongo.Database) error {
	service := NewStudentsService(db, mongoDB)
	return service.UpdateStudentAdvisorService(c)
}