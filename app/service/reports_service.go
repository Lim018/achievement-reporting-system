package service

import (
	"context"
	"database/sql"
	"sync"
	"time"

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
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var response model.SystemStatisticsResponse
	var wg sync.WaitGroup
	var mu sync.Mutex
	errChan := make(chan error, 5)

	wg.Add(5)

	go func() {
		defer wg.Done()
		totals, err := s.Repo.GetTotalCounts(ctx)
		if err != nil {
			errChan <- err
			return
		}
		mu.Lock()
		response.Totals = totals
		mu.Unlock()
	}()

	go func() {
		defer wg.Done()
		statusBreakdown, err := s.Repo.GetAchievementStatusBreakdown(ctx)
		if err != nil {
			errChan <- err
			return
		}
		mu.Lock()
		response.AchievementStatus = statusBreakdown
		mu.Unlock()
	}()

	go func() {
		defer wg.Done()
		typeBreakdown, err := s.Repo.GetAchievementTypeBreakdown(ctx)
		if err != nil {
			errChan <- err
			return
		}
		mu.Lock()
		response.AchievementByType = typeBreakdown
		mu.Unlock()
	}()

	go func() {
		defer wg.Done()
		topStudents, err := s.Repo.GetTopStudents(ctx, 10)
		if err != nil {
			errChan <- err
			return
		}
		mu.Lock()
		response.TopStudents = topStudents
		mu.Unlock()
	}()

	go func() {
		defer wg.Done()
		monthlyGrowth, err := s.Repo.GetMonthlyGrowth(ctx)
		if err != nil {
			errChan <- err
			return
		}
		mu.Lock()
		response.MonthlyGrowth = monthlyGrowth
		mu.Unlock()
	}()

	wg.Wait()
	close(errChan)

	if len(errChan) > 0 {
		err := <-errChan
		return c.Status(500).JSON(model.APIResponse{
			Status: "error",
			Error:  "Gagal mengambil statistik: " + err.Error(),
		})
	}

	return c.JSON(model.APIResponse{
		Status: "success",
		Data:   response,
	})
}

func (s *ReportsService) GetStudentReport(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	input := c.Params("id")
	role := getUserRole(c)
	userID := getUserID(c)

	student, err := s.Repo.GetStudentByIDOrStudentID(ctx, input)
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
		advisorUserID, err := s.Repo.GetAdvisorUserID(ctx, student.ID)
		if err != nil || advisorUserID != userID {
			return c.Status(403).JSON(model.APIResponse{
				Status: "error",
				Error:  "Anda bukan dosen wali mahasiswa ini",
			})
		}
	}

	var response model.StudentReportResponse
	var wg sync.WaitGroup
	var mu sync.Mutex
	errChan := make(chan error, 6)

	wg.Add(6)

	go func() {
		defer wg.Done()
		studentInfo, err := s.Repo.GetStudentInfo(ctx, student.ID)
		if err != nil {
			errChan <- err
			return
		}
		mu.Lock()
		response.Student = *studentInfo
		mu.Unlock()
	}()

	go func() {
		defer wg.Done()
		summary, err := s.Repo.GetStudentSummary(ctx, student.ID)
		if err != nil {
			summary = model.StudentSummary{}
		}
		mu.Lock()
		response.Summary = summary
		mu.Unlock()
	}()

	go func() {
		defer wg.Done()
		statusBreakdown, err := s.Repo.GetStudentAchievementStatusBreakdown(ctx, student.ID)
		if err != nil {
			statusBreakdown = make(map[string]int)
		}
		mu.Lock()
		response.AchievementStatus = statusBreakdown
		mu.Unlock()
	}()

	go func() {
		defer wg.Done()
		typeBreakdown, err := s.Repo.GetStudentAchievementTypeBreakdown(ctx, student.ID)
		if err != nil {
			typeBreakdown = make(map[string]int)
		}
		mu.Lock()
		response.AchievementByType = typeBreakdown
		mu.Unlock()
	}()

	go func() {
		defer wg.Done()
		monthlyGrowth, err := s.Repo.GetStudentMonthlyGrowth(ctx, student.ID)
		if err != nil {
			monthlyGrowth = []model.MonthlyGrowthData{}
		}
		mu.Lock()
		response.MonthlyGrowth = monthlyGrowth
		mu.Unlock()
	}()

	go func() {
		defer wg.Done()
		achievements, err := s.Repo.GetStudentAchievements(ctx, student.ID)
		if err != nil {
			achievements = []model.StudentAchievementInfo{}
		}
		mu.Lock()
		response.Achievements = achievements
		mu.Unlock()
	}()

	wg.Wait()
	close(errChan)

	if len(errChan) > 0 {
		err := <-errChan
		return c.Status(500).JSON(model.APIResponse{
			Status: "error",
			Error:  "Gagal mengambil informasi mahasiswa: " + err.Error(),
		})
	}

	return c.JSON(model.APIResponse{
		Status: "success",
		Data:   response,
	})
}