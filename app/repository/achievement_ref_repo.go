package repository

import (
	"database/sql"
	"go-fiber/app/model"
)

type AchievementRefRepo struct {
	PG *sql.DB
}

func NewAchievementRefRepo(pg *sql.DB) *AchievementRefRepo {
	return &AchievementRefRepo{PG: pg}
}

func (r *AchievementRefRepo) CreateReference(studentID, mongoHex string) (string, error) {
	var id string
	err := r.PG.QueryRow(`
        INSERT INTO achievement_references 
        (student_id, mongo_achievement_id, status, created_at, updated_at)
        VALUES ($1, $2, 'draft', NOW(), NOW())
        RETURNING id
    `, studentID, mongoHex).Scan(&id)
	return id, err
}

func (r *AchievementRefRepo) GetReference(refID string) (*model.AchievementDetailResponse, error) {
	var out model.AchievementDetailResponse
	var submittedAt, verifiedAt sql.NullTime
	var verifiedBy, rejectionNote sql.NullString

	var mongoHex, studentID string

	err := r.PG.QueryRow(`
        SELECT id, student_id, mongo_achievement_id, status,
               submitted_at, verified_at, verified_by, rejection_note,
               created_at, updated_at
        FROM achievement_references
        WHERE id = $1
    `, refID).Scan(
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
	var out model.AchievementDetailResponse
	var submittedAt, verifiedAt sql.NullTime
	var verifiedBy, rejectionNote sql.NullString
	var mongoHex, studentID, advisorID string

	err := r.PG.QueryRow(`
        SELECT ar.id, ar.student_id, ar.mongo_achievement_id, ar.status,
               ar.submitted_at, ar.verified_at, ar.verified_by, ar.rejection_note,
               ar.created_at, ar.updated_at,
               s.advisor_id
        FROM achievement_references ar
        JOIN students s ON ar.student_id = s.id
        WHERE ar.id = $1
    `, refID).Scan(
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
		&advisorID,
	)

	if err != nil {
		return nil, err
	}

	out.StudentID = studentID
	out.MongoID = mongoHex
	out.AdvisorID = advisorID

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

func (r *AchievementRefRepo) GetReferenceWithAdvisor(refID, advisorID string) (*model.AchievementDetailResponse, error) {
	var out model.AchievementDetailResponse
	var submittedAt, verifiedAt sql.NullTime
	var verifiedBy, rejectionNote sql.NullString

	var mongoHex, studentID, retrievedAdvisorID string

	err := r.PG.QueryRow(`
        SELECT ar.id, ar.student_id, ar.mongo_achievement_id, ar.status,
               ar.submitted_at, ar.verified_at, ar.verified_by, ar.rejection_note,
               ar.created_at, ar.updated_at,
               s.advisor_id
        FROM achievement_references ar
        JOIN students s ON ar.student_id = s.id
        WHERE ar.id = $1
    `, refID).Scan(
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
		&retrievedAdvisorID,
	)

	if err != nil {
		return nil, err
	}

	out.StudentID = studentID
	out.MongoID = mongoHex
	out.AdvisorID = retrievedAdvisorID

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
	_, err := r.PG.Exec(`
        UPDATE achievement_references
        SET status = 'submitted', submitted_at = NOW(), updated_at = NOW()
        WHERE id = $1
    `, refID)
	return err
}

func (r *AchievementRefRepo) VerifyReference(refID, verifierID string) error {
	_, err := r.PG.Exec(`
        UPDATE achievement_references
        SET status = 'verified', verified_at = NOW(), verified_by = $1, rejection_note = NULL, updated_at = NOW()
        WHERE id = $2
    `, verifierID, refID)
	return err
}

func (r *AchievementRefRepo) RejectReference(refID, verifierID, note string) error {
	_, err := r.PG.Exec(`
        UPDATE achievement_references
        SET status = 'rejected', verified_at = NOW(), verified_by = $1, rejection_note = $2, updated_at = NOW()
        WHERE id = $3
    `, verifierID, note, refID)
	return err
}

func (r *AchievementRefRepo) SoftDeleteReference(refID string) error {
	_, err := r.PG.Exec(`
        UPDATE achievement_references
        SET status = 'deleted', updated_at = NOW()
        WHERE id = $1
    `, refID)
	return err
}

func (r *AchievementRefRepo) ListForStudent(studentID string) ([]model.AchievementDetailResponse, error) {
	rows, err := r.PG.Query(`
        SELECT id, mongo_achievement_id, status,
               submitted_at, verified_at, verified_by, rejection_note,
               created_at, updated_at
        FROM achievement_references
        WHERE student_id = $1 AND status != 'deleted'
        ORDER BY created_at DESC
    `, studentID)
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

func (r *AchievementRefRepo) ListForAdvisor(advisorID string) ([]model.AchievementDetailResponse, error) {
	rows, err := r.PG.Query(`
        SELECT ar.id, ar.mongo_achievement_id, ar.status,
               ar.submitted_at, ar.verified_at, ar.verified_by, ar.rejection_note,
               ar.created_at, ar.updated_at
        FROM achievement_references ar
        JOIN students s ON ar.student_id = s.id
        WHERE s.advisor_id = $1 AND ar.status != 'deleted'
        ORDER BY ar.created_at DESC
    `, advisorID)
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
	rows, err := r.PG.Query(`
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