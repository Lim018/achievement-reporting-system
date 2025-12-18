package service

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	"go-fiber/app/model"
	"go-fiber/app/repository"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type AchievementService struct {
	PGRepo  *repository.AchievementRefRepo
	Mongo   *repository.AchievementMongoRepo
	PG      *sql.DB
	MongoDB *mongo.Database
}

func NewAchievementService(pg *sql.DB, mongoDB *mongo.Database) *AchievementService {
	return &AchievementService{
		PGRepo:  repository.NewAchievementRefRepo(pg),
		Mongo:   repository.NewAchievementMongoRepo(mongoDB),
		PG:      pg,
		MongoDB: mongoDB,
	}
}

func getUserID(c *fiber.Ctx) string {
	v := c.Locals("user_id")
	if v == nil {
		return ""
	}
	return v.(string)
}

func getUserRole(c *fiber.Ctx) string {
	v := c.Locals("role")
	if v == nil {
		return ""
	}
	return v.(string)
}

func getStudentID(c *fiber.Ctx, db *sql.DB) (string, error) {
	userID := getUserID(c)
	if userID == "" {
		return "", fmt.Errorf("unauthorized")
	}

	studentID, err := repository.GetStudentIDByUserID(db, userID)
	if err != nil || studentID == "" {
		return "", fmt.Errorf("not student")
	}
	return studentID, nil
}

func (s *AchievementService) CreateAchievementService(c *fiber.Ctx) error {
	studentID, err := getStudentID(c, s.PG)
	if err != nil {
		return c.Status(403).JSON(model.APIResponse{
			Status: "error",
			Error:  "Akun ini bukan mahasiswa",
		})
	}

	var req model.CreateAchievementRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(model.APIResponse{
			Status: "error",
			Error:  "Body tidak valid",
		})
	}

	now := time.Now()

	ach := model.Achievement{
		StudentID:       studentID,
		AchievementType: req.AchievementType,
		Title:           req.Title,
		Description:     req.Description,
		Tags:            req.Tags,
		Points:          0,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if req.Details != nil {
		ach.Details.CustomFields = req.Details
	}

	ctx := context.Background()
	mongoID, err := s.Mongo.Create(ctx, ach)
	if err != nil {
		return c.Status(500).JSON(model.APIResponse{
			Status: "error",
			Error:  "Gagal membuat MongoDB document",
		})
	}

	refID, err := s.PGRepo.CreateReference(studentID, mongoID)
	if err != nil {
		_ = s.Mongo.DeleteByHexID(ctx, mongoID)
		return c.Status(500).JSON(model.APIResponse{
			Status: "error",
			Error:  "Gagal membuat reference",
		})
	}

	return c.JSON(model.APIResponse{
		Status: "success",
		Data: fiber.Map{
			"reference_id": refID,
			"mongo_id":     mongoID,
		},
	})
}

func (s *AchievementService) UpdateAchievementService(c *fiber.Ctx) error {
	studentID, err := getStudentID(c, s.PG)
	if err != nil {
		return c.Status(403).JSON(model.APIResponse{Status: "error", Error: "Unauthorized"})
	}

	refID := c.Params("id")
	ref, err := s.PGRepo.GetReference(refID)
	if err != nil || ref.StudentID != studentID {
		return c.Status(404).JSON(model.APIResponse{Status: "error", Error: "Reference tidak ditemukan"})
	}

	if ref.ReferenceStatus != "draft" && ref.ReferenceStatus != "rejected" {
		return c.Status(400).JSON(model.APIResponse{Status: "error", Error: "Tidak bisa diupdate"})
	}

	var req model.UpdateAchievementRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(model.APIResponse{Status: "error", Error: "Body tidak valid"})
	}

	update := bson.M{}

	if req.Title != nil {
		update["title"] = *req.Title
	}
	if req.Description != nil {
		update["description"] = *req.Description
	}
	if req.Tags != nil {
		update["tags"] = req.Tags
	}
	if req.Details != nil {
		update["details"] = req.Details
	}
	if req.Points != nil {
		update["points"] = *req.Points
	}

	if len(update) == 0 {
		return c.Status(400).JSON(model.APIResponse{Status: "error", Error: "Tidak ada perubahan"})
	}

	update["updatedAt"] = time.Now()

	if err := s.Mongo.UpdateByHexID(context.Background(), ref.MongoID, update); err != nil {
		return c.Status(500).JSON(model.APIResponse{Status: "error", Error: "Gagal update MongoDB"})
	}

	return c.JSON(model.APIResponse{Status: "success", Message: "Updated"})
}

func (s *AchievementService) DeleteAchievementService(c *fiber.Ctx) error {
	studentID, err := getStudentID(c, s.PG)
	if err != nil {
		return c.Status(403).JSON(model.APIResponse{Status: "error", Error: "Unauthorized"})
	}

	refID := c.Params("id")
	ref, err := s.PGRepo.GetReference(refID)
	if err != nil || ref.StudentID != studentID {
		return c.Status(404).JSON(model.APIResponse{Status: "error", Error: "Reference tidak ditemukan"})
	}

	if ref.ReferenceStatus != "draft" {
		return c.Status(400).JSON(model.APIResponse{Status: "error", Error: "Hanya draft yang bisa dihapus"})
	}

	if err := s.PGRepo.SoftDeleteReference(refID); err != nil {
		return c.Status(500).JSON(model.APIResponse{Status: "error", Error: "Gagal menghapus"})
	}

	return c.JSON(model.APIResponse{Status: "success", Message: "deleted"})
}

func (s *AchievementService) SubmitAchievementService(c *fiber.Ctx) error {
	studentID, err := getStudentID(c, s.PG)
	if err != nil {
		return c.Status(403).JSON(model.APIResponse{Status: "error", Error: "Unauthorized"})
	}

	refID := c.Params("id")
	ref, err := s.PGRepo.GetReference(refID)
	if err != nil || ref.StudentID != studentID {
		return c.Status(404).JSON(model.APIResponse{Status: "error", Error: "Reference tidak ditemukan"})
	}

	if ref.ReferenceStatus != "draft" && ref.ReferenceStatus != "rejected" {
		return c.Status(400).JSON(model.APIResponse{Status: "error", Error: "Tidak bisa submit"})
	}

	if err := s.PGRepo.SubmitReference(refID); err != nil {
		return c.Status(500).JSON(model.APIResponse{Status: "error", Error: "Gagal submit"})
	}

	return c.JSON(model.APIResponse{Status: "success", Message: "submitted"})
}

func (s *AchievementService) VerifyAchievementService(c *fiber.Ctx) error {
	advisorID := getUserID(c)
	if getUserRole(c) != "Dosen Wali" {
		return c.Status(403).JSON(model.APIResponse{Status: "error", Error: "Akses ditolak"})
	}

	refID := c.Params("id")
	ref, err := s.PGRepo.GetReferenceWithAdvisor(refID, advisorID)
	if err != nil || ref.AdvisorID != advisorID {
		return c.Status(403).JSON(model.APIResponse{Status: "error", Error: "Bukan dosen wali"})
	}

	var body struct {
		Points int `json:"points"`
	}
	if err := c.BodyParser(&body); err != nil || body.Points <= 0 {
		return c.Status(400).JSON(model.APIResponse{Status: "error", Error: "Points tidak valid"})
	}

	_ = s.Mongo.UpdateByHexID(context.Background(), ref.MongoID, bson.M{
		"points":    body.Points,
		"updatedAt": time.Now(),
	})

	if err := s.PGRepo.VerifyReference(refID, advisorID); err != nil {
		return c.Status(500).JSON(model.APIResponse{Status: "error", Error: "Gagal verifikasi"})
	}

	return c.JSON(model.APIResponse{Status: "success", Message: "verified"})
}

func (s *AchievementService) RejectAchievementService(c *fiber.Ctx) error {
	advisorID := getUserID(c)
	refID := c.Params("id")

	ref, err := s.PGRepo.GetReferenceWithAdvisor(refID, advisorID)
	if err != nil || ref.AdvisorID != advisorID {
		return c.Status(403).JSON(model.APIResponse{Status: "error", Error: "Bukan dosen wali"})
	}

	var body struct {
		Note string `json:"note"`
	}
	_ = c.BodyParser(&body)

	if err := s.PGRepo.RejectReference(refID, advisorID, body.Note); err != nil {
		return c.Status(500).JSON(model.APIResponse{Status: "error", Error: "Gagal reject"})
	}

	return c.JSON(model.APIResponse{Status: "success", Message: "rejected"})
}

func (s *AchievementService) ListAchievementsService(c *fiber.Ctx) error {
	role := getUserRole(c)
	userID := getUserID(c)

	var list []model.AchievementDetailResponse
	var err error

	switch role {
	case "Admin":
		list, err = s.PGRepo.ListForAdmin()
	case "Dosen Wali":
		list, err = s.PGRepo.ListForAdvisor(userID)
	default:
		studentID, _ := getStudentID(c, s.PG)
		list, err = s.PGRepo.ListForStudent(studentID)
	}

	if err != nil {
		return c.Status(500).JSON(model.APIResponse{Status: "error", Error: "Gagal mengambil data"})
	}

	for i := range list {
		doc, _ := s.Mongo.FindByHexID(context.Background(), list[i].MongoID)
		list[i].Achievement = *doc
	}

	return c.JSON(model.APIResponse{Status: "success", Data: list})
}

func (s *AchievementService) GetAchievementDetailService(c *fiber.Ctx) error {
	refID := c.Params("id")
	role := getUserRole(c)

	ref, err := s.PGRepo.GetReferenceDetail(refID)
	if err != nil {
		return c.Status(404).JSON(model.APIResponse{
			Status: "error",
			Error:  "Reference tidak ditemukan",
		})
	}

	switch role {

	case "Mahasiswa":
		studentID, err := getStudentID(c, s.PG)
		if err != nil || ref.StudentID != studentID {
			return c.Status(403).JSON(model.APIResponse{
				Status: "error",
				Error:  "Tidak boleh melihat data milik orang lain",
			})
		}

	case "Dosen Wali":
		userID := getUserID(c)
		if ref.AdvisorID != userID {
			return c.Status(403).JSON(model.APIResponse{
				Status: "error",
				Error:  "Anda bukan dosen wali mahasiswa ini",
			})
		}
	}

	if ref.ReferenceStatus == "deleted" && role != "Admin" {
		return c.Status(403).JSON(model.APIResponse{
			Status: "error",
			Error:  "Data telah dihapus",
		})
	}

	ach, err := s.Mongo.FindByHexID(context.Background(), ref.MongoID)
	if err != nil {
		return c.Status(500).JSON(model.APIResponse{
			Status: "error",
			Error:  "Gagal mengambil data MongoDB",
		})
	}

	ref.Achievement = *ach

	return c.JSON(model.APIResponse{
		Status: "success",
		Data:   ref,
	})
}

func (s *AchievementService) GetHistoryService(c *fiber.Ctx) error {
	refID := c.Params("id")
	role := getUserRole(c)

	ref, err := s.PGRepo.GetReference(refID)
	if err != nil {
		return c.Status(404).JSON(model.APIResponse{
			Status: "error",
			Error:  "Reference tidak ditemukan",
		})
	}

	switch role {

	case "Mahasiswa":
		studentID, err := getStudentID(c, s.PG)
		if err != nil || ref.StudentID != studentID {
			return c.Status(403).JSON(model.APIResponse{
				Status: "error",
				Error:  "Tidak boleh melihat history milik orang lain",
			})
		}

	case "Dosen Wali":
		userID := getUserID(c)
		refAdv, err := s.PGRepo.GetReferenceWithAdvisor(refID, userID)
		if err != nil || refAdv.AdvisorID != userID {
			return c.Status(403).JSON(model.APIResponse{
				Status: "error",
				Error:  "Anda bukan dosen wali mahasiswa ini",
			})
		}
	}

	timeline := []fiber.Map{
		{
			"status":    "draft",
			"timestamp": ref.CreatedAtRef,
			"actor":     ref.StudentID,
			"note":      nil,
		},
	}

	if ref.SubmittedAt != nil {
		timeline = append(timeline, fiber.Map{
			"status":    "submitted",
			"timestamp": ref.SubmittedAt,
			"actor":     ref.StudentID,
		})
	}

	if ref.VerifiedAt != nil {
		status := "verified"
		if ref.ReferenceStatus == "rejected" {
			status = "rejected"
		}

		timeline = append(timeline, fiber.Map{
			"status":    status,
			"timestamp": ref.VerifiedAt,
			"actor":     ref.VerifiedBy,
			"note":      ref.RejectionNote,
		})
	}

	return c.JSON(model.APIResponse{
		Status: "success",
		Data: fiber.Map{
			"reference_id": ref.ReferenceID,
			"student_id":   ref.StudentID,
			"status":       ref.ReferenceStatus,
			"timeline":     timeline,
		},
	})
}

func (s *AchievementService) UploadAttachmentsService(c *fiber.Ctx) error {
	studentID, err := getStudentID(c, s.PG)
	if err != nil {
		return c.Status(403).JSON(model.APIResponse{
			Status: "error",
			Error:  "Unauthorized",
		})
	}

	refID := c.Params("id")
	ref, err := s.PGRepo.GetReference(refID)
	if err != nil || ref.StudentID != studentID {
		return c.Status(404).JSON(model.APIResponse{
			Status: "error",
			Error:  "Reference tidak ditemukan",
		})
	}

	form, err := c.MultipartForm()
	if err != nil {
		return c.Status(400).JSON(model.APIResponse{
			Status: "error",
			Error:  "Form-data tidak valid",
		})
	}

	files := form.File["files"]
	if len(files) == 0 {
		return c.Status(400).JSON(model.APIResponse{
			Status: "error",
			Error:  "File tidak ditemukan",
		})
	}

	saveDir := "uploads/achievements/" + refID
	_ = os.MkdirAll(saveDir, os.ModePerm)

	var attachments []model.Attachment
	var savedFiles []string

	for _, file := range files {
		filename := fmt.Sprintf("%d_%s", time.Now().UnixNano(), file.Filename)
		filePath := saveDir + "/" + filename

		if err := c.SaveFile(file, filePath); err != nil {
			return c.Status(500).JSON(model.APIResponse{
				Status: "error",
				Error:  "Gagal menyimpan file",
			})
		}

		savedFiles = append(savedFiles, filePath)

		attachments = append(attachments, model.Attachment{
			FileName:   file.Filename,
			FileUrl:    filePath,
			FileType:   file.Header.Get("Content-Type"),
			UploadedAt: time.Now(),
		})
	}

	if err := s.Mongo.AddAttachments(context.Background(), ref.MongoID, attachments); err != nil {
		for _, f := range savedFiles {
			_ = os.Remove(f)
		}
		return c.Status(500).JSON(model.APIResponse{
			Status: "error",
			Error:  "Gagal menambah attachment",
		})
	}

	return c.JSON(model.APIResponse{
		Status:  "success",
		Message: "Attachments uploaded successfully",
	})
}