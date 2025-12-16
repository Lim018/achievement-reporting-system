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

// Get system statistics
func (s *ReportsService) GetSystemStatistics(c *fiber.Ctx) error {
	ctx := context.Background()

	// Get totals
	totals, err := s.Repo.GetTotalCounts()
	if err != nil {
		return c.Status(500).JSON(model.APIResponse{
			Status: "error",
			Error:  "Gagal mengambil total counts",
		})
	}

	// Get achievement status breakdown
	statusBreakdown, err := s.Repo.GetAchievementStatusBreakdown()
	if err != nil {
		return c.Status(500).JSON(model.APIResponse{
			Status: "error",
			Error:  "Gagal mengambil status breakdown",
		})
	}

	// Get achievement type breakdown
	typeBreakdown, err := s.Repo.GetAchievementTypeBreakdown(ctx)
	if err != nil {
		return c.Status(500).JSON(model.APIResponse{
			Status: "error",
			Error:  "Gagal mengambil type breakdown",
		})
	}

	// Get top students
	topStudents, err := s.Repo.GetTopStudents(ctx, 10)
	if err != nil {
		return c.Status(500).JSON(model.APIResponse{
			Status: "error",
			Error:  "Gagal mengambil top students",
		})
	}

	// Get monthly growth
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

// Get student report
func (s *ReportsService) GetStudentReport(c *fiber.Ctx) error {
	ctx := context.Background()
	studentID := c.Params("id")

	// Validate access - students can only view their own report
	role := getUserRole(c)
	userID := getUserID(c)

	if role == "Mahasiswa" && userID != studentID {
		return c.Status(403).JSON(model.APIResponse{
			Status: "error",
			Error:  "Anda hanya dapat melihat laporan Anda sendiri",
		})
	}

	// For Dosen Wali, verify they are the advisor
	if role == "Dosen Wali" {
		var advisorID string
		err := s.Repo.PG.QueryRow(`
			SELECT advisor_id FROM students WHERE id = $1
		`, studentID).Scan(&advisorID)

		if err != nil || advisorID != userID {
			return c.Status(403).JSON(model.APIResponse{
				Status: "error",
				Error:  "Anda bukan dosen wali mahasiswa ini",
			})
		}
	}

	// Get student info
	studentInfo, err := s.Repo.GetStudentInfo(studentID)
	if err != nil {
		return c.Status(404).JSON(model.APIResponse{
			Status: "error",
			Error:  "Mahasiswa tidak ditemukan",
		})
	}

	// Get student summary
	summary, err := s.Repo.GetStudentSummary(ctx, studentID)
	if err != nil {
		return c.Status(500).JSON(model.APIResponse{
			Status: "error",
			Error:  "Gagal mengambil summary",
		})
	}

	// Get achievement status breakdown
	statusBreakdown, err := s.Repo.GetStudentAchievementStatusBreakdown(studentID)
	if err != nil {
		return c.Status(500).JSON(model.APIResponse{
			Status: "error",
			Error:  "Gagal mengambil status breakdown",
		})
	}

	// Get achievement type breakdown
	typeBreakdown, err := s.Repo.GetStudentAchievementTypeBreakdown(ctx, studentID)
	if err != nil {
		return c.Status(500).JSON(model.APIResponse{
			Status: "error",
			Error:  "Gagal mengambil type breakdown",
		})
	}

	// Get monthly growth
	monthlyGrowth, err := s.Repo.GetStudentMonthlyGrowth(studentID)
	if err != nil {
		return c.Status(500).JSON(model.APIResponse{
			Status: "error",
			Error:  "Gagal mengambil monthly growth",
		})
	}

	// Get achievements list
	achievements, err := s.Repo.GetStudentAchievements(ctx, studentID)
	if err != nil {
		return c.Status(500).JSON(model.APIResponse{
			Status: "error",
			Error:  "Gagal mengambil daftar achievements",
		})
	}

	response := model.StudentReportResponse{
		Student:           *studentInfo,
		Summary:           summary,
		AchievementStatus: statusBreakdown,
		AchievementByType: typeBreakdown,
		MonthlyGrowth:     monthlyGrowth,
		Achievements:      achievements,
	}

	return c.JSON(model.APIResponse{
		Status: "success",
		Data:   response,
	})
}