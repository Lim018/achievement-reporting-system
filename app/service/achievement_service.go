package service

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"sync"
	"time"

	"go-fiber/app/model"
	"go-fiber/app/repository"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type AchievementCache struct {
	sync.RWMutex
	data map[string]*model.Achievement
	ttl  time.Duration
	exp  map[string]time.Time
}

func NewAchievementCache(ttl time.Duration) *AchievementCache {
	cache := &AchievementCache{
		data: make(map[string]*model.Achievement),
		exp:  make(map[string]time.Time),
		ttl:  ttl,
	}
	
	go cache.cleanup()
	return cache
}

func (c *AchievementCache) Get(key string) (*model.Achievement, bool) {
	c.RLock()
	defer c.RUnlock()
	
	if exp, exists := c.exp[key]; exists {
		if time.Now().After(exp) {
			return nil, false
		}
		if val, ok := c.data[key]; ok {
			return val, true
		}
	}
	return nil, false
}

func (c *AchievementCache) Set(key string, val *model.Achievement) {
	c.Lock()
	defer c.Unlock()
	
	c.data[key] = val
	c.exp[key] = time.Now().Add(c.ttl)
}

func (c *AchievementCache) Delete(key string) {
	c.Lock()
	defer c.Unlock()
	
	delete(c.data, key)
	delete(c.exp, key)
}

func (c *AchievementCache) cleanup() {
	ticker := time.NewTicker(c.ttl)
	defer ticker.Stop()
	
	for range ticker.C {
		c.Lock()
		now := time.Now()
		for k, exp := range c.exp {
			if now.After(exp) {
				delete(c.data, k)
				delete(c.exp, k)
			}
		}
		c.Unlock()
	}
}

type AchievementService struct {
	PGRepo  *repository.AchievementRefRepo
	Mongo   *repository.AchievementMongoRepo
	PG      *sql.DB
	MongoDB *mongo.Database
	Cache   *AchievementCache
}

func NewAchievementService(pg *sql.DB, mongoDB *mongo.Database) *AchievementService {
	return &AchievementService{
		PGRepo:  repository.NewAchievementRefRepo(pg),
		Mongo:   repository.NewAchievementMongoRepo(mongoDB),
		PG:      pg,
		MongoDB: mongoDB,
		Cache:   NewAchievementCache(5 * time.Minute),
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

func (s *AchievementService) batchFetchAchievements(ctx context.Context, mongoIDs []string) (map[string]*model.Achievement, error) {
	if len(mongoIDs) == 0 {
		return make(map[string]*model.Achievement), nil
	}

	result := make(map[string]*model.Achievement)
	var mu sync.Mutex
	var wg sync.WaitGroup
	
	uncachedIDs := []string{}
	for _, id := range mongoIDs {
		if cached, ok := s.Cache.Get(id); ok {
			result[id] = cached
		} else {
			uncachedIDs = append(uncachedIDs, id)
		}
	}

	if len(uncachedIDs) == 0 {
		return result, nil
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
	
	semaphore := make(chan struct{}, 10)
	errChan := make(chan error, len(uncachedIDs))

	for _, mongoID := range uncachedIDs {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			doc, err := s.Mongo.FindByHexIDWithProjection(ctx, id, projection)
			if err != nil {
				errChan <- err
				return
			}

			mu.Lock()
			result[id] = doc
			s.Cache.Set(id, doc)
			mu.Unlock()
		}(mongoID)
	}

	wg.Wait()
	close(errChan)

	if len(errChan) > 0 {
		return result, <-errChan
	}

	return result, nil
}

// func utama

// CreateAchievementService membuat prestasi baru
// @Summary Create achievement
// @Description Mahasiswa membuat laporan prestasi baru (Draft)
// @Tags Achievements
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body model.CreateAchievementRequest true "Achievement Data"
// @Success 200 {object} model.APIResponse{data=object{reference_id=string,mongo_id=string}}
// @Failure 403 {object} model.APIResponse
// @Failure 400 {object} model.APIResponse
// @Failure 500 {object} model.APIResponse
// @Router /api/v1/achievements [post]
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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

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

// UpdateAchievementService memperbarui prestasi
// @Summary Update achievement
// @Description Memperbarui data prestasi yang masih berstatus draft/rejected
// @Tags Achievements
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Achievement Reference ID"
// @Param request body model.UpdateAchievementRequest true "Update Data"
// @Success 200 {object} model.APIResponse
// @Failure 400 {object} model.APIResponse
// @Failure 404 {object} model.APIResponse
// @Router /api/v1/achievements/{id} [put]
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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.Mongo.UpdateByHexID(ctx, ref.MongoID, update); err != nil {
		return c.Status(500).JSON(model.APIResponse{Status: "error", Error: "Gagal update MongoDB"})
	}

	s.Cache.Delete(ref.MongoID)

	return c.JSON(model.APIResponse{Status: "success", Message: "Updated"})
}

// DeleteAchievementService menghapus prestasi
// @Summary Delete achievement
// @Description Menghapus prestasi yang masih draft
// @Tags Achievements
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Achievement Reference ID"
// @Success 200 {object} model.APIResponse
// @Failure 400 {object} model.APIResponse
// @Failure 404 {object} model.APIResponse
// @Router /api/v1/achievements/{id} [delete]
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

	s.Cache.Delete(ref.MongoID)

	return c.JSON(model.APIResponse{Status: "success", Message: "deleted"})
}

// SubmitAchievementService submit prestasi
// @Summary Submit achievement
// @Description Mengirimkan prestasi untuk diverifikasi oleh dosen wali
// @Tags Achievements
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Achievement Reference ID"
// @Success 200 {object} model.APIResponse
// @Failure 400 {object} model.APIResponse
// @Failure 404 {object} model.APIResponse
// @Router /api/v1/achievements/{id}/submit [post]
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

// VerifyAchievementService verifikasi prestasi
// @Summary Verify achievement
// @Description Dosen Wali memverifikasi prestasi mahasiswa
// @Tags Achievements
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Achievement Reference ID"
// @Param request body object{points=int} true "Points Awarded"
// @Success 200 {object} model.APIResponse
// @Failure 403 {object} model.APIResponse
// @Router /api/v1/achievements/{id}/verify [post]
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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_ = s.Mongo.UpdateByHexID(ctx, ref.MongoID, bson.M{
		"points":    body.Points,
		"updatedAt": time.Now(),
	})

	s.Cache.Delete(ref.MongoID)

	if err := s.PGRepo.VerifyReference(refID, advisorID); err != nil {
		return c.Status(500).JSON(model.APIResponse{Status: "error", Error: "Gagal verifikasi"})
	}

	return c.JSON(model.APIResponse{Status: "success", Message: "verified"})
}

// RejectAchievementService menolak prestasi
// @Summary Reject achievement
// @Description Dosen Wali menolak prestasi mahasiswa dengan catatan
// @Tags Achievements
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Achievement Reference ID"
// @Param request body object{note=string} true "Rejection Note"
// @Success 200 {object} model.APIResponse
// @Failure 403 {object} model.APIResponse
// @Router /api/v1/achievements/{id}/reject [post]
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

// ListAchievementsService melihat daftar prestasi
// @Summary List achievements
// @Description Melihat daftar prestasi dengan filter dan pagination
// @Tags Achievements
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number"
// @Param limit query int false "Limit per page"
// @Param status query string false "Filter status"
// @Param sort_by query string false "Sort by field"
// @Param sort_order query string false "Sort order (asc/desc)"
// @Success 200 {object} model.APIResponse{data=model.PaginatedAchievementResponse}
// @Failure 500 {object} model.APIResponse
// @Router /api/v1/achievements [get]
func (s *AchievementService) ListAchievementsService(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 10)
	status := c.Query("status")
	sortBy := c.Query("sort_by", "created_at")
	sortOrder := c.Query("sort_order", "desc")

	filter := model.AchievementFilter{
		Page:      page,
		Limit:     limit,
		Status:    status,
		SortBy:    sortBy,
		SortOrder: sortOrder,
	}

	role := getUserRole(c)
	userID := getUserID(c)
	filter.Role = role

	if role == "Mahasiswa" {
		studentID, err := getStudentID(c, s.PG)
		if err != nil {
			return c.Status(403).JSON(model.APIResponse{Status: "error", Error: "Unauthorized student"})
		}
		filter.StudentID = studentID
	} else if role == "Dosen Wali" {
		filter.UserID = userID 
	}
	list, meta, err := s.PGRepo.FindAllWithFilter(filter)
	if err != nil {
		return c.Status(500).JSON(model.APIResponse{Status: "error", Error: "Gagal mengambil data referensi"})
	}

	mongoIDs := make([]string, 0, len(list))
	for _, item := range list {
		mongoIDs = append(mongoIDs, item.MongoID)
	}

	achievements, err := s.batchFetchAchievements(c.Context(), mongoIDs)
	if err != nil {
		fmt.Printf("Error fetching mongo details: %v\n", err)
	}

	for i := range list {
		if ach, ok := achievements[list[i].MongoID]; ok && ach != nil {
			list[i].Achievement = *ach
		}
	}
	
	response := model.PaginatedAchievementResponse{
		Data: list,
		Meta: meta,
	}

	return c.JSON(model.APIResponse{
		Status: "success", 
		Data:   response,
	})
}

// GetAchievementDetailService melihat detail prestasi
// @Summary Get achievement detail
// @Description Melihat detail lengkap prestasi
// @Tags Achievements
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Achievement Reference ID"
// @Success 200 {object} model.APIResponse{data=model.AchievementDetailResponse}
// @Failure 404 {object} model.APIResponse
// @Router /api/v1/achievements/{id} [get]
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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if cached, ok := s.Cache.Get(ref.MongoID); ok {
		ref.Achievement = *cached
		return c.JSON(model.APIResponse{
			Status: "success",
			Data:   ref,
		})
	}

	ach, err := s.Mongo.FindByHexID(ctx, ref.MongoID)
	if err != nil {
		return c.Status(500).JSON(model.APIResponse{
			Status: "error",
			Error:  "Gagal mengambil data MongoDB",
		})
	}

	s.Cache.Set(ref.MongoID, ach)
	ref.Achievement = *ach

	return c.JSON(model.APIResponse{
		Status: "success",
		Data:   ref,
	})
}

// GetHistoryService melihat history status
// @Summary Get achievement history
// @Description Melihat riwayat perubahan status prestasi
// @Tags Achievements
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Achievement Reference ID"
// @Success 200 {object} model.APIResponse
// @Router /api/v1/achievements/{id}/history [get]
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

// UploadAttachmentsService upload lampiran
// @Summary Upload attachments
// @Description Upload file bukti prestasi
// @Tags Achievements
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param id path string true "Achievement Reference ID"
// @Param files formData file true "Files to upload"
// @Success 200 {object} model.APIResponse
// @Failure 400 {object} model.APIResponse
// @Router /api/v1/achievements/{id}/attachments [post]
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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.Mongo.AddAttachments(ctx, ref.MongoID, attachments); err != nil {
		for _, f := range savedFiles {
			_ = os.Remove(f)
		}
		return c.Status(500).JSON(model.APIResponse{
			Status: "error",
			Error:  "Gagal menambah attachment",
		})
	}

	s.Cache.Delete(ref.MongoID)

	return c.JSON(model.APIResponse{
		Status:  "success",
		Message: "Attachments uploaded successfully",
	})
}