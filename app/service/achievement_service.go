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
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func getUserRole(c *fiber.Ctx) string {
	v := c.Locals("role")
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func (s *AchievementService) CreateAchievementService(c *fiber.Ctx) error {
	studentID := getUserID(c)
	if studentID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(model.APIResponse{Status: "error", Error: "Unauthorized"})
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
		return c.Status(500).JSON(model.APIResponse{Status: "error", Error: "Gagal menyimpan achievement ke MongoDB"})
	}

	refID, err := s.PGRepo.CreateReference(studentID, mongoHex)
	if err != nil {
		_ = s.Mongo.DeleteByHexID(ctx, mongoHex)
		return c.Status(500).JSON(model.APIResponse{Status: "error", Error: "Gagal menyimpan reference achievement"})
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
	if userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(model.APIResponse{Status: "error", Error: "Unauthorized"})
	}

	refs, err := s.PGRepo.ListForStudent(userID)
	if err != nil {
		return c.Status(500).JSON(model.APIResponse{Status: "error", Error: "Gagal mengecek reference"})
	}
	var target *model.AchievementDetailResponse
	for _, r := range refs {
		if r.ReferenceID == refID {
			target = &r
			break
		}
	}
	if target == nil {
		return c.Status(404).JSON(model.APIResponse{Status: "error", Error: "Reference tidak ditemukan atau bukan milik Anda"})
	}
	if target.ReferenceStatus != "draft" && target.ReferenceStatus != "rejected" {
		return c.Status(400).JSON(model.APIResponse{Status: "error", Error: "Hanya draft atau rejected yang bisa diupdate"})
	}

	var req model.UpdateAchievementRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(model.APIResponse{Status: "error", Error: "Request body tidak valid"})
	}

	update := bson.M{}
	if req.Title != nil {
		update["title"] = *req.Title
	}
	if req.Description != nil {
		update["description"] = *req.Description
	}
	if req.Details != nil {
		update["details"] = req.Details
	}
	if req.Tags != nil {
		update["tags"] = req.Tags
	}
	if req.Points != nil {
		update["points"] = *req.Points
	}

	if len(update) == 0 {
		return c.Status(400).JSON(model.APIResponse{Status: "error", Error: "Tidak ada field untuk diupdate"})
	}

	err = s.Mongo.UpdateByHexID(context.Background(), target.MongoID, update)
	if err != nil {
		return c.Status(500).JSON(model.APIResponse{Status: "error", Error: "Gagal mengupdate achievement di MongoDB"})
	}

	return c.JSON(model.APIResponse{Status: "success", Message: "Achievement berhasil diperbarui"})
}

func (s *AchievementService) DeleteAchievementService(c *fiber.Ctx) error {
	userID := getUserID(c)
	refID := c.Params("id")
	if userID == "" {
		return c.Status(401).JSON(model.APIResponse{Status: "error", Error: "Unauthorized"})
	}

	refs, err := s.PGRepo.ListForStudent(userID)
	if err != nil {
		return c.Status(500).JSON(model.APIResponse{Status: "error", Error: "Gagal mengecek reference"})
	}
	var target *model.AchievementDetailResponse
	for _, r := range refs {
		if r.ReferenceID == refID {
			target = &r
			break
		}
	}
	if target == nil {
		return c.Status(404).JSON(model.APIResponse{Status: "error", Error: "Reference tidak ditemukan"})
	}
	if target.ReferenceStatus != "draft" && target.ReferenceStatus != "rejected" {
		return c.Status(400).JSON(model.APIResponse{Status: "error", Error: "Hanya draft/rejected yang dapat dihapus"})
	}

	if err := s.Mongo.DeleteByHexID(context.Background(), target.MongoID); err != nil {
		return c.Status(500).JSON(model.APIResponse{Status: "error", Error: "Gagal menghapus data MongoDB"})
	}
	if err := s.PGRepo.DeleteReference(refID); err != nil {
		return c.Status(500).JSON(model.APIResponse{Status: "error", Error: "Gagal menghapus reference"})
	}

	return c.JSON(model.APIResponse{Status: "success", Message: "Achievement dihapus"})
}

func (s *AchievementService) SubmitAchievementService(c *fiber.Ctx) error {
	userID := getUserID(c)
	refID := c.Params("id")
	if userID == "" {
		return c.Status(401).JSON(model.APIResponse{Status: "error", Error: "Unauthorized"})
	}

	refs, err := s.PGRepo.ListForStudent(userID)
	if err != nil {
		return c.Status(500).JSON(model.APIResponse{Status: "error", Error: "Gagal mengecek reference"})
	}
	var target *model.AchievementDetailResponse
	for _, r := range refs {
		if r.ReferenceID == refID {
			target = &r
			break
		}
	}
	if target == nil {
		return c.Status(404).JSON(model.APIResponse{Status: "error", Error: "Reference tidak ditemukan"})
	}
	if target.ReferenceStatus != "draft" && target.ReferenceStatus != "rejected" {
		return c.Status(400).JSON(model.APIResponse{Status: "error", Error: "Hanya draft atau rejected yang dapat disubmit"})
	}

	if err := s.PGRepo.SubmitReference(refID); err != nil {
		return c.Status(500).JSON(model.APIResponse{Status: "error", Error: "Gagal submit"})
	}

	return c.JSON(model.APIResponse{Status: "success", Message: "Achievement berhasil disubmit"})
}

func (s *AchievementService) VerifyAchievementService(c *fiber.Ctx) error {
	verifierID := getUserID(c)
	if verifierID == "" || getUserRole(c) != "Dosen Wali" {
		return c.Status(403).JSON(model.APIResponse{Status: "error", Error: "Akses ditolak"})
	}
	refID := c.Params("id")

	if err := s.PGRepo.VerifyReference(refID, verifierID, ""); err != nil {
		return c.Status(500).JSON(model.APIResponse{Status: "error", Error: "Gagal verifikasi"})
	}

	return c.JSON(model.APIResponse{Status: "success", Message: "Achievement diverifikasi"})
}

func (s *AchievementService) RejectAchievementService(c *fiber.Ctx) error {
	verifierID := getUserID(c)
	if verifierID == "" || getUserRole(c) != "Dosen Wali" {
		return c.Status(403).JSON(model.APIResponse{Status: "error", Error: "Akses ditolak"})
	}
	refID := c.Params("id")

	var body struct {
		Note string `json:"note"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(model.APIResponse{Status: "error", Error: "Request body tidak valid"})
	}

	if err := s.PGRepo.RejectReference(refID, verifierID, body.Note); err != nil {
		return c.Status(500).JSON(model.APIResponse{Status: "error", Error: "Gagal reject"})
	}

	return c.JSON(model.APIResponse{Status: "success", Message: "Achievement ditolak"})
}

func (s *AchievementService) GetAchievementDetailService(c *fiber.Ctx) error {
	refID := c.Params("id")
	role := getUserRole(c)
	userID := getUserID(c)

	row := s.PG.QueryRow(`
		SELECT id, student_id, mongo_achievement_id, status,
		       submitted_at, verified_at, verified_by, rejection_note,
		       created_at, updated_at
		FROM achievement_references
		WHERE id = $1
	`, refID)

	var studentID, mongoHex, status string
	var submittedAt, verifiedAt sql.NullTime
	var verifiedBy, rejectionNote sql.NullString
	var createdAt, updatedAt time.Time

	err := row.Scan(
		&refID,
		&studentID,
		&mongoHex,
		&status,
		&submittedAt,
		&verifiedAt,
		&verifiedBy,
		&rejectionNote,
		&createdAt,
		&updatedAt,
	)

	if err != nil {
		return c.Status(404).JSON(model.APIResponse{
			Status: "error",
			Error:  "Reference tidak ditemukan",
		})
	}

	switch role {

	case "Admin":
		break

	case "Dosen Wali":
		var count int
		err := s.PG.QueryRow(`
			SELECT COUNT(*) FROM students
			WHERE id = $1 AND advisor_id = $2
		`, studentID, userID).Scan(&count)

		if err != nil || count == 0 {
			return c.Status(403).JSON(model.APIResponse{
				Status: "error",
				Error:  "Akses ditolak: bukan mahasiswa bimbingan",
			})
		}

	case "Mahasiswa":
		if studentID != userID {
			return c.Status(403).JSON(model.APIResponse{
				Status: "error",
				Error:  "Akses ditolak: bukan milik Anda",
			})
		}

	default:
		return c.Status(403).JSON(model.APIResponse{
			Status: "error",
			Error:  "Role tidak dikenali",
		})
	}

	achDoc, err := s.Mongo.FindByHexID(context.Background(), mongoHex)
	if err != nil {
		return c.Status(500).JSON(model.APIResponse{
			Status: "error",
			Error:  "Gagal mengambil data achievement dari MongoDB",
		})
	}

	resp := model.AchievementDetailResponse{
		ReferenceID:     refID,
		MongoID:         mongoHex,
		Achievement:     *achDoc,
		ReferenceStatus: status,
		CreatedAtRef:    createdAt,
		UpdatedAtRef:    updatedAt,
	}

	if submittedAt.Valid {
		resp.SubmittedAt = &submittedAt.Time
	}
	if verifiedAt.Valid {
		resp.VerifiedAt = &verifiedAt.Time
	}
	if verifiedBy.Valid {
		s := verifiedBy.String
		resp.VerifiedBy = &s
	}
	if rejectionNote.Valid {
		s := rejectionNote.String
		resp.RejectionNote = &s
	}

	return c.JSON(model.APIResponse{Status: "success", Data: resp})
}

func (s *AchievementService) ListAchievementsService(c *fiber.Ctx) error {
	role := getUserRole(c)
	userID := getUserID(c)

	switch role {
	case "Admin":
		rows, err := s.PG.Query(`
			SELECT id, mongo_achievement_id, status, submitted_at, verified_at, verified_by, rejection_note, created_at, updated_at
			FROM achievement_references
			ORDER BY created_at DESC
		`)
		if err != nil {
			return c.Status(500).JSON(model.APIResponse{Status: "error", Error: "Gagal mengambil data"})
		}
		defer rows.Close()

		var out []model.AchievementDetailResponse
		for rows.Next() {
			var item model.AchievementDetailResponse
			var submittedAt, verifiedAt sql.NullTime
			var verifiedBy, rejectionNote sql.NullString
			var mongoHex string
			err := rows.Scan(&item.ReferenceID, &mongoHex, &item.ReferenceStatus, &submittedAt, &verifiedAt, &verifiedBy, &rejectionNote, &item.CreatedAtRef, &item.UpdatedAtRef)
			if err != nil {
				return c.Status(500).JSON(model.APIResponse{Status: "error", Error: "Gagal membaca rows"})
			}
			item.MongoID = mongoHex
			if submittedAt.Valid {
				item.SubmittedAt = &submittedAt.Time
			}
			if verifiedAt.Valid {
				item.VerifiedAt = &verifiedAt.Time
			}
			if verifiedBy.Valid {
				sv := verifiedBy.String
				item.VerifiedBy = &sv
			}
			if rejectionNote.Valid {
				r := rejectionNote.String
				item.RejectionNote = &r
			}
			out = append(out, item)
		}
		return c.JSON(model.APIResponse{Status: "success", Data: out})

	case "Dosen Wali":
		rows, err := s.PG.Query(`
			SELECT ar.id, ar.mongo_achievement_id, ar.status, ar.submitted_at, ar.verified_at, ar.verified_by, ar.rejection_note, ar.created_at, ar.updated_at
			FROM achievement_references ar
			JOIN students s ON ar.student_id = s.id
			WHERE s.advisor_id = $1
			ORDER BY ar.created_at DESC
		`, userID)
		if err != nil {
			return c.Status(500).JSON(model.APIResponse{Status: "error", Error: "Gagal mengambil data"})
		}
		defer rows.Close()

		out := []model.AchievementDetailResponse{}
		for rows.Next() {
			var item model.AchievementDetailResponse
			var submittedAt, verifiedAt sql.NullTime
			var verifiedBy, rejectionNote sql.NullString
			var mongoHex string
			err := rows.Scan(&item.ReferenceID, &mongoHex, &item.ReferenceStatus, &submittedAt, &verifiedAt, &verifiedBy, &rejectionNote, &item.CreatedAtRef, &item.UpdatedAtRef)
			if err != nil {
				return c.Status(500).JSON(model.APIResponse{Status: "error", Error: "Gagal membaca rows"})
			}
			item.MongoID = mongoHex
			if submittedAt.Valid {
				item.SubmittedAt = &submittedAt.Time
			}
			if verifiedAt.Valid {
				item.VerifiedAt = &verifiedAt.Time
			}
			if verifiedBy.Valid {
				sv := verifiedBy.String
				item.VerifiedBy = &sv
			}
			if rejectionNote.Valid {
				r := rejectionNote.String
				item.RejectionNote = &r
			}
			out = append(out, item)
		}
		return c.JSON(model.APIResponse{Status: "success", Data: out})

	default:
		out, err := s.PGRepo.ListForStudent(userID)
		if err != nil {
			return c.Status(500).JSON(model.APIResponse{Status: "error", Error: "Gagal mengambil data mahasiswa"})
		}
		return c.JSON(model.APIResponse{Status: "success", Data: out})
	}
}

func (s *AchievementService) GetHistoryService(c *fiber.Ctx) error {
	refID := c.Params("id")
	row := s.PG.QueryRow(`
		SELECT status, submitted_at, verified_at, verified_by, rejection_note, created_at, updated_at
		FROM achievement_references WHERE id = $1
	`, refID)

	var status string
	var submittedAt, verifiedAt sql.NullTime
	var verifiedBy sql.NullString
	var rejectionNote sql.NullString
	var createdAt, updatedAt time.Time

	if err := row.Scan(&status, &submittedAt, &verifiedAt, &verifiedBy, &rejectionNote, &createdAt, &updatedAt); err != nil {
		return c.Status(404).JSON(model.APIResponse{Status: "error", Error: "Reference tidak ditemukan"})
	}

	history := fiber.Map{
		"status":        status,
		"submitted_at":  nil,
		"verified_at":   nil,
		"verified_by":   nil,
		"rejection_note": nil,
		"created_at":    createdAt,
		"updated_at":    updatedAt,
	}
	if submittedAt.Valid {
		history["submitted_at"] = submittedAt.Time
	}
	if verifiedAt.Valid {
		history["verified_at"] = verifiedAt.Time
	}
	if verifiedBy.Valid {
		history["verified_by"] = verifiedBy.String
	}
	if rejectionNote.Valid {
		history["rejection_note"] = rejectionNote.String
	}

	return c.JSON(model.APIResponse{Status: "success", Data: history})
}

func (s *AchievementService) UploadAttachmentsService(c *fiber.Ctx) error {
	userID := getUserID(c)
	if userID == "" {
		return c.Status(401).JSON(model.APIResponse{Status: "error", Error: "Unauthorized"})
	}
	refID := c.Params("id")

	out, err := s.PGRepo.ListForStudent(userID)
	if err != nil {
		return c.Status(500).JSON(model.APIResponse{Status: "error", Error: "Gagal mengecek reference"})
	}
	var target *model.AchievementDetailResponse
	for _, r := range out {
		if r.ReferenceID == refID {
			target = &r
			break
		}
	}
	if target == nil {
		return c.Status(404).JSON(model.APIResponse{Status: "error", Error: "Reference tidak ditemukan"})
	}

	var payload struct {
		Attachments []model.Attachment `json:"attachments" validate:"required"`
	}
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(400).JSON(model.APIResponse{Status: "error", Error: "Body tidak valid"})
	}

	if len(payload.Attachments) == 0 {
		return c.Status(400).JSON(model.APIResponse{Status: "error", Error: "Tidak ada attachment"})
	}

	if err := s.Mongo.AddAttachments(context.Background(), target.MongoID, payload.Attachments); err != nil {
		return c.Status(500).JSON(model.APIResponse{Status: "error", Error: "Gagal menambahkan attachment"})
	}

	return c.JSON(model.APIResponse{Status: "success", Message: "Attachment berhasil ditambahkan"})
}