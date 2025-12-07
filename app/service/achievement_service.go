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
	s, _ := v.(string)
	return s
}

func getUserRole(c *fiber.Ctx) string {
	v := c.Locals("role")
	if v == nil {
		return ""
	}
	s, _ := v.(string)
	return s
}

func (s *AchievementService) CreateAchievementService(c *fiber.Ctx) error {
	studentID := getUserID(c)
	if studentID == "" {
		return c.Status(401).JSON(model.APIResponse{Status: "error", Error: "Unauthorized"})
	}

	var req model.CreateAchievementRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(model.APIResponse{Status: "error", Error: "Request body tidak valid"})
	}

	now := time.Now()

	ach := model.Achievement{
		StudentID:       studentID,
		AchievementType: req.AchievementType,
		Title:           req.Title,
		Description:     req.Description,
		Tags:            req.Tags,
		Points:          req.Points,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if req.Details != nil {
		ach.Details.CustomFields = req.Details
	}

	ctx := context.Background()
	mongoHex, err := s.Mongo.Create(ctx, ach)
	if err != nil {
		return c.Status(500).JSON(model.APIResponse{Status: "error", Error: "Gagal membuat dokumen MongoDB"})
	}

	refID, err := s.PGRepo.CreateReference(studentID, mongoHex)
	if err != nil {
		_ = s.Mongo.DeleteByHexID(ctx, mongoHex)
		return c.Status(500).JSON(model.APIResponse{Status: "error", Error: "Gagal membuat reference"})
	}

	return c.JSON(model.APIResponse{
		Status: "success",
		Data: fiber.Map{
			"reference_id": refID,
			"mongo_id":     mongoHex,
		},
	})
}

func (s *AchievementService) UpdateAchievementService(c *fiber.Ctx) error {
	userID := getUserID(c)
	refID := c.Params("id")

	ref, err := s.PGRepo.GetReference(refID)
	if err != nil || ref.StudentID != userID {
		return c.Status(404).JSON(model.APIResponse{Status: "error", Error: "Reference tidak ditemukan"})
	}

	if ref.ReferenceStatus != "draft" && ref.ReferenceStatus != "rejected" {
		return c.Status(400).JSON(model.APIResponse{Status: "error", Error: "Hanya draft/rejected yang bisa update"})
	}

	var req model.UpdateAchievementRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(model.APIResponse{Status: "error", Error: "Invalid body"})
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

	err = s.Mongo.UpdateByHexID(context.Background(), ref.MongoID, update)
	if err != nil {
		return c.Status(500).JSON(model.APIResponse{Status: "error", Error: "Gagal update MongoDB"})
	}

	return c.JSON(model.APIResponse{Status: "success", Message: "Updated"})
}

func (s *AchievementService) DeleteAchievementService(c *fiber.Ctx) error {
	userID := getUserID(c)
	refID := c.Params("id")

	ref, err := s.PGRepo.GetReference(refID)
	if err != nil || ref.StudentID != userID {
		return c.Status(404).JSON(model.APIResponse{Status: "error", Error: "Reference tidak ditemukan"})
	}

	if ref.ReferenceStatus != "draft" {
		return c.Status(400).JSON(model.APIResponse{Status: "error", Error: "Hanya draft yang boleh dihapus"})
	}

	err = s.PGRepo.SoftDeleteReference(refID)
	if err != nil {
		return c.Status(500).JSON(model.APIResponse{Status: "error", Error: "Gagal menghapus"})
	}

	return c.JSON(model.APIResponse{Status: "success", Message: "deleted"})
}

func (s *AchievementService) SubmitAchievementService(c *fiber.Ctx) error {
	userID := getUserID(c)
	refID := c.Params("id")

	ref, err := s.PGRepo.GetReference(refID)
	if err != nil || ref.StudentID != userID {
		return c.Status(404).JSON(model.APIResponse{Status: "error", Error: "Reference tidak ditemukan"})
	}

	if ref.ReferenceStatus != "draft" && ref.ReferenceStatus != "rejected" {
		return c.Status(400).JSON(model.APIResponse{Status: "error", Error: "Tidak bisa disubmit"})
	}

	err = s.PGRepo.SubmitReference(refID)
	if err != nil {
		return c.Status(500).JSON(model.APIResponse{Status: "error", Error: "Gagal submit"})
	}

	return c.JSON(model.APIResponse{Status: "success", Message: "submitted"})
}

func (s *AchievementService) VerifyAchievementService(c *fiber.Ctx) error {
	advisorID := getUserID(c)
	refID := c.Params("id")

	ref, err := s.PGRepo.GetReferenceWithAdvisor(refID, advisorID)
	if err != nil || ref.AdvisorID != advisorID {
		return c.Status(403).JSON(model.APIResponse{Status: "error", Error: "Anda bukan dosen wali mahasiswa ini"})
	}

	err = s.PGRepo.VerifyReference(refID, advisorID)
	if err != nil {
		return c.Status(500).JSON(model.APIResponse{Status: "error", Error: "Gagal verifikasi"})
	}

	return c.JSON(model.APIResponse{Status: "success", Message: "verified"})
}

func (s *AchievementService) RejectAchievementService(c *fiber.Ctx) error {
	advisorID := getUserID(c)
	refID := c.Params("id")

	ref, err := s.PGRepo.GetReferenceWithAdvisor(refID, advisorID)
	if err != nil || ref.AdvisorID != advisorID {
		return c.Status(403).JSON(model.APIResponse{Status: "error", Error: "Anda bukan dosen wali mahasiswa ini"})
	}

	var body struct {
		Note string `json:"note"`
	}
	_ = c.BodyParser(&body)

	err = s.PGRepo.RejectReference(refID, advisorID, body.Note)
	if err != nil {
		return c.Status(500).JSON(model.APIResponse{Status: "error", Error: "Gagal reject"})
	}

	return c.JSON(model.APIResponse{Status: "success", Message: "rejected"})
}

func (s *AchievementService) GetAchievementDetailService(c *fiber.Ctx) error {
	refID := c.Params("id")
	role := getUserRole(c)
	userID := getUserID(c)

	var ref *model.AchievementDetailResponse
	var err error

	switch role {
	case "Admin":
		ref, err = s.PGRepo.GetReference(refID)

	case "Dosen Wali":
		ref, err = s.PGRepo.GetReferenceWithAdvisor(refID, userID)

	default:
		ref, err = s.PGRepo.GetReference(refID)
		if err == nil && ref.StudentID != userID {
			return c.Status(403).JSON(model.APIResponse{Status: "error", Error: "Tidak boleh melihat data orang lain"})
		}
	}

	if err != nil {
		return c.Status(404).JSON(model.APIResponse{Status: "error", Error: "Reference tidak ditemukan"})
	}

	if ref.ReferenceStatus == "deleted" && role != "Admin" {
		return c.Status(403).JSON(model.APIResponse{Status: "error", Error: "Data telah dihapus"})
	}

	ach, err := s.Mongo.FindByHexID(context.Background(), ref.MongoID)
	if err != nil {
		return c.Status(500).JSON(model.APIResponse{Status: "error", Error: "Gagal ambil data MongoDB"})
	}

	ref.Achievement = *ach

	return c.JSON(model.APIResponse{Status: "success", Data: ref})
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
		list, err = s.PGRepo.ListForStudent(userID)
	}

	if err != nil {
		return c.Status(500).JSON(model.APIResponse{Status: "error", Error: "Gagal mengambil data"})
	}

	for i := range list {
		doc, err := s.Mongo.FindByHexID(context.Background(), list[i].MongoID)
		if err == nil {
			list[i].Achievement = *doc
		}
	}

	return c.JSON(model.APIResponse{Status: "success", Data: list})
}

func (s *AchievementService) GetHistoryService(c *fiber.Ctx) error {
	refID := c.Params("id")
	ref, err := s.PGRepo.GetReference(refID)
	if err != nil {
		return c.Status(404).JSON(model.APIResponse{Status: "error", Error: "Not found"})
	}

	return c.JSON(model.APIResponse{Status: "success", Data: ref})
}

func (s *AchievementService) UploadAttachmentsService(c *fiber.Ctx) error {
	userID := getUserID(c)
	refID := c.Params("id")

	ref, err := s.PGRepo.GetReference(refID)
	if err != nil || ref.StudentID != userID {
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

	err = s.Mongo.AddAttachments(context.Background(), ref.MongoID, attachments)
	if err != nil {
		for _, f := range savedFiles {
			_ = os.Remove(f)
		}

		return c.Status(500).JSON(model.APIResponse{
			Status: "error",
			Error:  "Gagal menambah attachment di MongoDB (rollback file berhasil)",
		})
	}
	return c.JSON(model.APIResponse{
		Status:  "success",
		Message: "Attachments uploaded successfully",
	})
}