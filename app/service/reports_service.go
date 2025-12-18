package service

import (
	"context"
	"database/sql"

	"go-fiber/app/model"
	"go-fiber/app/repository"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/mongo"
)

type ReportsService struct {
	Repo *repository.ReportsRepo
}

func NewReportsService(pg *sql.DB, mongoDB *mongo.Database) *ReportsService {
	return &ReportsService{
		Repo: repository.NewReportsRepo(pg, mongoDB),
	}
}

func (s *ReportsService) GetSystemStatistics(c *fiber.Ctx) error {
	ctx := context.Background()

	totals, err := s.Repo.GetTotalCounts()
	if err != nil {
		return c.Status(500).JSON(model.APIResponse{
			Status: "error",
			Error:  "Gagal mengambil total counts",
		})
	}

	statusBreakdown, err := s.Repo.GetAchievementStatusBreakdown()
	if err != nil {
		return c.Status(500).JSON(model.APIResponse{
			Status: "error",
			Error:  "Gagal mengambil status breakdown",
		})
	}

	typeBreakdown, err := s.Repo.GetAchievementTypeBreakdown(ctx)
	if err != nil {
		return c.Status(500).JSON(model.APIResponse{
			Status: "error",
			Error:  "Gagal mengambil type breakdown",
		})
	}

	topStudents, err := s.Repo.GetTopStudents(ctx, 10)
	if err != nil {
		return c.Status(500).JSON(model.APIResponse{
			Status: "error",
			Error:  "Gagal mengambil top students",
		})
	}

	monthlyGrowth, err := s.Repo.GetMonthlyGrowth()
	if err != nil {
		return c.Status(500).JSON(model.APIResponse{
			Status: "error",
			Error:  "Gagal mengambil monthly growth",
		})
	}

	response := model.SystemStatisticsResponse{
		Totals:            totals,
		AchievementStatus: statusBreakdown,
		AchievementByType: typeBreakdown,
		TopStudents:       topStudents,
		MonthlyGrowth:     monthlyGrowth,
	}

	return c.JSON(model.APIResponse{
		Status: "success",
		Data:   response,
	})
}

func (s *ReportsService) GetStudentReport(c *fiber.Ctx) error {
	ctx := context.Background()

	input := c.Params("id")
	role := getUserRole(c)
	userID := getUserID(c)

	var student struct {
		ID     string
		UserID string
	}

	err := s.Repo.PG.QueryRow(`
		SELECT id, user_id
		FROM students
		WHERE id::text = $1 OR student_id = $1
	`, input).Scan(&student.ID, &student.UserID)

	if err != nil {
		if err == sql.ErrNoRows {
			return c.Status(404).JSON(model.APIResponse{
				Status: "error",
				Error:  "Mahasiswa tidak ditemukan",
			})
		}
		return c.Status(500).JSON(model.APIResponse{
			Status: "error",
			Error:  "Database error: " + err.Error(),
		})
	}

	if role == "Mahasiswa" && userID != student.UserID {
		return c.Status(403).JSON(model.APIResponse{
			Status: "error",
			Error:  "Anda hanya dapat melihat laporan Anda sendiri",
		})
	}

	if role == "Dosen Wali" {
		var advisorUserID string

		err := s.Repo.PG.QueryRow(`
			SELECT l.user_id
			FROM students s
			JOIN lecturers l ON s.advisor_id = l.id
			WHERE s.id = $1
		`, student.ID).Scan(&advisorUserID)

		if err != nil || advisorUserID != userID {
			return c.Status(403).JSON(model.APIResponse{
				Status: "error",
				Error:  "Anda bukan dosen wali mahasiswa ini",
			})
		}
	}

	studentInfo, err := s.Repo.GetStudentInfo(student.ID)
	if err != nil {
		return c.Status(500).JSON(model.APIResponse{
			Status: "error",
			Error:  "Gagal mengambil informasi mahasiswa",
		})
	}

	summary, _ := s.Repo.GetStudentSummary(ctx, student.ID)
	statusBreakdown, _ := s.Repo.GetStudentAchievementStatusBreakdown(student.ID)
	typeBreakdown, _ := s.Repo.GetStudentAchievementTypeBreakdown(ctx, student.ID)
	monthlyGrowth, _ := s.Repo.GetStudentMonthlyGrowth(student.ID)
	achievements, _ := s.Repo.GetStudentAchievements(ctx, student.ID)

	return c.JSON(model.APIResponse{
		Status: "success",
		Data: model.StudentReportResponse{
			Student:           *studentInfo,
			Summary:           summary,
			AchievementStatus: statusBreakdown,
			AchievementByType: typeBreakdown,
			MonthlyGrowth:     monthlyGrowth,
			Achievements:      achievements,
		},
	})
}