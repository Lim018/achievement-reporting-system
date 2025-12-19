package repository

import (
	"context"
	"database/sql"
	"fmt"
	"go-fiber/app/model"
	"math"
	"strings"
	"time"
)

type AchievementRefRepo struct {
	PG *sql.DB
	stmtGetRef    *sql.Stmt
	stmtGetRefDet *sql.Stmt
	stmtListStd   *sql.Stmt
	stmtListAdv   *sql.Stmt
}

func NewAchievementRefRepo(pg *sql.DB) *AchievementRefRepo {
	repo := &AchievementRefRepo{PG: pg}
	repo.prepareStatements()
	return repo
}

// prepare statements
func (r *AchievementRefRepo) prepareStatements() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	r.stmtGetRef, _ = r.PG.PrepareContext(ctx, `
		SELECT id, student_id, mongo_achievement_id, status,
		       submitted_at, verified_at, verified_by, rejection_note,
		       created_at, updated_at
		FROM achievement_references
		WHERE id = $1
	`)

	r.stmtGetRefDet, _ = r.PG.PrepareContext(ctx, `
		SELECT ar.id, ar.student_id, ar.mongo_achievement_id, ar.status,
		       ar.submitted_at, ar.verified_at, ar.verified_by, ar.rejection_note,
		       ar.created_at, ar.updated_at,
		       l.user_id
		FROM achievement_references ar
		JOIN students s ON ar.student_id = s.id
		LEFT JOIN lecturers l ON s.advisor_id = l.id
		WHERE ar.id = $1
	`)

	r.stmtListStd, _ = r.PG.PrepareContext(ctx, `
		SELECT id, mongo_achievement_id, status,
		       submitted_at, verified_at, verified_by, rejection_note,
		       created_at, updated_at
		FROM achievement_references
		WHERE student_id = $1 AND status != 'deleted'
		ORDER BY created_at DESC
	`)

	r.stmtListAdv, _ = r.PG.PrepareContext(ctx, `
		SELECT ar.id, ar.mongo_achievement_id, ar.status,
		       ar.submitted_at, ar.verified_at, ar.verified_by, ar.rejection_note,
		       ar.created_at, ar.updated_at
		FROM achievement_references ar
		JOIN students s ON ar.student_id = s.id
		JOIN lecturers l ON s.advisor_id = l.id
		WHERE l.user_id = $1 AND ar.status != 'deleted'
		ORDER BY ar.created_at DESC
	`)
}

func (r *AchievementRefRepo) Close() {
	if r.stmtGetRef != nil {
		r.stmtGetRef.Close()
	}
	if r.stmtGetRefDet != nil {
		r.stmtGetRefDet.Close()
	}
	if r.stmtListStd != nil {
		r.stmtListStd.Close()
	}
	if r.stmtListAdv != nil {
		r.stmtListAdv.Close()
	}
}

func (r *AchievementRefRepo) FindAllWithFilter(filter model.AchievementFilter) ([]model.AchievementDetailResponse, model.MetaPagination, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var whereClauses []string
	var args []interface{}
	argIdx := 1

	baseQuery := `
		FROM achievement_references ar
		JOIN students s ON ar.student_id = s.id
		LEFT JOIN lecturers l ON s.advisor_id = l.id
	`

	whereClauses = append(whereClauses, "ar.status != 'deleted'")

	if filter.Role == "Mahasiswa" {
		whereClauses = append(whereClauses, fmt.Sprintf("ar.student_id = $%d", argIdx))
		args = append(args, filter.StudentID)
		argIdx++
	} else if filter.Role == "Dosen Wali" {
		whereClauses = append(whereClauses, fmt.Sprintf("l.user_id = $%d", argIdx))
		args = append(args, filter.UserID)
		argIdx++
	}

	if filter.Status != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("ar.status = $%d", argIdx))
		args = append(args, filter.Status)
		argIdx++
	}

	whereString := ""
	if len(whereClauses) > 0 {
		whereString = " WHERE " + strings.Join(whereClauses, " AND ")
	}

	var totalData int64
	countQuery := "SELECT COUNT(*) " + baseQuery + whereString
	err := r.PG.QueryRowContext(ctx, countQuery, args...).Scan(&totalData)
	if err != nil {
		return nil, model.MetaPagination{}, err
	}

	sortField := "ar.created_at"
	if filter.SortBy == "updated_at" {
		sortField = "ar.updated_at"
	} else if filter.SortBy == "status" {
		sortField = "ar.status"
	}

	sortOrder := "DESC"
	if strings.ToLower(filter.SortOrder) == "asc" {
		sortOrder = "ASC"
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 10
	}
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	finalQuery := `
		SELECT ar.id, ar.student_id, ar.mongo_achievement_id, ar.status,
		       ar.submitted_at, ar.verified_at, ar.verified_by, ar.rejection_note,
		       ar.created_at, ar.updated_at, l.user_id
	` + baseQuery + whereString + fmt.Sprintf(" ORDER BY %s %s LIMIT $%d OFFSET $%d", sortField, sortOrder, argIdx, argIdx+1)

	args = append(args, limit, offset)

	rows, err := r.PG.QueryContext(ctx, finalQuery, args...)
	if err != nil {
		return nil, model.MetaPagination{}, err
	}
	defer rows.Close()

	var out []model.AchievementDetailResponse
	for rows.Next() {
		var item model.AchievementDetailResponse
		var submittedAt, verifiedAt sql.NullTime
		var verifiedBy, rejectionNote sql.NullString
		var advisorUserID sql.NullString 

		err := rows.Scan(
			&item.ReferenceID,
			&item.StudentID,
			&item.MongoID,
			&item.ReferenceStatus,
			&submittedAt,
			&verifiedAt,
			&verifiedBy,
			&rejectionNote,
			&item.CreatedAtRef,
			&item.UpdatedAtRef,
			&advisorUserID,
		)
		if err != nil {
			return nil, model.MetaPagination{}, err
		}

		if submittedAt.Valid { item.SubmittedAt = &submittedAt.Time }
		if verifiedAt.Valid { item.VerifiedAt = &verifiedAt.Time }
		if verifiedBy.Valid { s := verifiedBy.String; item.VerifiedBy = &s }
		if rejectionNote.Valid { s := rejectionNote.String; item.RejectionNote = &s }
		if advisorUserID.Valid { item.AdvisorID = advisorUserID.String }

		out = append(out, item)
	}

	totalPage := int(math.Ceil(float64(totalData) / float64(limit)))

	meta := model.MetaPagination{
		CurrentPage: page,
		TotalPage:   totalPage,
		TotalData:   totalData,
		PerPage:     limit,
	}

	return out, meta, nil
}

func (r *AchievementRefRepo) CreateReference(studentID, mongoHex string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var id string
	err := r.PG.QueryRowContext(ctx, `
		INSERT INTO achievement_references 
		(student_id, mongo_achievement_id, status, created_at, updated_at)
		VALUES ($1, $2, 'draft', NOW(), NOW())
		RETURNING id
	`, studentID, mongoHex).Scan(&id)
	return id, err
}

func (r *AchievementRefRepo) GetReference(refID string) (*model.AchievementDetailResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var out model.AchievementDetailResponse
	var submittedAt, verifiedAt sql.NullTime
	var verifiedBy, rejectionNote sql.NullString
	var mongoHex, studentID string

	var row *sql.Row
	if r.stmtGetRef != nil {
		row = r.stmtGetRef.QueryRowContext(ctx, refID)
	} else {
		row = r.PG.QueryRowContext(ctx, `
			SELECT id, student_id, mongo_achievement_id, status,
			       submitted_at, verified_at, verified_by, rejection_note,
			       created_at, updated_at
			FROM achievement_references
			WHERE id = $1
		`, refID)
	}

	err := row.Scan(
		&out.ReferenceID,
		&studentID,
		&mongoHex,
		&out.ReferenceStatus,
		&submittedAt,
		&verifiedAt,
		&verifiedBy,
		&rejectionNote,
		&out.CreatedAtRef,
		&out.UpdatedAtRef,
	)

	if err != nil {
		return nil, err
	}

	out.StudentID = studentID
	out.MongoID = mongoHex

	if submittedAt.Valid {
		out.SubmittedAt = &submittedAt.Time
	}
	if verifiedAt.Valid {
		out.VerifiedAt = &verifiedAt.Time
	}
	if verifiedBy.Valid {
		s := verifiedBy.String
		out.VerifiedBy = &s
	}
	if rejectionNote.Valid {
		s := rejectionNote.String
		out.RejectionNote = &s
	}

	return &out, nil
}

func (r *AchievementRefRepo) GetReferenceDetail(refID string) (*model.AchievementDetailResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var out model.AchievementDetailResponse
	var submittedAt, verifiedAt sql.NullTime
	var verifiedBy, rejectionNote sql.NullString
	var mongoHex, studentID, advisorUserID string

	var row *sql.Row
	if r.stmtGetRefDet != nil {
		row = r.stmtGetRefDet.QueryRowContext(ctx, refID)
	} else {
		row = r.PG.QueryRowContext(ctx, `
			SELECT ar.id, ar.student_id, ar.mongo_achievement_id, ar.status,
			       ar.submitted_at, ar.verified_at, ar.verified_by, ar.rejection_note,
			       ar.created_at, ar.updated_at,
			       l.user_id
			FROM achievement_references ar
			JOIN students s ON ar.student_id = s.id
			LEFT JOIN lecturers l ON s.advisor_id = l.id
			WHERE ar.id = $1
		`, refID)
	}

	err := row.Scan(
		&out.ReferenceID,
		&studentID,
		&mongoHex,
		&out.ReferenceStatus,
		&submittedAt,
		&verifiedAt,
		&verifiedBy,
		&rejectionNote,
		&out.CreatedAtRef,
		&out.UpdatedAtRef,
		&advisorUserID,
	)

	if err != nil {
		return nil, err
	}

	out.StudentID = studentID
	out.MongoID = mongoHex
	out.AdvisorID = advisorUserID

	if submittedAt.Valid {
		out.SubmittedAt = &submittedAt.Time
	}
	if verifiedAt.Valid {
		out.VerifiedAt = &verifiedAt.Time
	}
	if verifiedBy.Valid {
		s := verifiedBy.String
		out.VerifiedBy = &s
	}
	if rejectionNote.Valid {
		s := rejectionNote.String
		out.RejectionNote = &s
	}

	return &out, nil
}

func (r *AchievementRefRepo) GetReferenceWithAdvisor(refID, advisorUserID string) (*model.AchievementDetailResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var out model.AchievementDetailResponse
	var submittedAt, verifiedAt sql.NullTime
	var verifiedBy, rejectionNote sql.NullString
	var mongoHex, studentID, retrievedAdvisorUserID string

	err := r.PG.QueryRowContext(ctx, `
		SELECT ar.id, ar.student_id, ar.mongo_achievement_id, ar.status,
		       ar.submitted_at, ar.verified_at, ar.verified_by, ar.rejection_note,
		       ar.created_at, ar.updated_at,
		       l.user_id
		FROM achievement_references ar
		JOIN students s ON ar.student_id = s.id
		LEFT JOIN lecturers l ON s.advisor_id = l.id
		WHERE ar.id = $1 AND l.user_id = $2
	`, refID, advisorUserID).Scan(
		&out.ReferenceID,
		&studentID,
		&mongoHex,
		&out.ReferenceStatus,
		&submittedAt,
		&verifiedAt,
		&verifiedBy,
		&rejectionNote,
		&out.CreatedAtRef,
		&out.UpdatedAtRef,
		&retrievedAdvisorUserID,
	)

	if err != nil {
		return nil, err
	}

	out.StudentID = studentID
	out.MongoID = mongoHex
	out.AdvisorID = retrievedAdvisorUserID

	if submittedAt.Valid {
		out.SubmittedAt = &submittedAt.Time
	}
	if verifiedAt.Valid {
		out.VerifiedAt = &verifiedAt.Time
	}
	if verifiedBy.Valid {
		s := verifiedBy.String
		out.VerifiedBy = &s
	}
	if rejectionNote.Valid {
		s := rejectionNote.String
		out.RejectionNote = &s
	}

	return &out, nil
}

func (r *AchievementRefRepo) SubmitReference(refID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := r.PG.ExecContext(ctx, `
		UPDATE achievement_references
		SET status = 'submitted', submitted_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`, refID)
	return err
}

func (r *AchievementRefRepo) VerifyReference(refID, verifierID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := r.PG.ExecContext(ctx, `
		UPDATE achievement_references
		SET status = 'verified', verified_at = NOW(), verified_by = $1, rejection_note = NULL, updated_at = NOW()
		WHERE id = $2
	`, verifierID, refID)
	return err
}

func (r *AchievementRefRepo) RejectReference(refID, verifierID, note string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := r.PG.ExecContext(ctx, `
		UPDATE achievement_references
		SET status = 'rejected', verified_at = NOW(), verified_by = $1, rejection_note = $2, updated_at = NOW()
		WHERE id = $3
	`, verifierID, note, refID)
	return err
}

func (r *AchievementRefRepo) SoftDeleteReference(refID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := r.PG.ExecContext(ctx, `
		UPDATE achievement_references
		SET status = 'deleted', updated_at = NOW()
		WHERE id = $1
	`, refID)
	return err
}

func (r *AchievementRefRepo) ListForStudent(studentID string) ([]model.AchievementDetailResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var rows *sql.Rows
	var err error

	if r.stmtListStd != nil {
		rows, err = r.stmtListStd.QueryContext(ctx, studentID)
	} else {
		rows, err = r.PG.QueryContext(ctx, `
			SELECT id, mongo_achievement_id, status,
			       submitted_at, verified_at, verified_by, rejection_note,
			       created_at, updated_at
			FROM achievement_references
			WHERE student_id = $1 AND status != 'deleted'
			ORDER BY created_at DESC
		`, studentID)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []model.AchievementDetailResponse{}
	for rows.Next() {
		var item model.AchievementDetailResponse
		var submittedAt, verifiedAt sql.NullTime
		var verifiedBy, rejectionNote sql.NullString
		var mongoHex string

		err := rows.Scan(
			&item.ReferenceID,
			&mongoHex,
			&item.ReferenceStatus,
			&submittedAt,
			&verifiedAt,
			&verifiedBy,
			&rejectionNote,
			&item.CreatedAtRef,
			&item.UpdatedAtRef,
		)
		if err != nil {
			return nil, err
		}

		item.MongoID = mongoHex

		if submittedAt.Valid {
			item.SubmittedAt = &submittedAt.Time
		}
		if verifiedAt.Valid {
			item.VerifiedAt = &verifiedAt.Time
		}
		if verifiedBy.Valid {
			s := verifiedBy.String
			item.VerifiedBy = &s
		}
		if rejectionNote.Valid {
			s := rejectionNote.String
			item.RejectionNote = &s
		}

		out = append(out, item)
	}
	return out, nil
}

func (r *AchievementRefRepo) ListForAdvisor(advisorUserID string) ([]model.AchievementDetailResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var rows *sql.Rows
	var err error

	if r.stmtListAdv != nil {
		rows, err = r.stmtListAdv.QueryContext(ctx, advisorUserID)
	} else {
		rows, err = r.PG.QueryContext(ctx, `
			SELECT ar.id, ar.mongo_achievement_id, ar.status,
			       ar.submitted_at, ar.verified_at, ar.verified_by, ar.rejection_note,
			       ar.created_at, ar.updated_at
			FROM achievement_references ar
			JOIN students s ON ar.student_id = s.id
			JOIN lecturers l ON s.advisor_id = l.id
			WHERE l.user_id = $1 AND ar.status != 'deleted'
			ORDER BY ar.created_at DESC
		`, advisorUserID)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []model.AchievementDetailResponse{}
	for rows.Next() {
		var item model.AchievementDetailResponse
		var submittedAt, verifiedAt sql.NullTime
		var verifiedBy, rejectionNote sql.NullString
		var mongoHex string

		err := rows.Scan(
			&item.ReferenceID,
			&mongoHex,
			&item.ReferenceStatus,
			&submittedAt,
			&verifiedAt,
			&verifiedBy,
			&rejectionNote,
			&item.CreatedAtRef,
			&item.UpdatedAtRef,
		)
		if err != nil {
			return nil, err
		}

		item.MongoID = mongoHex

		if submittedAt.Valid {
			item.SubmittedAt = &submittedAt.Time
		}
		if verifiedAt.Valid {
			item.VerifiedAt = &verifiedAt.Time
		}
		if verifiedBy.Valid {
			s := verifiedBy.String
			item.VerifiedBy = &s
		}
		if rejectionNote.Valid {
			s := rejectionNote.String
			item.RejectionNote = &s
		}

		out = append(out, item)
	}

	return out, nil
}

func (r *AchievementRefRepo) ListForAdmin() ([]model.AchievementDetailResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := r.PG.QueryContext(ctx, `
		SELECT id, mongo_achievement_id, status,
		       submitted_at, verified_at, verified_by, rejection_note,
		       created_at, updated_at
		FROM achievement_references
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []model.AchievementDetailResponse{}
	for rows.Next() {
		var item model.AchievementDetailResponse
		var submittedAt, verifiedAt sql.NullTime
		var verifiedBy, rejectionNote sql.NullString
		var mongoHex string

		err := rows.Scan(
			&item.ReferenceID,
			&mongoHex,
			&item.ReferenceStatus,
			&submittedAt,
			&verifiedAt,
			&verifiedBy,
			&rejectionNote,
			&item.CreatedAtRef,
			&item.UpdatedAtRef,
		)
		if err != nil {
			return nil, err
		}

		item.MongoID = mongoHex

		if submittedAt.Valid {
			item.SubmittedAt = &submittedAt.Time
		}
		if verifiedAt.Valid {
			item.VerifiedAt = &verifiedAt.Time
		}
		if verifiedBy.Valid {
			s := verifiedBy.String
			item.VerifiedBy = &s
		}
		if rejectionNote.Valid {
			s := rejectionNote.String
			item.RejectionNote = &s
		}

		out = append(out, item)
	}

	return out, nil
}